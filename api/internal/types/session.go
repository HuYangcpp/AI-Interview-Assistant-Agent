package types

import (
	"context"

	"ai-gozero-agent/api/internal/config"
	"ai-gozero-agent/api/internal/utils"
)

// 新增向量存储消息结构
type VectorMessage struct {
	ID        int64  `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

// 新增知识块结构
type KnowledgeChunk struct {
	ID         int64   `json:"id"`         // 知识块ID
	Title      string  `json:"title"`      // 知识标题
	Content    string  `json:"content"`    // 知识内容
	Similarity float64 `json:"similarity"` // 余弦相似度
}

type ResumeChunk struct {
	ID         int64   `json:"id"`
	StudentID  int64   `json:"studentId"`
	ChunkIndex int     `json:"chunkIndex"`
	Content    string  `json:"content"`
	Similarity float64 `json:"similarity"`
}

// 会话存储接口更新
type SessionStore interface {
	GetMessages(ctx context.Context, chatId string, limit int) ([]VectorMessage, error)               // 获取消息历史
	SaveMessage(ctx context.Context, chatId, role, content string) error                              // 保存单条消息
	SaveKnowledge(title, content string, cfg config.VectorDBConfig, splitter ...utils.Splitter) error // 保存知识库
	RetrieveKnowledge(query string, topK int) ([]KnowledgeChunk, error)                               // 检索知识库
}
