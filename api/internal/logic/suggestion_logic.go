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

type SuggestionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSuggestionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SuggestionLogic {
	return &SuggestionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

type suggestionAIResp struct {
	Career    string `json:"career"`
	Skill     string `json:"skill"`
	Interview string `json:"interview"`
	Resume    string `json:"resume"`
}

const suggestionMaxLen = 90

func buildFallbackSuggestionContent(student *model.Student, analysis *model.InterviewAnalysis, employment map[string]any) suggestionAIResp {
	strengths := utils.ParseStringArrayJSON(analysis.Strengths)
	weaknesses := utils.ParseStringArrayJSON(analysis.Weaknesses)
	skills := utils.ParseStringArrayJSON(student.Skills)

	strengthText := joinSuggestionPoints(strengths)
	weaknessText := joinSuggestionPoints(weaknesses)
	skillText := joinSuggestionPoints(skills)

	employmentStatus, _ := employment["status"].(string)
	companyName, _ := employment["company_name"].(string)
	position, _ := employment["position"].(string)

	career := fmt.Sprintf("本场面试综合得分 %.0f 分，建议继续围绕目标岗位打磨竞争力。", analysis.OverallScore)
	if companyName != "" || position != "" {
		career = fmt.Sprintf("你当前的就业进展已指向%s%s，建议把后续准备继续聚焦到该岗位能力要求上。", emptyFallback(companyName, "目标公司"), emptyFallback(position, "目标岗位"))
	}
	if employmentStatus == "seeking" || employmentStatus == "" {
		career += " 在继续投递时，优先选择与你本次面试表现更匹配的岗位，并把本场高分表现沉淀成项目亮点和自我介绍素材。"
	} else {
		career += " 后续可以针对该方向持续做专项准备，争取把单场高分转化为稳定的求职通过率。"
	}

	skill := fmt.Sprintf("技术维度得分 %.0f 分。", analysis.TechnicalScore)
	if weaknessText != "" {
		skill += " 优先补齐本场暴露出的短板：" + weaknessText + "。"
	}
	if skillText != "" {
		skill += " 同时把已有技能 " + skillText + " 继续强化为可量化的项目成果。"
	} else {
		skill += " 建议尽快补充并固化你的核心技能标签，方便后续面试和简历同步表达。"
	}

	interview := fmt.Sprintf("表达 %.0f 分、逻辑 %.0f 分。", analysis.ExpressionScore, analysis.LogicScore)
	if strengthText != "" {
		interview += " 保留本场表现较好的部分：" + strengthText + "。"
	}
	if weaknessText != "" {
		interview += " 下一轮面试重点针对 " + weaknessText + " 做复盘，练习更完整的答题结构和追问应对。"
	} else {
		interview += " 下一轮建议继续通过模拟追问训练，把高分状态稳定下来。"
	}

	resume := "建议把本场面试里表现较好的能力点同步回简历。"
	if skillText != "" {
		resume += " 将 " + skillText + " 放到技能标签和项目描述的前排位置。"
	} else {
		resume += " 当前技能标签仍不够完整，建议补齐核心技术栈、项目职责和量化结果。"
	}
	if weaknessText != "" {
		resume += " 对于本场暴露出的短板 " + weaknessText + "，可以在简历中增加能证明补强过程的项目或实践。"
	}

	return suggestionAIResp{
		Career:    strings.TrimSpace(career),
		Skill:     strings.TrimSpace(skill),
		Interview: strings.TrimSpace(interview),
		Resume:    strings.TrimSpace(resume),
	}
}

func fillEmptySuggestionContent(ai suggestionAIResp, fallback suggestionAIResp) suggestionAIResp {
	if strings.TrimSpace(ai.Career) == "" {
		ai.Career = fallback.Career
	}
	if strings.TrimSpace(ai.Skill) == "" {
		ai.Skill = fallback.Skill
	}
	if strings.TrimSpace(ai.Interview) == "" {
		ai.Interview = fallback.Interview
	}
	if strings.TrimSpace(ai.Resume) == "" {
		ai.Resume = fallback.Resume
	}
	return ai
}

func normalizeSuggestionContent(ai suggestionAIResp) suggestionAIResp {
	ai.Career = compactSuggestionText(ai.Career, suggestionMaxLen)
	ai.Skill = compactSuggestionText(ai.Skill, suggestionMaxLen)
	ai.Interview = compactSuggestionText(ai.Interview, suggestionMaxLen)
	ai.Resume = compactSuggestionText(ai.Resume, suggestionMaxLen)
	return ai
}

func joinSuggestionPoints(items []string) string {
	var cleaned []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		cleaned = append(cleaned, utils.TruncateText(item, 24))
		if len(cleaned) == 2 {
			break
		}
	}
	return strings.Join(cleaned, "；")
}

func emptyFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func compactSuggestionText(text string, maxLen int) string {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if maxLen > 0 {
		text = utils.TruncateText(text, maxLen)
	}
	return text
}

func (l *SuggestionLogic) List(userId int64) ([]*types.SuggestionItem, error) {
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return nil, err
	}
	list, err := l.svcCtx.SuggestionModel.FindByStudentId(l.ctx, s.ID, 100)
	if err != nil {
		return nil, err
	}

	var resp []*types.SuggestionItem
	for _, sg := range list {
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
		resp = append(resp, item)
	}
	return resp, nil
}

func (l *SuggestionLogic) MarkRead(userId, id int64) error {
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return err
	}
	return l.svcCtx.SuggestionModel.MarkRead(l.ctx, id, s.ID)
}

// GenerateSuggestions generates suggestions for a specific interview session.
func (l *SuggestionLogic) GenerateSuggestions(studentId, sessionId int64) error {
	student, err := l.svcCtx.StudentModel.FindById(l.ctx, studentId)
	if err != nil {
		return err
	}
	user, err := l.svcCtx.UserModel.FindById(l.ctx, student.UserID)
	if err != nil {
		return err
	}

	employment := map[string]any{}
	{
		const query = `SELECT status, company_name, position, city, salary_range
FROM employments WHERE student_id=$1 ORDER BY updated_at DESC NULLS LAST, id DESC LIMIT 1`
		var status, company, position, city, salary string
		if err := l.svcCtx.DB.QueryRow(l.ctx, query, studentId).Scan(&status, &company, &position, &city, &salary); err == nil {
			employment["status"] = status
			employment["company_name"] = company
			employment["position"] = position
			employment["city"] = city
			employment["salary_range"] = salary
		}
	}

	analysis, err := l.svcCtx.InterviewAnalysisModel.FindLatestBySessionId(l.ctx, sessionId)
	if err != nil {
		l.Logger.Errorf("获取分析记录失败: %v", err)
		return err
	}

	profileJSON, err := json.Marshal(map[string]any{
		"major":             student.Major,
		"class_name":        student.ClassName,
		"graduation_year":   student.GraduationYear,
		"skills":            utils.ParseStringArrayJSON(student.Skills),
		"self_introduction": student.SelfIntroduction,
		"real_name":         user.RealName,
	})
	if err != nil {
		return fmt.Errorf("序列化学生画像失败: %w", err)
	}

	employmentJSON, err := json.Marshal(employment)
	if err != nil {
		return fmt.Errorf("序列化就业信息失败: %w", err)
	}

	analysisSummary := map[string]any{
		"session_id":        analysis.SessionID,
		"overall_score":     analysis.OverallScore,
		"technical_score":   analysis.TechnicalScore,
		"expression_score":  analysis.ExpressionScore,
		"logic_score":       analysis.LogicScore,
		"strengths":         utils.ParseStringArrayJSON(analysis.Strengths),
		"weaknesses":        utils.ParseStringArrayJSON(analysis.Weaknesses),
		"session_suggested": analysis.Suggestions,
	}
	analysisJSON, err := json.Marshal(analysisSummary)
	if err != nil {
		return fmt.Errorf("序列化分析摘要失败: %w", err)
	}

	userPrompt := "请基于以下学生画像、就业状态、这一场面试分析，生成 4 类个性化建议，要求建议与本次面试表现一一对应，并且每类建议只保留最关键的行动项，控制为 1 句话、尽量不超过 70 个汉字，输出严格 JSON（不要输出多余文字，不要使用代码块）：\n" +
		"{\"career\":\"...\",\"skill\":\"...\",\"interview\":\"...\",\"resume\":\"...\"}\n\n" +
		"学生画像: " + string(profileJSON) + "\n" +
		"就业状态: " + string(employmentJSON) + "\n" +
		"本次面试分析: " + string(analysisJSON)

	req := openai.ChatCompletionRequest{
		Model: l.svcCtx.Config.OpenAI.Model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "你是专业的就业与面试辅导老师。你必须只输出一个 JSON 对象，不要输出任何额外文字、解释、Markdown 或代码块。"},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Stream:      false,
		MaxTokens:   l.svcCtx.Config.OpenAI.MaxTokens,
		Temperature: 0.3,
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

	var ai suggestionAIResp
	if err := json.Unmarshal([]byte(jsonText), &ai); err != nil {
		logx.Errorf("AI raw response: %s", content)
		return errors.New("invalid ai json, please retry")
	}
	ai = fillEmptySuggestionContent(ai, buildFallbackSuggestionContent(student, analysis, employment))
	ai = normalizeSuggestionContent(ai)

	if err := l.svcCtx.SuggestionModel.DeleteByStudentIdAndSessionId(l.ctx, studentId, sessionId); err != nil {
		return err
	}

	sessionID := sessionId
	suggestions := []model.PersonalizedSuggestion{
		{StudentID: studentId, SessionID: &sessionID, SuggestionType: "career", Content: strings.TrimSpace(ai.Career), IsRead: false},
		{StudentID: studentId, SessionID: &sessionID, SuggestionType: "skill", Content: strings.TrimSpace(ai.Skill), IsRead: false},
		{StudentID: studentId, SessionID: &sessionID, SuggestionType: "interview", Content: strings.TrimSpace(ai.Interview), IsRead: false},
		{StudentID: studentId, SessionID: &sessionID, SuggestionType: "resume", Content: strings.TrimSpace(ai.Resume), IsRead: false},
	}

	created := 0
	for _, sg := range suggestions {
		if sg.Content == "" {
			continue
		}
		if _, err := l.svcCtx.SuggestionModel.Insert(l.ctx, &sg); err != nil {
			l.Logger.Errorf("保存建议失败: %v", err)
			return err
		}
		created++
	}

	if created == 0 {
		return errors.New("no suggestions generated")
	}

	return nil
}

// GenerateSuggestionsForUser is the handler-facing wrapper: resolves studentId from userId.
func (l *SuggestionLogic) GenerateSuggestionsForUser(userId int64) error {
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return err
	}
	analyses, err := l.svcCtx.InterviewAnalysisModel.FindRecentByStudentId(l.ctx, s.ID, 1)
	if err != nil {
		return err
	}
	if len(analyses) == 0 {
		return errors.New("暂无可生成建议的面试分析")
	}
	return l.GenerateSuggestions(s.ID, analyses[0].SessionID)
}
