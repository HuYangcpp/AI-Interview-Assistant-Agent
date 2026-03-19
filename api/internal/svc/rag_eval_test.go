package svc

import (
	"context"
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode"

	"ai-gozero-agent/api/internal/utils"

	openai "github.com/sashabaranov/go-openai"
)

const (
	ragAnswerTimeout         = 60 * time.Second
	ragJudgeTimeout          = 60 * time.Second
	deterministicTemperature = math.SmallestNonzeroFloat32
)

var (
	ragEvalCases    = flattenDatasetCases(knowledgeEvalDatasets)
	resumeEvalCases = flattenDatasetCases(resumeEvalDatasets)
)

func modelCandidatesFromEnv(envKey string, defaults []string) []string {
	raw := strings.TrimSpace(os.Getenv(envKey))
	if raw == "" {
		return append([]string(nil), defaults...)
	}

	items := strings.Split(raw, ",")
	models := make([]string, 0, len(items))
	for _, item := range items {
		model := strings.TrimSpace(item)
		if model != "" {
			models = append(models, model)
		}
	}
	if len(models) == 0 {
		return append([]string(nil), defaults...)
	}
	return models
}

func evalChatModels() []string {
	return modelCandidatesFromEnv("RAG_EVAL_CHAT_MODELS", []string{
		"glm-5",
		"tongyi-xiaomi-analysis-pro",
		"qwen3.5-122b-a10b",
	})
}

func evalJudgeModels() []string {
	return modelCandidatesFromEnv("RAG_EVAL_JUDGE_MODELS", evalChatModels())
}

func evalAgenticModel() string {
	models := modelCandidatesFromEnv("RAG_EVAL_AGENTIC_MODELS", []string{"qwen3.5-122b-a10b"})
	return models[0]
}

func createChatCompletionWithFallback(
	ctx context.Context,
	client *openai.Client,
	models []string,
	request openai.ChatCompletionRequest,
) (openai.ChatCompletionResponse, string, error) {
	candidates := models
	if len(candidates) == 0 {
		candidates = []string{request.Model}
	}

	var lastErr error
	for _, model := range candidates {
		request.Model = model
		resp, err := client.CreateChatCompletion(ctx, request)
		if err == nil && len(resp.Choices) > 0 {
			return resp, model, nil
		}
		if err != nil {
			lastErr = fmt.Errorf("model %s: %w", model, err)
			continue
		}
		lastErr = fmt.Errorf("model %s returned no choices", model)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no available model candidates")
	}
	return openai.ChatCompletionResponse{}, "", lastErr
}

// keywordHitRate 计算 keywords 中有多少出现在 answer 中
func keywordHitRate(answer string, keywords []string) float64 {
	if len(keywords) == 0 {
		return 0
	}
	hit := 0
	for _, kw := range keywords {
		if strings.Contains(answer, kw) {
			hit++
		}
	}
	return float64(hit) / float64(len(keywords))
}

// askLLM 直接调用 LLM，无 RAG
func askLLM(client *openai.Client, models []string, question string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ragAnswerTimeout)
	defer cancel()

	resp, _, err := createChatCompletionWithFallback(ctx, client, models, openai.ChatCompletionRequest{
		Model: "",
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "你是一个技术问答助手，请简洁准确地回答问题。"},
			{Role: openai.ChatMessageRoleUser, Content: question},
		},
		MaxTokens:   300,
		Temperature: deterministicTemperature,
	})
	if err != nil || len(resp.Choices) == 0 {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

// askLLMWithRAG 检索知识后注入 prompt，再调用 LLM
func askLLMWithRAG(store *VectorStore, client *openai.Client, models []string, question string, topK int, threshold float64) (string, error) {
	chunks, err := store.RetrieveKnowledge(question, topK)
	if err != nil {
		return "", err
	}

	knowledgeContext := ""
	for i, c := range chunks {
		if c.Similarity < threshold {
			continue
		}
		knowledgeContext += fmt.Sprintf("[知识片段%d] %s\n", i+1, c.Content)
	}

	systemPrompt := "你是一个技术问答助手，请简洁准确地回答问题。"
	if knowledgeContext != "" {
		systemPrompt += "\n\n参考知识：\n" + knowledgeContext
	}

	ctx, cancel := context.WithTimeout(context.Background(), ragAnswerTimeout)
	defer cancel()

	resp, _, err := createChatCompletionWithFallback(ctx, client, models, openai.ChatCompletionRequest{
		Model: "",
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: question},
		},
		MaxTokens:   300,
		Temperature: deterministicTemperature,
	})
	if err != nil || len(resp.Choices) == 0 {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

var (
	jsonScorePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)"score"\s*:\s*([1-5])`),
		regexp.MustCompile(`(?i)score\s*[:=]\s*([1-5])`),
		regexp.MustCompile(`(?i)分数\s*[:：]\s*([1-5])`),
	}
	standaloneScorePattern = regexp.MustCompile(`(?:^|[^0-9])([1-5])(?:[^0-9]|$)`)
)

func normalizeJudgeOutput(raw string) string {
	replacer := strings.NewReplacer(
		"１", "1",
		"２", "2",
		"３", "3",
		"４", "4",
		"５", "5",
		"一", "1",
		"二", "2",
		"三", "3",
		"四", "4",
		"五", "5",
	)
	return replacer.Replace(strings.TrimSpace(raw))
}

func parseJudgeScore(raw string) int {
	normalized := normalizeJudgeOutput(raw)
	if normalized == "" {
		return 0
	}

	for _, pattern := range jsonScorePatterns {
		match := pattern.FindStringSubmatch(normalized)
		if len(match) == 2 {
			score := int(match[1][0] - '0')
			if score >= 1 && score <= 5 {
				return score
			}
		}
	}

	if match := standaloneScorePattern.FindStringSubmatch(normalized); len(match) == 2 {
		score := int(match[1][0] - '0')
		if score >= 1 && score <= 5 {
			return score
		}
	}

	for _, r := range normalized {
		if !unicode.IsDigit(r) {
			continue
		}
		score := int(r - '0')
		if score >= 1 && score <= 5 {
			return score
		}
	}

	return 0
}

type judgeResult struct {
	score int
	raw   string
	model string
	err   error
}

// llmJudge 用 LLM 对答案打分（1-5），LLM-as-Judge
func llmJudge(client *openai.Client, models []string, question, answer, groundTruth string) judgeResult {
	prompt := fmt.Sprintf(
		"请你充当严格评分器，只评估候选答案与参考答案在事实层面的符合程度。\n"+
			"评分标准：1=核心事实错误，2=只命中少量事实点，3=主要事实基本正确但细节不足，4=事实较完整且基本准确，5=事实准确且覆盖充分。\n"+
			"输出要求：只返回 JSON，对象中仅包含一个字段 score，例如 {\"score\": 4}。不要输出任何解释、Markdown 或额外文本。\n"+
			"问题：%s\n参考答案：%s\n候选答案：%s",
		question, groundTruth, answer)

	result := judgeResult{}
	seed := 1

	// 最多重试 3 次
	for attempt := 0; attempt < 3; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), ragJudgeTimeout)
		resp, usedModel, err := createChatCompletionWithFallback(ctx, client, models, openai.ChatCompletionRequest{
			Model: "",
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: "你是一个只输出 JSON 的评分器。"},
				{Role: openai.ChatMessageRoleUser, Content: prompt},
			},
			MaxTokens:   32,
			Temperature: deterministicTemperature,
			Seed:        &seed,
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
			// Qwen Judge 在思考模式下更容易把推理内容混入输出，导致 score 解析失败。
			ChatTemplateKwargs: map[string]any{
				"enable_thinking": false,
			},
		})
		cancel()

		result.model = usedModel
		if err != nil {
			result.err = err
			continue
		}
		if len(resp.Choices) == 0 {
			result.err = fmt.Errorf("judge model %s returned no choices", usedModel)
			continue
		}

		raw := strings.TrimSpace(resp.Choices[0].Message.Content)
		result.raw = raw
		result.score = parseJudgeScore(raw)
		if result.score != 0 {
			result.err = nil
			return result
		}

		result.err = fmt.Errorf("unable to parse judge score from raw output")
	}
	return result
}

func runEvalGroup(
	t *testing.T,
	groupName string,
	cases []evalCase,
	client *openai.Client,
	judgeModels []string,
	answerFn func(string) (string, error),
) (float64, float64, float64, int) {
	t.Helper()

	totalRouge, totalKW, totalScore := 0.0, 0.0, 0.0
	successCount := 0
	totalCases := len(cases)
	groupStart := time.Now()

	t.Logf("[%s] 开始评测，共 %d 题", groupName, totalCases)

	for i, tc := range cases {
		caseStart := time.Now()
		t.Logf("[%s][%d/%d] 开始: %s", groupName, i+1, totalCases, tc.question)

		answer, err := answerFn(tc.question)
		if err != nil {
			t.Logf("[%s][%d/%d] 调用失败，耗时=%s: %v", groupName, i+1, totalCases, time.Since(caseStart).Round(time.Millisecond), err)
			continue
		}

		rouge := rougeL(answer, tc.groundTruth)
		kw := keywordHitRate(answer, tc.keywords)
		judge := llmJudge(client, judgeModels, tc.question, answer, tc.groundTruth)

		totalRouge += rouge
		totalKW += kw
		totalScore += float64(judge.score)
		successCount++

		t.Logf(
			"[%s][%d/%d] 完成，耗时=%s，ROUGE-L=%.4f，关键词命中率=%.4f，LLM评分=%d",
			groupName,
			i+1,
			totalCases,
			time.Since(caseStart).Round(time.Millisecond),
			rouge,
			kw,
			judge.score,
		)
		if judge.score == 0 && judge.err != nil {
			t.Logf(
				"[%s][%d/%d] Judge 未得到有效分数，model=%s, raw=%q, err=%v",
				groupName,
				i+1,
				totalCases,
				judge.model,
				judge.raw,
				judge.err,
			)
		}
	}

	t.Logf("[%s] 评测结束，成功=%d/%d，总耗时=%s", groupName, successCount, totalCases, time.Since(groupStart).Round(time.Second))

	return totalRouge, totalKW, totalScore, successCount
}

func runJudgeOnlyGroup(
	t *testing.T,
	groupName string,
	cases []evalCase,
	client *openai.Client,
	judgeModels []string,
	answerFn func(string) (string, error),
) (float64, int, int) {
	t.Helper()

	totalScore := 0.0
	validJudgeCount := 0
	answerSuccessCount := 0
	totalCases := len(cases)
	groupStart := time.Now()

	t.Logf("[%s] 开始评分，仅统计 LLM 评分，共 %d 题", groupName, totalCases)

	for i, tc := range cases {
		caseStart := time.Now()
		t.Logf("[%s][%d/%d] 开始: %s", groupName, i+1, totalCases, tc.question)

		answer, err := answerFn(tc.question)
		if err != nil {
			t.Logf("[%s][%d/%d] 调用失败，耗时=%s: %v", groupName, i+1, totalCases, time.Since(caseStart).Round(time.Millisecond), err)
			continue
		}
		answerSuccessCount++

		judge := llmJudge(client, judgeModels, tc.question, answer, tc.groundTruth)
		if judge.score == 0 {
			t.Logf(
				"[%s][%d/%d] Judge 未得到有效分数，耗时=%s，model=%s, raw=%q, err=%v",
				groupName,
				i+1,
				totalCases,
				time.Since(caseStart).Round(time.Millisecond),
				judge.model,
				judge.raw,
				judge.err,
			)
			continue
		}

		totalScore += float64(judge.score)
		validJudgeCount++

		t.Logf(
			"[%s][%d/%d] 完成，耗时=%s，LLM评分=%d，Judge模型=%s",
			groupName,
			i+1,
			totalCases,
			time.Since(caseStart).Round(time.Millisecond),
			judge.score,
			judge.model,
		)
	}

	t.Logf(
		"[%s] 评分结束，有效评分=%d/%d，成功回答=%d/%d，总耗时=%s",
		groupName,
		validJudgeCount,
		totalCases,
		answerSuccessCount,
		totalCases,
		time.Since(groupStart).Round(time.Second),
	)

	return totalScore, validJudgeCount, answerSuccessCount
}

func TestRAGvsNoRAG(t *testing.T) {
	store := newTestStore(t)
	defer store.Pool.Close()

	openaiCfg := openai.DefaultConfig(testOpenAIKey)
	openaiCfg.BaseURL = testOpenAIBase
	client := openai.NewClientWithConfig(openaiCfg)
	chatModels := evalChatModels()
	judgeModels := evalJudgeModels()
	agenticModel := evalAgenticModel()
	testTitles := datasetTitles(knowledgeEvalDatasets)
	defer cleanupKnowledgeTitles(t, store, testTitles...)

	groups := []struct {
		name  string
		setup func() error
		fn    func(q string) (string, error)
	}{
		{
			name: "NoRAG",
			fn: func(q string) (string, error) {
				return askLLM(client, chatModels, q)
			},
		},
		{
			name: "RAG-FixedSize",
			setup: func() error {
				cleanupKnowledgeTitles(t, store, testTitles...)
				return saveDatasetsWithSplitter(t, store, knowledgeEvalDatasets,
					&utils.FixedSizeSplitter{ChunkSize: 200})
			},
			fn: func(q string) (string, error) {
				return askLLMWithRAG(store, client, chatModels, q, 3, 0.6)
			},
		},
		{
			name: "RAG-Markdown",
			setup: func() error {
				cleanupKnowledgeTitles(t, store, testTitles...)
				return saveDatasetsWithSplitter(t, store, knowledgeEvalDatasets,
					&utils.MarkdownSplitter{FallbackChunkSize: 200})
			},
			fn: func(q string) (string, error) {
				return askLLMWithRAG(store, client, chatModels, q, 3, 0.6)
			},
		},
		{
			name: "RAG-Agentic",
			setup: func() error {
				cleanupKnowledgeTitles(t, store, testTitles...)
				return saveDatasetsWithSplitter(t, store, knowledgeEvalDatasets,
					&utils.AgenticSplitter{Client: client, Model: agenticModel})
			},
			fn: func(q string) (string, error) {
				return askLLMWithRAG(store, client, chatModels, q, 3, 0.6)
			},
		},
	}

	t.Log("\n## RAG vs 无RAG 对比结果")
	t.Log("\n| 组别         | 平均 ROUGE-L | 平均关键词命中率 | 平均 LLM 评分 |")
	t.Log("|-------------|------------|----------------|-------------|")

	for _, g := range groups {
		if g.setup != nil {
			if err := g.setup(); err != nil {
				t.Fatalf("[%s] 准备知识库失败: %v", g.name, err)
			}
		}

		totalRouge, totalKW, totalScore, successCount := runEvalGroup(
			t,
			g.name,
			ragEvalCases,
			client,
			judgeModels,
			g.fn,
		)

		if successCount == 0 {
			t.Logf("| %-12s | ERROR（全部请求失败）|", g.name)
			continue
		}
		n := float64(successCount)
		t.Logf("| %-12s | %.4f       | %.4f           | %.2f  (%d/%d成功) |",
			g.name, totalRouge/n, totalKW/n, totalScore/n, successCount, len(ragEvalCases))
	}
}

// TestResumeRAGQuality 简历场景 RAG 生成质量对比（FixedSize / Markdown / Agentic）
func TestResumeRAGQuality(t *testing.T) {
	store := newTestStore(t)
	defer store.Pool.Close()

	openaiCfg := openai.DefaultConfig(testOpenAIKey)
	openaiCfg.BaseURL = testOpenAIBase
	client := openai.NewClientWithConfig(openaiCfg)
	chatModels := evalChatModels()
	judgeModels := evalJudgeModels()
	agenticModel := evalAgenticModel()
	testTitles := datasetTitles(resumeEvalDatasets)
	defer cleanupKnowledgeTitles(t, store, testTitles...)

	groups := []struct {
		name  string
		setup func() error
		fn    func(q string) (string, error)
	}{
		{
			name: "NoRAG",
			fn: func(q string) (string, error) {
				return askLLM(client, chatModels, q)
			},
		},
		{
			name: "RAG-FixedSize",
			setup: func() error {
				cleanupKnowledgeTitles(t, store, testTitles...)
				return saveDatasetsWithSplitter(t, store, resumeEvalDatasets,
					&utils.FixedSizeSplitter{ChunkSize: 100})
			},
			fn: func(q string) (string, error) {
				return askLLMWithRAG(store, client, chatModels, q, 3, 0.5)
			},
		},
		{
			name: "RAG-Markdown",
			setup: func() error {
				cleanupKnowledgeTitles(t, store, testTitles...)
				return saveDatasetsWithSplitter(t, store, resumeEvalDatasets,
					&utils.MarkdownSplitter{FallbackChunkSize: 100})
			},
			fn: func(q string) (string, error) {
				return askLLMWithRAG(store, client, chatModels, q, 3, 0.5)
			},
		},
		{
			name: "RAG-Agentic",
			setup: func() error {
				cleanupKnowledgeTitles(t, store, testTitles...)
				return saveDatasetsWithSplitter(t, store, resumeEvalDatasets,
					&utils.AgenticSplitter{Client: client, Model: agenticModel})
			},
			fn: func(q string) (string, error) {
				return askLLMWithRAG(store, client, chatModels, q, 3, 0.5)
			},
		},
	}

	t.Log("\n## 简历 RAG 生成质量对比结果")
	t.Log("\n| 组别         | 平均 ROUGE-L | 平均关键词命中率 | 平均 LLM 评分 | 成功样本数 |")
	t.Log("|-------------|------------|----------------|-------------|----------|")

	for _, g := range groups {
		if g.setup != nil {
			if err := g.setup(); err != nil {
				t.Fatalf("[%s] 准备知识库失败: %v", g.name, err)
			}
		}

		totalRouge, totalKW, totalScore, successCount := runEvalGroup(
			t,
			g.name,
			resumeEvalCases,
			client,
			judgeModels,
			g.fn,
		)

		if successCount == 0 {
			t.Logf("| %-12s | ERROR（全部请求失败）|", g.name)
			continue
		}
		n := float64(successCount)
		t.Logf("| %-12s | %.4f       | %.4f           | %.2f          | %d/%d |",
			g.name, totalRouge/n, totalKW/n, totalScore/n, successCount, len(resumeEvalCases))
	}
}

// TestResumeRAGJudgeScoreOnly 用 qwen3.6-plus 重跑简历多样本组，仅保留 LLM 评分结果。
func TestResumeRAGJudgeScoreOnly(t *testing.T) {
	store := newTestStore(t)
	defer store.Pool.Close()

	openaiCfg := openai.DefaultConfig(testOpenAIKey)
	openaiCfg.BaseURL = testOpenAIBase
	client := openai.NewClientWithConfig(openaiCfg)
	chatModels := modelCandidatesFromEnv("RAG_EVAL_CHAT_MODELS", []string{"qwen3.6-plus"})
	judgeModels := modelCandidatesFromEnv("RAG_EVAL_JUDGE_MODELS", []string{"qwen3.6-plus"})
	agenticModel := modelCandidatesFromEnv("RAG_EVAL_AGENTIC_MODELS", []string{"qwen3.6-plus"})[0]
	testTitles := datasetTitles(resumeEvalDatasets)
	defer cleanupKnowledgeTitles(t, store, testTitles...)

	groups := []struct {
		name  string
		setup func() error
		fn    func(q string) (string, error)
	}{
		{
			name: "NoRAG",
			fn: func(q string) (string, error) {
				return askLLM(client, chatModels, q)
			},
		},
		{
			name: "RAG-FixedSize",
			setup: func() error {
				cleanupKnowledgeTitles(t, store, testTitles...)
				return saveDatasetsWithSplitter(t, store, resumeEvalDatasets,
					&utils.FixedSizeSplitter{ChunkSize: 100})
			},
			fn: func(q string) (string, error) {
				return askLLMWithRAG(store, client, chatModels, q, 3, 0.5)
			},
		},
		{
			name: "RAG-Markdown",
			setup: func() error {
				cleanupKnowledgeTitles(t, store, testTitles...)
				return saveDatasetsWithSplitter(t, store, resumeEvalDatasets,
					&utils.MarkdownSplitter{FallbackChunkSize: 100})
			},
			fn: func(q string) (string, error) {
				return askLLMWithRAG(store, client, chatModels, q, 3, 0.5)
			},
		},
		{
			name: "RAG-Agentic",
			setup: func() error {
				cleanupKnowledgeTitles(t, store, testTitles...)
				return saveDatasetsWithSplitter(t, store, resumeEvalDatasets,
					&utils.AgenticSplitter{Client: client, Model: agenticModel})
			},
			fn: func(q string) (string, error) {
				return askLLMWithRAG(store, client, chatModels, q, 3, 0.5)
			},
		},
	}

	t.Log("\n## 简历多样本 LLM 评分结果（Score Only）")
	t.Logf("\n默认 Chat 模型：%s", strings.Join(chatModels, ","))
	t.Logf("默认 Judge 模型：%s", strings.Join(judgeModels, ","))
	t.Logf("默认 Agentic 切分模型：%s", agenticModel)
	t.Log("\n| 组别         | 平均 LLM 评分 | 有效评分样本数 | 成功回答样本数 |")
	t.Log("|-------------|-------------|--------------|--------------|")

	for _, g := range groups {
		if g.setup != nil {
			if err := g.setup(); err != nil {
				t.Fatalf("[%s] 准备知识库失败: %v", g.name, err)
			}
		}

		totalScore, validJudgeCount, answerSuccessCount := runJudgeOnlyGroup(
			t,
			g.name,
			resumeEvalCases,
			client,
			judgeModels,
			g.fn,
		)

		if validJudgeCount == 0 {
			t.Logf("| %-12s | ERROR（无有效评分） | 0/%d         | %d/%d         |",
				g.name, len(resumeEvalCases), answerSuccessCount, len(resumeEvalCases))
			continue
		}

		n := float64(validJudgeCount)
		t.Logf("| %-12s | %.2f          | %d/%d         | %d/%d         |",
			g.name, totalScore/n, validJudgeCount, len(resumeEvalCases), answerSuccessCount, len(resumeEvalCases))
	}
}
