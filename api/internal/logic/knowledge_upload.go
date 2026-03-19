package logic

import (
	"context"
	"fmt"
	"strings"

	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"
	"ai-gozero-agent/api/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxKnowledgeContentSize = 100 * 1024 // 100KB

type KnowledgeUploadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewKnowledgeUploadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *KnowledgeUploadLogic {
	return &KnowledgeUploadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *KnowledgeUploadLogic) KnowledgeUpload(req *types.KnowledgeUploadReq) (*types.KnowledgeUploadResp, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("标题不能为空")
	}
	if len(req.Content) > maxKnowledgeContentSize {
		return nil, fmt.Errorf("内容过长，最大支持 %dKB", maxKnowledgeContentSize/1024)
	}

	var splitter utils.Splitter
	switch strings.TrimSpace(req.SplitterType) {
	case "markdown":
		splitter = &utils.MarkdownSplitter{
			FallbackChunkSize: l.svcCtx.Config.VectorDB.Knowledge.MaxChunkSize,
		}
	case "agentic":
		splitter = &utils.AgenticSplitter{
			Client: l.svcCtx.OpenAIClient,
			Model:  "qwen3.5-122b-a10b",
		}
	default:
		splitter = &utils.FixedSizeSplitter{
			ChunkSize: l.svcCtx.Config.VectorDB.Knowledge.MaxChunkSize,
		}
	}

	if err := l.svcCtx.VectorStore.SaveKnowledge(req.Title, req.Content, l.svcCtx.Config.VectorDB, splitter); err != nil {
		logx.Errorf("保存知识失败: %v", err)
		return nil, err
	}

	chunks := splitter.Split(req.Content)
	return &types.KnowledgeUploadResp{
		Msg:    "知识上传成功",
		Chunks: len(chunks),
	}, nil
}
