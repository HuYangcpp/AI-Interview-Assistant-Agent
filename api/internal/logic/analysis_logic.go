package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ai-gozero-agent/api/internal/model"
	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"
	"ai-gozero-agent/api/internal/utils"

	openai "github.com/sashabaranov/go-openai"
	"github.com/zeromicro/go-zero/core/logx"
)

type AnalysisLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAnalysisLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AnalysisLogic {
	return &AnalysisLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

type analysisAIResp struct {
	OverallScore    float64  `json:"overall_score"`
	TechnicalScore  float64  `json:"technical_score"`
	ExpressionScore float64  `json:"expression_score"`
	LogicScore      float64  `json:"logic_score"`
	Strengths       []string `json:"strengths"`
	Weaknesses      []string `json:"weaknesses"`
	Suggestions     string   `json:"suggestions"`
	DetailReport    string   `json:"detail_report"`
}

func (l *AnalysisLogic) GenerateAnalysis(userId, sessionId int64) error {
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return err
	}
	studentId := s.ID

	session, err := l.svcCtx.InterviewSessionModel.FindById(l.ctx, sessionId)
	if err != nil {
		return err
	}
	if session.StudentID != studentId {
		return errors.New("forbidden")
	}

	messages, err := l.fetchSessionMessages(session.ChatID, 200)
	if err != nil {
		return err
	}
	if len(messages) == 0 {
		return errors.New("no messages to analyze")
	}

	prompt := buildAnalysisPrompt(messages)
	req := openai.ChatCompletionRequest{
		Model: l.svcCtx.Config.OpenAI.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: "你是一个严谨的面试评测官。你必须只输出一个 JSON 对象，不要输出任何额外文字、解释、Markdown 或代码块。" +
					"JSON 必须包含字段：" +
					"overall_score, technical_score, expression_score, logic_score(0-100 数字), strengths(字符串数组), weaknesses(字符串数组), suggestions(字符串), detail_report(Markdown 字符串)。",
			},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		Stream:      false,
		MaxTokens:   l.svcCtx.Config.OpenAI.MaxTokens,
		Temperature: 0.2,
	}

	resp, err := l.svcCtx.OpenAIClient.CreateChatCompletion(l.ctx, req)
	if err != nil {
		return err
	}
	if len(resp.Choices) == 0 {
		return errors.New("empty ai response")
	}

	content := resp.Choices[0].Message.Content
	jsonText := extractJSONObject(content)
	if jsonText == "" || !json.Valid([]byte(jsonText)) {
		logx.Errorf("AI raw response: %s", content)
		return errors.New("invalid ai json, please retry")
	}

	var ai analysisAIResp
	if err := json.Unmarshal([]byte(jsonText), &ai); err != nil {
		logx.Errorf("AI raw response: %s", content)
		return errors.New("invalid ai json, please retry")
	}

	strengthsJSON, err := json.Marshal(ai.Strengths)
	if err != nil {
		return fmt.Errorf("序列化优势失败: %w", err)
	}
	weaknessesJSON, err := json.Marshal(ai.Weaknesses)
	if err != nil {
		return fmt.Errorf("序列化劣势失败: %w", err)
	}

	_, err = l.svcCtx.InterviewAnalysisModel.Insert(l.ctx, &model.InterviewAnalysis{
		SessionID:       session.ID,
		StudentID:       studentId,
		OverallScore:    ai.OverallScore,
		TechnicalScore:  ai.TechnicalScore,
		ExpressionScore: ai.ExpressionScore,
		LogicScore:      ai.LogicScore,
		Strengths:       string(strengthsJSON),
		Weaknesses:      string(weaknessesJSON),
		Suggestions:     ai.Suggestions,
		DetailReport:    ai.DetailReport,
	})
	if err != nil {
		return err
	}

	scoreDetail, err := json.Marshal(map[string]any{
		"technical_score":  ai.TechnicalScore,
		"expression_score": ai.ExpressionScore,
		"logic_score":      ai.LogicScore,
	})
	if err != nil {
		return fmt.Errorf("序列化评分详情失败: %w", err)
	}
	if err := l.svcCtx.InterviewSessionModel.UpdateScore(l.ctx, session.ID, ai.OverallScore, scoreDetail, utils.TruncateText(ai.Suggestions, 500)); err != nil {
		l.Logger.Errorf("更新会话评分失败: %v", err)
	}

	// 自动触发个性化建议（异步）
	go func(studentID, sessionID int64) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := NewSuggestionLogic(ctx, l.svcCtx).GenerateSuggestions(studentID, sessionID); err != nil {
			logx.Errorf("GenerateSuggestions failed: %v", err)
			if _, insertErr := l.svcCtx.SuggestionModel.Insert(ctx, &model.PersonalizedSuggestion{
				StudentID:      studentID,
				SessionID:      &sessionID,
				SuggestionType: "system",
				Content:        "建议生成失败，请手动重试",
				IsRead:         false,
			}); insertErr != nil {
				logx.Errorf("Insert system suggestion failed: %v", insertErr)
			}
		}
	}(studentId, session.ID)

	return nil
}

func (l *AnalysisLogic) GetSessionAnalysis(userId, sessionId int64) (*types.AnalysisDetailResp, error) {
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return nil, err
	}
	studentId := s.ID

	session, err := l.svcCtx.InterviewSessionModel.FindById(l.ctx, sessionId)
	if err != nil {
		return nil, err
	}
	if session.StudentID != studentId {
		return nil, errors.New("forbidden")
	}

	a, err := l.svcCtx.InterviewAnalysisModel.FindLatestBySessionId(l.ctx, sessionId)
	if err != nil {
		return nil, err
	}

	suggestions, err := l.svcCtx.SuggestionModel.FindByStudentIdAndSessionId(l.ctx, studentId, sessionId)
	if err != nil {
		return nil, err
	}

	var sessionSuggestions []*types.SuggestionItem
	for _, sg := range suggestions {
		item := &types.SuggestionItem{
			ID:             sg.ID,
			SuggestionType: sg.SuggestionType,
			Content:        sg.Content,
			IsRead:         sg.IsRead,
			CreatedAt:      sg.CreatedAt.Format(time.RFC3339),
		}
		if sg.SessionID != nil {
			item.SessionID = *sg.SessionID
			item.SessionTitle = sg.SessionTitle
			item.SessionType = sg.SessionType
			item.SessionCreated = sg.SessionCreated.Format(time.RFC3339)
		}
		sessionSuggestions = append(sessionSuggestions, item)
	}

	return &types.AnalysisDetailResp{
		SessionID:          a.SessionID,
		StudentID:          studentId,
		SessionTitle:       session.Title,
		OverallScore:       a.OverallScore,
		TechnicalScore:     a.TechnicalScore,
		ExpressionScore:    a.ExpressionScore,
		LogicScore:         a.LogicScore,
		Strengths:          utils.ParseStringArrayJSON(a.Strengths),
		Weaknesses:         utils.ParseStringArrayJSON(a.Weaknesses),
		Suggestions:        a.Suggestions,
		DetailReport:       a.DetailReport,
		SessionSuggestions: sessionSuggestions,
		CreatedAt:          a.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (l *AnalysisLogic) GetOverview(userId int64) (*types.AnalysisOverviewResp, error) {
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return nil, err
	}

	const query = `SELECT COUNT(1),
       COALESCE(AVG(ia.overall_score::float8),0),
       COALESCE(AVG(ia.technical_score::float8),0),
       COALESCE(AVG(ia.expression_score::float8),0),
       COALESCE(AVG(ia.logic_score::float8),0)
FROM interview_analyses ia
JOIN interview_sessions sess ON sess.id = ia.session_id
WHERE sess.student_id=$1`

	var resp types.AnalysisOverviewResp
	if err := l.svcCtx.DB.QueryRow(l.ctx, query, s.ID).Scan(
		&resp.TotalCompleted,
		&resp.AvgOverall,
		&resp.AvgTechnical,
		&resp.AvgExpression,
		&resp.AvgLogic,
	); err != nil {
		return nil, err
	}

	const sessionsQuery = `SELECT latest.session_id, latest.title, latest.interview_type, latest.overall_score::float8, latest.created_at
FROM (
	SELECT DISTINCT ON (ia.session_id)
	       ia.session_id, sess.title, sess.interview_type, ia.overall_score, ia.created_at
	FROM interview_analyses ia
	JOIN interview_sessions sess ON sess.id = ia.session_id
	WHERE sess.student_id=$1
	ORDER BY ia.session_id, ia.created_at DESC
) latest
ORDER BY latest.created_at DESC
LIMIT 20`

	rows, err := l.svcCtx.DB.Query(l.ctx, sessionsQuery, s.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			sessionID     int64
			title         string
			interviewType string
			overallScore  float64
			createdAt     time.Time
		)
		if err := rows.Scan(&sessionID, &title, &interviewType, &overallScore, &createdAt); err != nil {
			return nil, err
		}
		resp.Sessions = append(resp.Sessions, &types.AnalysisOverviewSession{
			ID:            sessionID,
			Title:         title,
			Date:          createdAt.Format(time.RFC3339),
			OverallScore:  overallScore,
			InterviewType: interviewType,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (l *AnalysisLogic) GetTrend(userId int64, limit int) ([]*types.AnalysisTrendPoint, error) {
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 20
	}
	const query = `SELECT ia.created_at, ia.overall_score::float8
FROM interview_analyses ia
JOIN interview_sessions sess ON sess.id = ia.session_id
WHERE sess.student_id=$1
ORDER BY ia.created_at ASC
LIMIT $2`

	rows, err := l.svcCtx.DB.Query(l.ctx, query, s.ID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*types.AnalysisTrendPoint
	for rows.Next() {
		var (
			t  time.Time
			sc float64
		)
		if err := rows.Scan(&t, &sc); err != nil {
			return nil, err
		}
		list = append(list, &types.AnalysisTrendPoint{
			CreatedAt:    t.Format(time.RFC3339),
			OverallScore: sc,
		})
	}

	return list, rows.Err()
}

func (l *AnalysisLogic) fetchSessionMessages(chatID string, limit int) ([]openai.ChatCompletionMessage, error) {
	if limit <= 0 {
		limit = 200
	}

	const query = `SELECT role, content
FROM vector_store
WHERE chat_id=$1 AND source_type='message'
ORDER BY created_at ASC
LIMIT $2`

	rows, err := l.svcCtx.DB.Query(l.ctx, query, chatID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []openai.ChatCompletionMessage
	for rows.Next() {
		var role, content string
		if err := rows.Scan(&role, &content); err != nil {
			return nil, err
		}
		out = append(out, openai.ChatCompletionMessage{
			Role:    role,
			Content: content,
		})
	}
	return out, rows.Err()
}

func buildAnalysisPrompt(messages []openai.ChatCompletionMessage) string {
	var b strings.Builder
	b.WriteString("面试对话如下：\n")
	for i, m := range messages {
		content := utils.TruncateText(strings.TrimSpace(m.Content), 800)
		b.WriteString(strings.TrimSpace(strings.ToLower(m.Role)))
		b.WriteString(": ")
		b.WriteString(content)
		if i != len(messages)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func extractJSONObject(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	if strings.HasPrefix(s, "{") && json.Valid([]byte(s)) {
		return s
	}

	start := -1
	depth := 0
	inString := false
	escaped := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 && start >= 0 {
				candidate := strings.TrimSpace(s[start : i+1])
				if json.Valid([]byte(candidate)) {
					return candidate
				}
				start = -1
			}
		}
	}

	return ""
}
