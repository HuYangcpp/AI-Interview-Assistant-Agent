package logic

import (
	"context"
	"errors"
	"fmt"
	openai "github.com/sashabaranov/go-openai"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"path/filepath"
	"strings"
	"time"

	"ai-gozero-agent/api/internal/model"
	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"
	"ai-gozero-agent/api/internal/utils"
)

type ChatLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type resumePromptContext struct {
	Mode        string
	DirectText  string
	SummaryText string
}

func NewChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatLogic {
	return &ChatLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ChatLogic) Chat(session *model.InterviewSession, message string) (<-chan *types.ChatResponse, error) {
	return l.chat(session, message, true)
}

func (l *ChatLogic) AutoStart(session *model.InterviewSession, message string) (<-chan *types.ChatResponse, error) {
	return l.chat(session, message, false)
}

func (l *ChatLogic) chat(session *model.InterviewSession, message string, persistUserMessage bool) (<-chan *types.ChatResponse, error) {
	ch := make(chan *types.ChatResponse)

	// 读取学生简历上下文
	resumeCtx := l.loadResumeContext(session.StudentID)

	go func() {
		defer close(ch)
		stateManager := NewStateManager(l.svcCtx)
		chatID := session.ChatID

		// 1. 保存用户消息
		if persistUserMessage {
			if err := l.svcCtx.VectorStore.SaveMessage(
				l.ctx,
				chatID,
				openai.ChatMessageRoleUser,
				message,
			); err != nil {
				l.Logger.Errorf("保存用户消息失败: %v", err)
			}
		}

		// 2. 获取当前状态（确保初始化）
		currentState, err := stateManager.GetOrInitState(chatID)
		if err != nil {
			l.Logger.Errorf("获取状态失败: %v", err)
			currentState = types.StateStart
		}

		// 3. 知识检索
		knowledge, err := l.svcCtx.VectorStore.RetrieveKnowledge(message, 3)
		if err != nil {
			l.Logger.Errorf("知识检索失败: %v", err)
			knowledge = []types.KnowledgeChunk{}
		} else {
			for _, k := range knowledge {
				l.Logger.Infof("[RAG] title=%s similarity=%.4f content_len=%d",
					k.Title, k.Similarity, len(k.Content))
			}
		}

		var resumeChunks []types.ResumeChunk
		if resumeCtx.Mode == "long" {
			resumeChunks, err = l.svcCtx.VectorStore.RetrieveResumeChunks(session.StudentID, message, 2)
			if err != nil {
				l.Logger.Errorf("简历检索失败: %v", err)
				resumeChunks = nil
			}
		}

		// 4. 构建系统消息（带状态 + 简历）
		messages, err := l.buildMessagesWithState(chatID, session.InterviewType, currentState, knowledge, resumeCtx, resumeChunks, session.CustomPrompt)
		if err != nil {
			l.Logger.Errorf("构建消息失败: %v", err)
			ch <- &types.ChatResponse{Content: "系统错误：无法构建对话", IsLast: true}
			return
		}
		if !persistUserMessage {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: message,
			})
		}

		// 5. 创建OpenAI请求
		request := openai.ChatCompletionRequest{
			Model:       l.svcCtx.Config.OpenAI.Model,
			Messages:    messages,
			Stream:      true,
			MaxTokens:   l.svcCtx.Config.OpenAI.MaxTokens,
			Temperature: l.svcCtx.Config.OpenAI.Temperature,
		}

		// 6. 处理流式响应
		var fullResponse strings.Builder
		streamedLen := 0
		stream, err := l.svcCtx.OpenAIClient.CreateChatCompletionStream(l.ctx, request)
		if err != nil {
			l.Logger.Error("创建聊天完成流失败: ", err)
			ch <- &types.ChatResponse{Content: "系统错误：无法连接AI服务", IsLast: true}
			return
		}
		defer stream.Close()

		for {
			select {
			case <-l.ctx.Done():
				return
			default:
				response, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					rawResponse := fullResponse.String()
					finalResponse := strings.TrimSpace(stripStateTags(rawResponse))
					if finalResponse != "" && len(finalResponse) > streamedLen {
						delta := finalResponse[streamedLen:]
						streamedLen = len(finalResponse)
						ch <- &types.ChatResponse{Content: delta, IsLast: false}
					}

					if rawResponse != "" {
						if finalResponse != "" {
							if err := l.svcCtx.VectorStore.SaveMessage(
								l.ctx,
								chatID,
								openai.ChatMessageRoleAssistant,
								finalResponse,
							); err != nil {
								l.Logger.Errorf("保存助手消息失败: %v", err)
							}
						}

						newState, err := stateManager.EvaluateAndUpdateState(chatID, rawResponse)
						if err != nil {
							l.Logger.Errorf("更新状态失败: %v", err)
						} else {
							l.Logger.Infof("状态更新: %s -> %s", currentState, newState)
							if newState != currentState && newState == types.StateQuestion {
								if err := l.svcCtx.InterviewSessionModel.IncrementQuestions(l.ctx, session.ID); err != nil {
									l.Logger.Errorf("递增题目数失败: %v", err)
								}
							}
							if newState != currentState && newState == types.StateEnd {
								duration := int(time.Since(session.CreatedAt).Seconds())
								if duration < 0 {
									duration = 0
								}
								if err := l.svcCtx.InterviewSessionModel.MarkCompleted(l.ctx, session.ID, duration); err != nil && !errors.Is(err, model.ErrNotFound) {
									l.Logger.Errorf("结束会话失败: %v", err)
								}
								go func(studentId, sessionId int64) {
									ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
									defer cancel()
									if err := NewAnalysisLogic(ctx, l.svcCtx).GenerateAnalysis(studentId, sessionId); err != nil {
										l.Logger.Errorf("生成分析失败: %v", err)
									}
								}(session.StudentID, session.ID)
							}
						}
					}

					ch <- &types.ChatResponse{IsLast: true}
					return
				}
				if err != nil {
					l.Logger.Error("接收流数据失败: ", err)
					return
				}

				if len(response.Choices) > 0 && response.Choices[0].Delta.Content != "" {
					content := response.Choices[0].Delta.Content
					fullResponse.WriteString(content)

					visibleResponse := streamVisibleAssistantText(fullResponse.String())
					if len(visibleResponse) > streamedLen {
						delta := visibleResponse[streamedLen:]
						streamedLen = len(visibleResponse)
						ch <- &types.ChatResponse{
							Content: delta,
							IsLast:  false,
						}
					}
				}
			}
		}
	}()

	return ch, nil
}

// 构建带状态的消息
func (l *ChatLogic) buildMessagesWithState(chatId, interviewType, currentState string, knowledge []types.KnowledgeChunk, resumeCtx resumePromptContext, resumeChunks []types.ResumeChunk, customPrompt string) ([]openai.ChatCompletionMessage, error) {
	// 构建状态特定的系统消息
	var systemMessage string
	if customPrompt != "" {
		systemMessage = customPrompt
	} else {
		systemMessage = buildInterviewSystemPrompt(interviewType)
	}
	systemMessage += "\n\n当前状态: " + currentState

	switch currentState {
	case types.StateStart:
		systemMessage += "\n当前阶段: 开场。简短欢迎候选人，然后直接提出第一个技术问题。"
	case types.StateQuestion:
		systemMessage += "\n当前阶段: 提问。直接提出一个技术问题，不要解释题目背景，不要给出任何提示。"
	case types.StateFollowUp:
		systemMessage += "\n当前阶段: 追问。针对候选人刚才的回答进行追问，考察更深层的理解。不要评价对错，不要补充答案。"
	case types.StateEvaluate:
		systemMessage += "\n当前阶段: 评估。对候选人整体表现做简短评价，可以指出优缺点，但不要给出知识讲解。"
	case types.StateEnd:
		systemMessage += "\n当前阶段: 结束。感谢候选人参与，告知面试结束。"
	}

	// 注入简历内容
	if resumeCtx.Mode == "long" && resumeCtx.SummaryText != "" {
		systemMessage += "\n\n候选人简历摘要：\n" + resumeCtx.SummaryText
		systemMessage += "\n请优先根据摘要控制提问方向，再结合相关简历片段细化追问。"
	} else if resumeCtx.DirectText != "" {
		truncated := utils.TruncateText(resumeCtx.DirectText, shortResumeInjectMaxLength)
		systemMessage += "\n\n候选人简历信息：\n" + truncated
		systemMessage += "\n请根据简历内容有针对性地提问，考察与其背景相关的技术点。"
	}

	if len(resumeChunks) > 0 {
		systemMessage += "\n\n候选人相关简历片段："
		for i, chunk := range resumeChunks {
			if chunk.Similarity < 0.45 {
				continue
			}
			truncatedContent := utils.TruncateText(chunk.Content, 400)
			systemMessage += fmt.Sprintf("\n[简历片段%d] %s", i+1, truncatedContent)
		}
	}

	// 注入知识库
	if len(knowledge) > 0 {
		systemMessage += "\n\n相关背景知识："
		for i, k := range knowledge {
			if k.Similarity < 0.6 {
				continue
			}
			truncatedContent := utils.TruncateText(k.Content, 500)
			systemMessage += fmt.Sprintf("\n[知识片段%d] %s: %s", i+1, k.Title, truncatedContent)
		}
	}

	// 获取历史消息
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemMessage,
		},
	}

	history, err := l.svcCtx.VectorStore.GetMessages(l.ctx, chatId, 10)
	if err != nil {
		return nil, err
	}

	for _, msg := range history {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return messages, nil
}

// loadResumeContext 读取学生简历上下文
func (l *ChatLogic) loadResumeContext(studentId int64) resumePromptContext {
	resumeCacheKey := fmt.Sprintf("resume:%d", studentId)
	summaryCacheKey := fmt.Sprintf("resume_summary:%d", studentId)
	modeCacheKey := fmt.Sprintf("resume_mode:%d", studentId)

	student, err := l.svcCtx.StudentModel.FindById(l.ctx, studentId)
	if err != nil || student.ResumeURL == "" {
		return resumePromptContext{}
	}

	if student.ResumeMode == "long" {
		summaryText := strings.TrimSpace(student.ResumeSummaryText)
		if summaryText == "" {
			if cached, err := l.svcCtx.Redis.Get(l.ctx, summaryCacheKey).Result(); err == nil && cached != "" {
				summaryText = cached
			}
		}
		if summaryText != "" {
			_ = l.svcCtx.Redis.SetEx(l.ctx, summaryCacheKey, summaryText, time.Hour).Err()
			_ = l.svcCtx.Redis.SetEx(l.ctx, modeCacheKey, "long", time.Hour).Err()
			return resumePromptContext{Mode: "long", SummaryText: summaryText}
		}
	}

	if cached, err := l.svcCtx.Redis.Get(l.ctx, resumeCacheKey).Result(); err == nil && cached != "" {
		mode := student.ResumeMode
		if mode == "" {
			if modeCached, err := l.svcCtx.Redis.Get(l.ctx, modeCacheKey).Result(); err == nil {
				mode = modeCached
			}
		}
		if mode == "long" {
			return resumePromptContext{Mode: "long", SummaryText: student.ResumeSummaryText}
		}
		return resumePromptContext{Mode: "short", DirectText: cached}
	}

	uploadDir := l.svcCtx.Config.UploadDir
	relativePath := strings.TrimPrefix(student.ResumeURL, "/uploads")
	filePath := filepath.Join(uploadDir, relativePath)

	text, err := utils.ExtractPDFTextFromFile(filePath)
	if err != nil {
		l.Logger.Errorf("解析简历PDF失败: %v", err)
		return resumePromptContext{Mode: student.ResumeMode, SummaryText: student.ResumeSummaryText}
	}

	text = normalizeResumeText(text)
	if text == "" {
		return resumePromptContext{Mode: student.ResumeMode, SummaryText: student.ResumeSummaryText}
	}
	_ = l.svcCtx.Redis.SetEx(l.ctx, resumeCacheKey, text, time.Hour).Err()

	if student.ResumeMode == "long" {
		return resumePromptContext{Mode: "long", SummaryText: student.ResumeSummaryText}
	}

	return resumePromptContext{Mode: "short", DirectText: text}
}

func buildInterviewSystemPrompt(interviewType string) string {
	var role string
	switch interviewType {
	case "go":
		role = "Go语言"
	case "java":
		role = "Java"
	case "frontend":
		role = "前端工程"
	case "system_design":
		role = "系统设计"
	default:
		role = "软件工程"
	}

	return fmt.Sprintf(`你是一名资深%s面试官，正在对候选人进行技术面试。

【角色要求】
- 你的身份是面试官，不是老师、不是助手、不是辅导者
- 你的任务是通过提问来考察候选人的技术能力，而不是教导或帮助候选人
- 你只能提问、追问、给出简短的过渡语，以及在面试结束时给出评价
- 严禁输出任何形式的答案解析、知识讲解、"正确答案是..."、"你可以这样回答..."等内容
- 严禁以辅导者或教练的口吻说话

【行为规范】
- 每次只问一个问题
- 候选人回答后，根据回答质量决定追问还是进入下一题
- 如果候选人回答错误或不完整，不要纠正或补充，只需追问或继续下一题
- 保持专业、简洁的面试官语气
- 不要使用"✅"、"小结"、"总结"等辅导类格式

【禁止输出的内容示例】
- "✅ 小结（你可以这样回答面试官）"
- "正确答案是..."
- "这道题考察的是..."
- "建议你..."
- 任何形式的答案讲解

【输出格式要求】
每次回复的最后一行必须输出一个状态标记，格式为：
[STATE:当前状态]

状态值只能是以下之一：
- [STATE:QUESTION]   - 当你刚提出了一个面试问题
- [STATE:FOLLOWUP]   - 当你在追问候选人
- [STATE:EVALUATE]   - 当你在对候选人整体表现做评价
- [STATE:END]        - 当面试结束

此标记会被系统自动过滤，候选人不会看到。`, role)
}

func streamVisibleAssistantText(raw string) string {
	if idx := strings.LastIndex(raw, "\n"); idx >= 0 && isStateTagPrefix(raw[idx+1:]) {
		raw = raw[:idx]
	} else if idx < 0 && isStateTagPrefix(raw) {
		return ""
	}

	return stripStateTags(raw)
}
