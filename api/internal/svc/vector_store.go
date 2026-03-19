package svc

import (
	"ai-gozero-agent/api/internal/config"
	"ai-gozero-agent/api/internal/utils"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ai-gozero-agent/api/internal/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
	openai "github.com/sashabaranov/go-openai"
)

// 向量存储结构
type VectorStore struct {
	Pool             *pgxpool.Pool  // 数据库连接池
	OpenAIClient     *openai.Client // OpenAI客户端
	EmbeddingModel   string         // 向量模型名称
	EmbeddingAPIKey  string
	EmbeddingBaseURL string
	EmbeddingHTTPCLi *http.Client
	EmbeddingTimeout time.Duration
}

// 初始化向量存储
func NewVectorStore(cfg config.VectorDBConfig, openAIClient *openai.Client, apiKey, baseURL string) (*VectorStore, error) {
	// 构建连接字符串
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	// 解析配置
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	poolConfig.MaxConns = int32(cfg.MaxConn) // 设置最大连接数
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}

	// 创建连接池
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, err
	}

	if err := ensureResumeSchema(context.Background(), pool); err != nil {
		return nil, err
	}

	embeddingTimeout := 60 * time.Second
	if cfg.EmbeddingTimeoutSeconds > 0 {
		embeddingTimeout = time.Duration(cfg.EmbeddingTimeoutSeconds) * time.Second
	}

	return &VectorStore{
		Pool:             pool,
		OpenAIClient:     openAIClient,
		EmbeddingModel:   cfg.EmbeddingModel,
		EmbeddingAPIKey:  apiKey,
		EmbeddingBaseURL: baseURL,
		EmbeddingHTTPCLi: &http.Client{},
		EmbeddingTimeout: embeddingTimeout,
	}, nil
}

const defaultEmbeddingDimension = 1536

type multimodalEmbeddingRequest struct {
	Model      string                   `json:"model"`
	Input      multimodalEmbeddingInput `json:"input"`
	Parameters map[string]any           `json:"parameters,omitempty"`
}

type multimodalEmbeddingInput struct {
	Contents []map[string]string `json:"contents"`
}

type multimodalEmbeddingResponse struct {
	Output struct {
		Embeddings []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"embeddings"`
	} `json:"output"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (vs *VectorStore) multimodalEmbeddingEndpoint() string {
	base := strings.TrimRight(vs.EmbeddingBaseURL, "/")
	base = strings.TrimSuffix(base, "/compatible-mode/v1")
	if base == "" {
		base = "https://dashscope.aliyuncs.com"
	}
	return base + "/api/v1/services/embeddings/multimodal-embedding/multimodal-embedding"
}

func (vs *VectorStore) generateQwenVLTextEmbedding(ctx context.Context, text string) ([]float32, error) {
	reqBody := multimodalEmbeddingRequest{
		Model: vs.EmbeddingModel,
		Input: multimodalEmbeddingInput{
			Contents: []map[string]string{
				{"text": text},
			},
		},
		Parameters: map[string]any{
			"dimension":   defaultEmbeddingDimension,
			"output_type": "dense",
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化多模态向量请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, vs.multimodalEmbeddingEndpoint(), bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("创建多模态向量请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+vs.EmbeddingAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := vs.EmbeddingHTTPCLi.Do(req)
	if err != nil {
		return nil, fmt.Errorf("多模态向量接口请求失败: %w", err)
	}
	defer resp.Body.Close()

	var result multimodalEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析多模态向量响应失败: %w", err)
	}

	if resp.StatusCode >= 400 {
		if result.Message != "" {
			return nil, fmt.Errorf("多模态向量接口错误: %s", result.Message)
		}
		return nil, fmt.Errorf("多模态向量接口错误: status=%s", resp.Status)
	}

	if result.Code != "" {
		return nil, fmt.Errorf("多模态向量接口错误[%s]: %s", result.Code, result.Message)
	}

	if len(result.Output.Embeddings) == 0 || len(result.Output.Embeddings[0].Embedding) == 0 {
		return nil, errors.New("未返回多模态嵌入数据")
	}

	return result.Output.Embeddings[0].Embedding, nil
}

// 保存消息到向量数据库
func (vs *VectorStore) SaveMessage(ctx context.Context, chatId, role, content string) error {
	// 生成文本向量
	embedding, err := vs.generateEmbedding(content)
	if err != nil {
		return fmt.Errorf("生成嵌入失败: %w", err)
	}

	// 添加source_type字段
	sql := `INSERT INTO vector_store (chat_id, role, content, embedding, source_type)
	            VALUES ($1, $2, $3, $4, 'message')`
	_, err = vs.Pool.Exec(ctx, sql,
		chatId, role, content, pgvector.NewVector(embedding))

	return err
}

// 获取会话历史消息
func (vs *VectorStore) GetMessages(ctx context.Context, chatId string, limit int) ([]types.VectorMessage, error) {
	// 查询数据库
	sql := `SELECT id, role, content, created_at
	            FROM vector_store
	            WHERE chat_id = $1 AND source_type = 'message'
	            ORDER BY created_at DESC, id DESC
	            LIMIT $2`

	rows, err := vs.Pool.Query(ctx, sql, chatId, limit)
	if err != nil {
		return nil, fmt.Errorf("数据库查询失败: %w", err)
	}
	defer rows.Close()

	// 处理查询结果
	var messages []types.VectorMessage
	for rows.Next() {
		var id int64
		var role, content string
		var createdAt time.Time
		if err := rows.Scan(&id, &role, &content, &createdAt); err != nil {
			return nil, fmt.Errorf("行扫描失败: %w", err)
		}
		messages = append(messages, types.VectorMessage{
			ID:        id,
			Role:      role,
			Content:   content,
			CreatedAt: createdAt.Format(time.RFC3339),
		})
	}

	// 反转消息顺序（最新消息在最后）
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// CountMessages 统计指定 chatId 的消息数量
func (vs *VectorStore) CountMessages(ctx context.Context, chatID string) (int, error) {
	sql := `SELECT COUNT(*) FROM vector_store WHERE chat_id = $1 AND source_type = 'message'`
	var count int
	err := vs.Pool.QueryRow(ctx, sql, chatID).Scan(&count)
	return count, err
}

// 生成文本向量
func (vs *VectorStore) generateEmbedding(text string) ([]float32, error) {
	if text == "" {
		return make([]float32, defaultEmbeddingDimension), nil
	}

	timeout := vs.EmbeddingTimeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if vs.EmbeddingModel == "qwen3-vl-embedding" {
		return vs.generateQwenVLTextEmbedding(ctx, text)
	}

	// 调用OpenAI Embedding API
	resp, err := vs.OpenAIClient.CreateEmbeddings(ctx,
		openai.EmbeddingRequest{
			Input: []string{text},
			Model: openai.EmbeddingModel(vs.EmbeddingModel),
		})

	if err != nil {
		return nil, fmt.Errorf("OpenAI API错误: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, errors.New("未返回嵌入数据")
	}

	return resp.Data[0].Embedding, nil
}

func ensureResumeSchema(ctx context.Context, pool *pgxpool.Pool) error {
	statements := []string{
		`ALTER TABLE interview_sessions ADD COLUMN IF NOT EXISTS custom_prompt TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE students ADD COLUMN IF NOT EXISTS resume_mode VARCHAR(20) NOT NULL DEFAULT 'short'`,
		`ALTER TABLE students ADD COLUMN IF NOT EXISTS resume_length INT NOT NULL DEFAULT 0`,
		`ALTER TABLE students ADD COLUMN IF NOT EXISTS resume_summary_json TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE students ADD COLUMN IF NOT EXISTS resume_summary_text TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE employments ADD COLUMN IF NOT EXISTS verification_status VARCHAR(20) NOT NULL DEFAULT 'pending'`,
		`ALTER TABLE employments ADD COLUMN IF NOT EXISTS review_comment TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE employments ADD COLUMN IF NOT EXISTS reviewer_id BIGINT REFERENCES users(id) ON DELETE SET NULL`,
		`ALTER TABLE employments ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ`,
		`CREATE INDEX IF NOT EXISTS idx_employments_verification_status ON employments(verification_status)`,
		`CREATE TABLE IF NOT EXISTS employment_evidences (
			id BIGSERIAL PRIMARY KEY,
			employment_id BIGINT NOT NULL REFERENCES employments(id) ON DELETE CASCADE,
			file_url VARCHAR(500) NOT NULL,
			file_name VARCHAR(255) NOT NULL,
			mime_type VARCHAR(100) NOT NULL DEFAULT '',
			uploaded_by BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_employment_evidences_employment_id ON employment_evidences(employment_id)`,
		`CREATE TABLE IF NOT EXISTS resume_chunks (
			id BIGSERIAL PRIMARY KEY,
			student_id BIGINT NOT NULL REFERENCES students(id) ON DELETE CASCADE,
			chunk_index INT NOT NULL,
			content TEXT NOT NULL,
			embedding vector(1536) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_resume_chunks_student_id ON resume_chunks(student_id)`,
		`CREATE INDEX IF NOT EXISTS idx_resume_chunks_embedding_hnsw
			ON resume_chunks USING hnsw (embedding vector_cosine_ops)
			WITH (m = 16, ef_construction = 64)`,
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = 'public'
				  AND table_name = 'interview_analyses'
				  AND column_name = 'student_id'
			) THEN
				ALTER TABLE interview_analyses ADD COLUMN student_id BIGINT;
			END IF;
		END $$;`,
		`UPDATE interview_analyses ia
		SET student_id = s.student_id
		FROM interview_sessions s
		WHERE ia.session_id = s.id
		  AND ia.student_id IS NULL`,
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conname = 'interview_analyses_student_id_fkey'
			) THEN
				ALTER TABLE interview_analyses
				ADD CONSTRAINT interview_analyses_student_id_fkey
				FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE;
			END IF;
		END $$;`,
		`DO $$
		BEGIN
			IF EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = 'public'
				  AND table_name = 'interview_analyses'
				  AND column_name = 'student_id'
				  AND is_nullable = 'YES'
			)
			AND NOT EXISTS (
				SELECT 1 FROM interview_analyses WHERE student_id IS NULL
			) THEN
				ALTER TABLE interview_analyses ALTER COLUMN student_id SET NOT NULL;
			END IF;
		END $$;`,
		`CREATE INDEX IF NOT EXISTS idx_interview_analyses_student_id ON interview_analyses(student_id)`,
		`ALTER TABLE personalized_suggestions ADD COLUMN IF NOT EXISTS session_id BIGINT REFERENCES interview_sessions(id) ON DELETE CASCADE`,
		`CREATE INDEX IF NOT EXISTS idx_suggestions_session_id ON personalized_suggestions(session_id)`,
		`DELETE FROM personalized_suggestions WHERE session_id IS NULL`,
	}

	for _, stmt := range statements {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("初始化简历增强 schema 失败: %w", err)
		}
	}

	return nil
}

// 添加知识库保存方法
func (vs *VectorStore) SaveKnowledge(title, content string, cfg config.VectorDBConfig, splitter ...utils.Splitter) error {
	var sp utils.Splitter
	if len(splitter) > 0 && splitter[0] != nil {
		sp = splitter[0]
	} else {
		sp = &utils.FixedSizeSplitter{ChunkSize: cfg.Knowledge.MaxChunkSize}
	}

	chunks := sp.Split(content)
	for _, chunk := range chunks {
		embedding, err := vs.generateEmbedding(chunk)
		if err != nil {
			return fmt.Errorf("生成嵌入失败: %w", err)
		}

		sql := `INSERT INTO knowledge_base (title, content, embedding)
	             VALUES ($1, $2, $3)`
		_, err = vs.Pool.Exec(context.Background(), sql, title, chunk, pgvector.NewVector(embedding))
		if err != nil {
			return err
		}
	}

	return nil
}

// 添加知识检索方法
func (vs *VectorStore) RetrieveKnowledge(query string, topK int) ([]types.KnowledgeChunk, error) {
	queryEmbedding, err := vs.generateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("生成查询嵌入失败: %w", err)
	}

	// 使用余弦相似度检索
	sql := `SELECT id, title, content, 1 - (embedding <=> $1::vector) AS similarity
	          FROM knowledge_base
	          ORDER BY embedding <=> $1::vector
	          LIMIT $2`

	rows, err := vs.Pool.Query(context.Background(), sql, pgvector.NewVector(queryEmbedding), topK)
	if err != nil {
		return nil, fmt.Errorf("知识检索失败: %w", err)
	}
	defer rows.Close()

	var results []types.KnowledgeChunk
	for rows.Next() {
		var id int64
		var title, content string
		var similarity float64
		if err := rows.Scan(&id, &title, &content, &similarity); err != nil {
			return nil, fmt.Errorf("扫描结果失败: %w", err)
		}
		results = append(results, types.KnowledgeChunk{
			ID:         id,
			Title:      title,
			Content:    content,
			Similarity: similarity,
		})
	}

	return results, nil
}

func (vs *VectorStore) ReplaceResumeChunks(studentID int64, chunks []string) error {
	tx, err := vs.Pool.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	if _, err := tx.Exec(context.Background(), `DELETE FROM resume_chunks WHERE student_id = $1`, studentID); err != nil {
		return err
	}

	for idx, chunk := range chunks {
		embedding, err := vs.generateEmbedding(chunk)
		if err != nil {
			return fmt.Errorf("生成简历分块嵌入失败: %w", err)
		}
		if _, err := tx.Exec(
			context.Background(),
			`INSERT INTO resume_chunks (student_id, chunk_index, content, embedding) VALUES ($1, $2, $3, $4)`,
			studentID,
			idx,
			chunk,
			pgvector.NewVector(embedding),
		); err != nil {
			return err
		}
	}

	return tx.Commit(context.Background())
}

func (vs *VectorStore) RetrieveResumeChunks(studentID int64, query string, topK int) ([]types.ResumeChunk, error) {
	queryEmbedding, err := vs.generateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("生成简历查询嵌入失败: %w", err)
	}

	sql := `SELECT id, student_id, chunk_index, content, 1 - (embedding <=> $1::vector) AS similarity
		FROM resume_chunks
		WHERE student_id = $2
		ORDER BY embedding <=> $1::vector
		LIMIT $3`

	rows, err := vs.Pool.Query(context.Background(), sql, pgvector.NewVector(queryEmbedding), studentID, topK)
	if err != nil {
		return nil, fmt.Errorf("简历片段检索失败: %w", err)
	}
	defer rows.Close()

	var results []types.ResumeChunk
	for rows.Next() {
		var chunk types.ResumeChunk
		if err := rows.Scan(&chunk.ID, &chunk.StudentID, &chunk.ChunkIndex, &chunk.Content, &chunk.Similarity); err != nil {
			return nil, fmt.Errorf("扫描简历片段失败: %w", err)
		}
		results = append(results, chunk)
	}

	return results, nil
}

// 测试数据库连接
func (vs *VectorStore) TestConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return vs.Pool.Ping(ctx)
}
