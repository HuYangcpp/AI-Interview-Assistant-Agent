package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-gozero-agent/api/internal/model"
	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"
	"ai-gozero-agent/api/internal/utils"

	openai "github.com/sashabaranov/go-openai"
	"github.com/zeromicro/go-zero/core/logx"
)

type StudentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStudentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StudentLogic {
	return &StudentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StudentLogic) GetProfile(userId int64) (*types.StudentProfileResp, error) {
	u, err := l.svcCtx.UserModel.FindById(l.ctx, userId)
	if err != nil {
		return nil, err
	}
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return nil, err
	}

	skills := utils.ParseStringArrayJSON(s.Skills)

	return &types.StudentProfileResp{
		StudentId:        s.ID,
		UserId:           u.ID,
		Username:         u.Username,
		RealName:         u.RealName,
		Phone:            u.Phone,
		Email:            u.Email,
		StudentNo:        s.StudentNo,
		Major:            s.Major,
		ClassName:        s.ClassName,
		GraduationYear:   s.GraduationYear,
		Skills:           skills,
		SelfIntroduction: s.SelfIntroduction,
		ResumeURL:        s.ResumeURL,
		ResumeMode:       s.ResumeMode,
		ResumeSummary:    s.ResumeSummaryText,
	}, nil
}

func (l *StudentLogic) UpdateProfile(userId int64, req *types.UpdateStudentProfileReq) error {
	u, err := l.svcCtx.UserModel.FindById(l.ctx, userId)
	if err != nil {
		return err
	}
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return err
	}

	skillsJSON, err := json.Marshal(req.Skills)
	if err != nil {
		return err
	}

	u.RealName = strings.TrimSpace(req.RealName)
	u.Phone = strings.TrimSpace(req.Phone)
	u.Email = strings.TrimSpace(req.Email)
	if err := l.svcCtx.UserModel.Update(l.ctx, u); err != nil {
		return err
	}

	s.Skills = string(skillsJSON)
	s.SelfIntroduction = strings.TrimSpace(req.SelfIntroduction)
	return l.svcCtx.StudentModel.Update(l.ctx, s)
}

func (l *StudentLogic) ListProfileChangeRequests(userId int64) ([]*types.StudentProfileChangeRequestItem, error) {
	student, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return nil, err
	}

	const query = `
SELECT id, requested_student_no, requested_major, requested_class_name,
       requested_graduation_year, reason, status, review_comment,
       created_at, reviewed_at
FROM student_profile_change_requests
WHERE student_id=$1
ORDER BY created_at DESC
LIMIT 20
`

	rows, err := l.svcCtx.DB.Query(l.ctx, query, student.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*types.StudentProfileChangeRequestItem
	for rows.Next() {
		item := new(types.StudentProfileChangeRequestItem)
		if err := rows.Scan(
			&item.ID,
			&item.RequestedStudentNo,
			&item.RequestedMajor,
			&item.RequestedClassName,
			&item.RequestedGraduationYear,
			&item.Reason,
			&item.Status,
			&item.ReviewComment,
			&item.CreatedAt,
			&item.ReviewedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (l *StudentLogic) CreateProfileChangeRequest(userId int64, req *types.StudentCreateProfileChangeRequestReq) error {
	student, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return err
	}

	requestedStudentNo := strings.TrimSpace(req.StudentNo)
	requestedMajor := strings.TrimSpace(req.Major)
	requestedClassName := strings.TrimSpace(req.ClassName)
	reason := strings.TrimSpace(req.Reason)

	if requestedStudentNo == "" {
		requestedStudentNo = student.StudentNo
	}
	if requestedMajor == "" {
		requestedMajor = student.Major
	}
	if requestedClassName == "" {
		requestedClassName = student.ClassName
	}
	if req.GraduationYear == 0 {
		req.GraduationYear = student.GraduationYear
	}
	if reason == "" {
		return errors.New("reason required")
	}

	if requestedStudentNo == student.StudentNo &&
		requestedMajor == student.Major &&
		requestedClassName == student.ClassName &&
		req.GraduationYear == student.GraduationYear {
		return errors.New("请至少提交一项变更")
	}

	const query = `
INSERT INTO student_profile_change_requests (
	student_id, requested_by, requested_student_no, requested_major,
	requested_class_name, requested_graduation_year, reason
) VALUES ($1,$2,$3,$4,$5,$6,$7)
`

	_, err = l.svcCtx.DB.Exec(l.ctx, query,
		student.ID,
		userId,
		requestedStudentNo,
		requestedMajor,
		requestedClassName,
		req.GraduationYear,
		reason,
	)
	return err
}

// allowedResumeExts 允许的简历文件扩展名
var allowedResumeExts = map[string]bool{
	".pdf": true,
}

// maxResumeSize 最大简历文件大小：10MB
const maxResumeSize = 10 << 20

const (
	shortResumeThreshold       = 1500
	shortResumeInjectMaxLength = 1500
	resumeChunkSize            = 400
)

type resumeSummaryItem struct {
	Company        string `json:"company"`
	Role           string `json:"role"`
	Responsibility string `json:"responsibility"`
	Achievement    string `json:"achievement"`
}

type resumeSummary struct {
	Name            string              `json:"name"`
	Years           string              `json:"years"`
	TargetDirection string              `json:"target_direction"`
	WorkExperience  []resumeSummaryItem `json:"work_experience"`
	CoreSkills      []string            `json:"core_skills"`
	Highlights      []string            `json:"highlights"`
	CurrentFocus    string              `json:"current_focus"`
	Interests       []string            `json:"interests"`
}

func (l *StudentLogic) UploadResume(userId int64, file multipart.File, header *multipart.FileHeader) (string, error) {
	// 文件大小校验
	if header.Size > maxResumeSize {
		return "", errors.New("文件大小超过限制（最大 10MB）")
	}

	// 扩展名校验
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedResumeExts[ext] {
		return "", errors.New("当前仅支持 PDF 格式")
	}

	filename := header.Filename

	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return "", err
	}

	uploadDir := strings.TrimSpace(l.svcCtx.Config.UploadDir)
	if uploadDir == "" {
		return "", errors.New("uploadDir not configured")
	}

	resumeDir := filepath.Join(uploadDir, "resumes")
	if err := os.MkdirAll(resumeDir, 0o755); err != nil {
		return "", err
	}

	safeName := filepath.Base(filename)
	safeName = strings.ReplaceAll(safeName, " ", "_")
	if safeName == "." || safeName == string(filepath.Separator) {
		safeName = "resume" + ext
	}

	outName := strings.Join([]string{
		itoa64(s.ID),
		itoa64(time.Now().UnixNano()),
		safeName,
	}, "_")
	outPath := filepath.Join(resumeDir, outName)

	out, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		return "", err
	}

	// 仅保存路径（部署时可自行配置静态资源访问）
	resumeURL := "/uploads/resumes/" + outName
	s.ResumeURL = resumeURL
	if err := l.svcCtx.StudentModel.Update(l.ctx, s); err != nil {
		return "", err
	}

	if err := l.processResumeAfterUpload(s, outPath, ext); err != nil {
		l.Logger.Errorf("处理简历增强信息失败: %v", err)
	}

	return resumeURL, nil
}

func normalizeResumeText(text string) string {
	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		cleaned = append(cleaned, line)
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func buildResumeChunks(text string, maxLen int) []string {
	lines := strings.Split(text, "\n")
	var chunks []string
	var current strings.Builder

	flush := func() {
		chunk := strings.TrimSpace(current.String())
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		current.Reset()
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len([]rune(line)) > maxLen {
			if current.Len() > 0 {
				flush()
			}
			chunks = append(chunks, utils.SplitText(line, maxLen)...)
			continue
		}
		if current.Len() > 0 && len([]rune(current.String()))+len([]rune(line))+1 > maxLen {
			flush()
		}
		if current.Len() > 0 {
			current.WriteString("\n")
		}
		current.WriteString(line)
	}
	flush()

	if len(chunks) == 0 {
		return utils.SplitText(text, maxLen)
	}
	return chunks
}

func fallbackResumeSummaryText(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) > 8 {
		lines = lines[:8]
	}
	return "候选人简历摘要：\n" + strings.Join(lines, "\n")
}

func extractJSONObjectText(s string) string {
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
			}
		}
	}
	return ""
}

func renderResumeSummaryText(summary resumeSummary) string {
	var lines []string
	lines = append(lines, "候选人简历摘要：")
	if summary.Years != "" || summary.TargetDirection != "" {
		base := strings.TrimSpace(strings.Join([]string{summary.Years, summary.TargetDirection}, "，"))
		if base != "" {
			lines = append(lines, "- 基础背景："+base)
		}
	}
	if len(summary.WorkExperience) > 0 {
		for _, item := range summary.WorkExperience {
			parts := []string{item.Company, item.Role}
			if item.Responsibility != "" {
				parts = append(parts, "职责："+item.Responsibility)
			}
			if item.Achievement != "" {
				parts = append(parts, "成果："+item.Achievement)
			}
			lines = append(lines, "- 工作经历："+strings.Join(parts, "，"))
		}
	}
	if len(summary.CoreSkills) > 0 {
		lines = append(lines, "- 核心技能："+strings.Join(summary.CoreSkills, "、"))
	}
	if len(summary.Highlights) > 0 {
		lines = append(lines, "- 重点亮点："+strings.Join(summary.Highlights, "；"))
	}
	if summary.CurrentFocus != "" {
		lines = append(lines, "- 当前关注："+summary.CurrentFocus)
	}
	if len(summary.Interests) > 0 {
		lines = append(lines, "- 其他信息："+strings.Join(summary.Interests, "、"))
	}
	return strings.Join(lines, "\n")
}

func (l *StudentLogic) generateResumeSummary(rawText string) (string, string, error) {
	prompt := "请从以下中文简历文本中提取结构化摘要，只输出一个 JSON 对象，不要输出任何解释、Markdown 或代码块。" +
		"JSON 字段必须包含：" +
		"name, years, target_direction, work_experience(数组，每项包含 company, role, responsibility, achievement), core_skills(数组), highlights(数组), current_focus, interests(数组)。\n\n" +
		"简历文本：\n" + rawText

	ctx, cancel := context.WithTimeout(l.ctx, 60*time.Second)
	defer cancel()

	resp, err := l.svcCtx.OpenAIClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: l.svcCtx.Config.OpenAI.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "你是简历信息抽取助手。你必须只输出一个 JSON 对象，不要输出任何其他文字。",
			},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		MaxTokens:   800,
		Temperature: 0.1,
	})
	if err != nil {
		return "", "", err
	}
	if len(resp.Choices) == 0 {
		return "", "", errors.New("empty resume summary response")
	}

	jsonText := extractJSONObjectText(resp.Choices[0].Message.Content)
	if jsonText == "" {
		return "", "", errors.New("resume summary json not found")
	}

	var summary resumeSummary
	if err := json.Unmarshal([]byte(jsonText), &summary); err != nil {
		return "", "", err
	}

	return jsonText, renderResumeSummaryText(summary), nil
}

func (l *StudentLogic) processResumeAfterUpload(student *model.Student, filePath, ext string) error {
	resumeCacheKey := fmt.Sprintf("resume:%d", student.ID)
	summaryCacheKey := fmt.Sprintf("resume_summary:%d", student.ID)
	modeCacheKey := fmt.Sprintf("resume_mode:%d", student.ID)

	if ext != ".pdf" {
		if err := l.svcCtx.StudentModel.UpdateResumeProcessing(l.ctx, student.ID, "short", 0, "", ""); err != nil {
			return err
		}
		if err := l.svcCtx.VectorStore.ReplaceResumeChunks(student.ID, nil); err != nil {
			return err
		}
		_ = l.svcCtx.Redis.Del(l.ctx, resumeCacheKey, summaryCacheKey, modeCacheKey).Err()
		return nil
	}

	text, err := utils.ExtractPDFTextFromFile(filePath)
	if err != nil {
		return err
	}
	text = normalizeResumeText(text)
	if text == "" {
		if err := l.svcCtx.StudentModel.UpdateResumeProcessing(l.ctx, student.ID, "short", 0, "", ""); err != nil {
			return err
		}
		if err := l.svcCtx.VectorStore.ReplaceResumeChunks(student.ID, nil); err != nil {
			return err
		}
		_ = l.svcCtx.Redis.Del(l.ctx, resumeCacheKey, summaryCacheKey, modeCacheKey).Err()
		return nil
	}

	length := len([]rune(text))
	mode := "short"
	summaryJSON := ""
	summaryText := ""
	var chunks []string

	if length > shortResumeThreshold {
		mode = "long"
		summaryJSON, summaryText, err = l.generateResumeSummary(text)
		if err != nil {
			l.Logger.Errorf("生成简历摘要失败，使用回退摘要: %v", err)
			summaryText = fallbackResumeSummaryText(text)
			rawJSON, _ := json.Marshal(map[string]string{"fallback_summary": summaryText})
			summaryJSON = string(rawJSON)
		}
		chunks = buildResumeChunks(text, resumeChunkSize)
	}

	if err := l.svcCtx.StudentModel.UpdateResumeProcessing(l.ctx, student.ID, mode, length, summaryJSON, summaryText); err != nil {
		return err
	}
	if err := l.svcCtx.VectorStore.ReplaceResumeChunks(student.ID, chunks); err != nil {
		return err
	}

	_ = l.svcCtx.Redis.SetEx(l.ctx, resumeCacheKey, text, time.Hour).Err()
	if summaryText != "" {
		_ = l.svcCtx.Redis.SetEx(l.ctx, summaryCacheKey, summaryText, time.Hour).Err()
	} else {
		_ = l.svcCtx.Redis.Del(l.ctx, summaryCacheKey).Err()
	}
	_ = l.svcCtx.Redis.SetEx(l.ctx, modeCacheKey, mode, time.Hour).Err()

	return nil
}

func itoa64(i int64) string {
	if i == 0 {
		return "0"
	}

	var buf [32]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
