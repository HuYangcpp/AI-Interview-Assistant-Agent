package logic

import (
	"context"
	"errors"
	"strings"
	"time"

	"ai-gozero-agent/api/internal/model"
	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	autoStartMessage     = "你好，我准备好开始面试了。"
	autoStartTimeout     = 45 * time.Second
	autoStartMaxAttempts = 2
	autoStartRetryDelay  = 1500 * time.Millisecond
)

type InterviewLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewInterviewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InterviewLogic {
	return &InterviewLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *InterviewLogic) CreateSession(userId int64, req *types.CreateInterviewSessionReq) (*types.CreateInterviewSessionResp, error) {
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return nil, err
	}

	it := strings.TrimSpace(req.InterviewType)
	if it == "" {
		it = "general"
	}
	if !isAllowedInterviewType(it) {
		return nil, errors.New("invalid interviewType")
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "AI 模拟面试"
	}

	customPrompt := strings.TrimSpace(req.CustomPrompt)
	if it == "custom" && customPrompt == "" {
		// 根据简历或求职意向生成自定义提示词
		customPrompt = l.buildCustomPrompt(s)
	}

	chatID := uuid.NewString()
	id, err := l.svcCtx.InterviewSessionModel.Insert(l.ctx, &model.InterviewSession{
		ChatID:        chatID,
		StudentID:     s.ID,
		Title:         title,
		InterviewType: it,
		CustomPrompt:  customPrompt,
	})
	if err != nil {
		return nil, err
	}

	session, err := l.svcCtx.InterviewSessionModel.FindById(l.ctx, id)
	if err != nil {
		return nil, err
	}

	// 异步触发 AI 主动开场
	go func(sess *model.InterviewSession) {
		l.autoStartSession(sess)
	}(session)

	return &types.CreateInterviewSessionResp{
		Session: toSessionItem(session),
	}, nil
}

func (l *InterviewLogic) autoStartSession(session *model.InterviewSession) {
	for attempt := 1; attempt <= autoStartMaxAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), autoStartTimeout)
		chatLogic := NewChatLogic(ctx, l.svcCtx)
		ch, err := chatLogic.AutoStart(session, autoStartMessage)
		if err != nil {
			cancel()
			l.Logger.Errorf("自动开场启动失败: sessionId=%d attempt=%d err=%v", session.ID, attempt, err)
			if attempt < autoStartMaxAttempts {
				time.Sleep(autoStartRetryDelay)
			}
			continue
		}

		for range ch {
			// 消费 channel，消息会在 Chat 内部保存
		}
		cancel()

		hasAssistant, err := l.hasAssistantMessage(session.ChatID)
		if err != nil {
			l.Logger.Errorf("检查自动开场结果失败: sessionId=%d attempt=%d err=%v", session.ID, attempt, err)
		}
		if hasAssistant {
			if attempt > 1 {
				l.Logger.Infof("自动开场重试成功: sessionId=%d attempt=%d", session.ID, attempt)
			}
			return
		}

		l.Logger.Errorf("自动开场未生成面试官消息: sessionId=%d attempt=%d", session.ID, attempt)
		if attempt < autoStartMaxAttempts {
			time.Sleep(autoStartRetryDelay)
		}
	}
}

func (l *InterviewLogic) hasAssistantMessage(chatID string) (bool, error) {
	const query = `SELECT COUNT(1) FROM vector_store WHERE chat_id = $1 AND source_type = 'message' AND role = 'assistant'`

	var count int
	if err := l.svcCtx.DB.QueryRow(context.Background(), query, chatID).Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

func (l *InterviewLogic) buildCustomPrompt(student *model.Student) string {
	if student.ResumeSummaryText != "" {
		return "你是" + student.Major + "专业面试官，候选人简历摘要：\n" +
			student.ResumeSummaryText +
			"\n请根据简历内容定制化提问，考察候选人与背景相关的技术深度。"
	}
	if student.Skills != "" && student.Skills != "[]" {
		return "你是技术面试官，候选人技能标签：" + student.Skills +
			"，专业：" + student.Major +
			"。请根据这些技能方向提问，考察候选人的技术深度和项目经验。"
	}
	return "你是资深技术面试官，请根据候选人的专业背景（" + student.Major + "）进行综合技术面试，覆盖基础、进阶和项目经验三个层次。"
}

func (l *InterviewLogic) ListSessions(userId int64) ([]*types.InterviewSessionItem, error) {
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return nil, err
	}
	list, err := l.svcCtx.InterviewSessionModel.FindByStudentId(l.ctx, s.ID, 50)
	if err != nil {
		return nil, err
	}

	var resp []*types.InterviewSessionItem
	for _, ss := range list {
		item := toSessionItem(ss)
		if count, err := l.svcCtx.VectorStore.CountMessages(l.ctx, ss.ChatID); err == nil {
			item.MessageCount = count
		}
		resp = append(resp, &item)
	}
	return resp, nil
}

func (l *InterviewLogic) GetSession(userId, sessionId int64) (*types.InterviewSessionDetailResp, error) {
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return nil, err
	}
	session, err := l.svcCtx.InterviewSessionModel.FindById(l.ctx, sessionId)
	if err != nil {
		return nil, err
	}
	if session.StudentID != s.ID {
		return nil, errors.New("forbidden")
	}

	item := toSessionItem(session)
	detail := &types.InterviewSessionDetailResp{
		InterviewSessionItem: item,
		ChatID:               session.ChatID,
		AISummary:            session.AISummary,
	}

	return detail, nil
}

func (l *InterviewLogic) DeleteSession(userId, sessionId int64) error {
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return err
	}
	// 先删除关联的分析记录，再删除会话，避免 FK 约束错误
	_ = l.svcCtx.InterviewAnalysisModel.DeleteBySessionId(l.ctx, sessionId)
	return l.svcCtx.InterviewSessionModel.DeleteByIdAndStudentId(l.ctx, sessionId, s.ID)
}

func (l *InterviewLogic) EndSession(userId, sessionId int64) error {
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return err
	}
	session, err := l.svcCtx.InterviewSessionModel.FindById(l.ctx, sessionId)
	if err != nil {
		return err
	}
	if session.StudentID != s.ID {
		return errors.New("forbidden")
	}

	duration := int(time.Since(session.CreatedAt).Seconds())
	if duration < 0 {
		duration = 0
	}
	if err := l.svcCtx.InterviewSessionModel.MarkCompleted(l.ctx, sessionId, duration); err != nil && !errors.Is(err, model.ErrNotFound) {
		return err
	}

	// 触发评分/分析（传入 studentId）
	return NewAnalysisLogic(l.ctx, l.svcCtx).GenerateAnalysis(userId, sessionId)
}

func toSessionItem(s *model.InterviewSession) types.InterviewSessionItem {
	item := types.InterviewSessionItem{
		ID:              s.ID,
		Title:           s.Title,
		InterviewType:   s.InterviewType,
		Status:          s.Status,
		TotalQuestions:  s.TotalQuestions,
		DurationSeconds: s.DurationSeconds,
		Score:           s.Score,
		CreatedAt:       s.CreatedAt.Format(time.RFC3339),
	}
	if s.CompletedAt != nil {
		item.CompletedAt = s.CompletedAt.Format(time.RFC3339)
	}
	return item
}

func isAllowedInterviewType(t string) bool {
	switch t {
	case "general", "go", "java", "frontend", "system_design", "custom":
		return true
	default:
		return false
	}
}
