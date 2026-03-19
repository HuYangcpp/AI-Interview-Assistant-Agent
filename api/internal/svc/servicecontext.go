package svc

import (
	"ai-gozero-agent/api/internal/config"
	"ai-gozero-agent/api/internal/model"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	openai "github.com/sashabaranov/go-openai"
	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config       config.Config
	OpenAIClient *openai.Client
	VectorStore  *VectorStore // 替换SessionStore
	DB           *pgxpool.Pool
	PdfClient    *PdfClient
	Redis        *redis.Client

	UserModel               *model.UserModel
	StudentModel            *model.StudentModel
	EmploymentModel         *model.EmploymentModel
	EmploymentEvidenceModel *model.EmploymentEvidenceModel
	InterviewSessionModel   *model.InterviewSessionModel
	InterviewAnalysisModel  *model.InterviewAnalysisModel
	SuggestionModel         *model.SuggestionModel
	StatsModel              *model.StatsModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 创建OpenAI客户端
	openaiConfig := openai.DefaultConfig(c.OpenAI.ApiKey)
	openaiConfig.BaseURL = c.OpenAI.BaseURL
	openAIClient := openai.NewClientWithConfig(openaiConfig)

	// 初始化Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port),
		Password: c.Redis.Password,
		DB:       c.Redis.DB,
	})

	// 测试Redis连接
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		logx.Must(fmt.Errorf("Redis连接失败: %w", err))
	}
	logx.Info("Redis连接成功")

	// 初始化向量存储
	vectorStore, err := NewVectorStore(c.VectorDB, openAIClient, c.OpenAI.ApiKey, c.OpenAI.BaseURL)
	if err != nil {
		logx.Must(fmt.Errorf("初始化向量数据库失败: %w", err))
	}

	// 测试数据库连接
	if err := vectorStore.TestConnection(); err != nil {
		logx.Must(fmt.Errorf("向量数据库连接失败: %w", err))
	}
	logx.Info("向量数据库连接成功")

	return &ServiceContext{
		Config:       c,
		OpenAIClient: openAIClient,
		VectorStore:  vectorStore,
		DB:           vectorStore.Pool,
		PdfClient:    NewPdfClient(c.MCP.Endpoint),
		Redis:        rdb,

		UserModel:               model.NewUserModel(vectorStore.Pool),
		StudentModel:            model.NewStudentModel(vectorStore.Pool),
		EmploymentModel:         model.NewEmploymentModel(vectorStore.Pool),
		EmploymentEvidenceModel: model.NewEmploymentEvidenceModel(vectorStore.Pool),
		InterviewSessionModel:   model.NewInterviewSessionModel(vectorStore.Pool),
		InterviewAnalysisModel:  model.NewInterviewAnalysisModel(vectorStore.Pool),
		SuggestionModel:         model.NewSuggestionModel(vectorStore.Pool),
		StatsModel:              model.NewStatsModel(vectorStore.Pool),
	}
}
