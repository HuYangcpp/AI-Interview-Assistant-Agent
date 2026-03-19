package config

import (
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	UploadDir string
	Auth struct {
		AccessSecret string
		AccessExpire int64 // 秒
	}
	OpenAI struct {
		ApiKey      string
		Model       string
		BaseURL     string
		MaxTokens   int
		Temperature float32
	}
	VectorDB VectorDBConfig // 新增向量数据库配置
	MCP      struct {
		Endpoint string
	}
	Redis Redis
}

// 向量数据库配置
type VectorDBConfig struct {
	Host           string
	Port           int
	DBName         string
	User           string
	Password       string
	Table          string
	MaxConn        int
	EmbeddingModel string
	EmbeddingTimeoutSeconds int
	Knowledge      Knowledge
}

type Knowledge struct {
	MaxChunkSize     int
	TopK             int
	MaxContextLength int
}

type Redis struct {
	Host     string
	Port     int
	Password string
	DB       int
}
