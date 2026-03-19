package svc

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"ai-gozero-agent/api/internal/config"
	"ai-gozero-agent/api/internal/types"
	"ai-gozero-agent/api/internal/utils"

	openai "github.com/sashabaranov/go-openai"
)

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

// 测试用配置，默认兼容本地直连，也允许在容器中通过环境变量切到 postgres 服务名。
var testCfg = config.VectorDBConfig{
	Host:           envOrDefault("RAG_EVAL_DB_HOST", "127.0.0.1"),
	Port:           envIntOrDefault("RAG_EVAL_DB_PORT", 5432),
	DBName:         envOrDefault("RAG_EVAL_DB_NAME", "hy_ai_agent"),
	User:           envOrDefault("RAG_EVAL_DB_USER", "root"),
	Password:       envOrDefault("RAG_EVAL_DB_PASSWORD", ""),
	MaxConn:        5,
	EmbeddingModel: "qwen3-vl-embedding",
	Knowledge: config.Knowledge{
		MaxChunkSize: 1000,
	},
}

var testOpenAIKey = os.Getenv("DASHSCOPE_API_KEY")
const testOpenAIBase = "https://dashscope.aliyuncs.com/compatible-mode/v1"
const testKnowledgeTitle = "__rag_test_goroutine__"

const testMarkdownDoc = `
## goroutine 基本概念
goroutine 是 Go 语言的轻量级线程，由 Go 运行时调度。初始栈约 2KB，可动态扩缩。

## goroutine 与线程的区别
操作系统线程需要 1~8MB 固定栈，goroutine 更轻量。调度由用户态运行时完成，避免系统调用开销。

## channel 通信机制
goroutine 之间通过 channel 通信，遵循 CSP 模型。channel 分有缓冲和无缓冲两种。
`

const testMarkdownDoc2 = `
## goroutine 基本概念
goroutine 是 Go 语言的轻量级线程，由 Go 运行时调度。初始栈约 2KB，可动态扩缩。

## goroutine 与线程的区别
操作系统线程需要 1~8MB 固定栈，goroutine 更轻量。调度由用户态运行时完成，避免系统调用开销。

## channel 通信机制
goroutine 之间通过 channel 通信，遵循 CSP 模型。channel 分有缓冲和无缓冲两种。

## 内存模型
Go 的内存模型定义了 goroutine 之间可见性规则，依赖 happens-before 关系。

## GC 与 goroutine
Go 使用三色标记 GC，STW 时间极短，goroutine 感知不到大多数 GC 停顿。
`

const testResumeText = `张三，5年Go开发经验。
曾在字节跳动负责推荐系统后端开发，主导了用户行为数据处理管道的重构，将吞吐量提升了3倍。
熟悉分布式系统设计，了解Raft共识算法原理。
业余时间喜欢打篮球，曾获校级三分球比赛冠军。
目前关注云原生方向，正在学习Kubernetes Operator开发。`

var multiQueries = []struct {
	query string
}{
	{"goroutine 和操作系统线程有什么区别"},
	{"Go 语言的并发原语是什么"},
	{"goroutine 的初始栈大小"},
	{"channel 是什么"},
	{"Go 的垃圾回收机制"},
}

// resumeMultiQueries 用于简历场景的多查询测试
var resumeMultiQueries = []struct {
	query string
}{
	{"候选人有哪些工作经历"},
	{"候选人熟悉哪些技术"},
	{"候选人的项目成果是什么"},
	{"候选人有什么分布式系统经验"},
	{"候选人目前关注什么方向"},
}

var thresholdPositiveQueries = []string{
	"goroutine 和线程的区别",
	"Go 并发模型",
	"channel 通信",
	"goroutine 初始内存",
	"Go 运行时调度",
	"有缓冲和无缓冲 channel",
	"happens-before 规则",
	"GC 停顿时间",
}

var thresholdNegativeQueries = []string{
	"Python 的 GIL 锁是什么",
	"Java Spring 框架介绍",
	"MySQL 索引优化方法",
	"今天天气怎么样",
}

func newTestStore(t *testing.T) *VectorStore {
	t.Helper()
	openaiCfg := openai.DefaultConfig(testOpenAIKey)
	openaiCfg.BaseURL = testOpenAIBase
	client := openai.NewClientWithConfig(openaiCfg)

	store, err := NewVectorStore(testCfg, client, testOpenAIKey, testOpenAIBase)
	if err != nil {
		t.Fatalf("连接向量数据库失败: %v", err)
	}
	return store
}

func cleanupKnowledgeTitle(t *testing.T, store *VectorStore, title string) {
	t.Helper()
	_, err := store.Pool.Exec(context.Background(),
		"DELETE FROM knowledge_base WHERE title = $1", title)
	if err != nil {
		t.Logf("清理测试数据失败（title=%s，可忽略）: %v", title, err)
	}
}

func cleanupTestData(t *testing.T, store *VectorStore) {
	t.Helper()
	cleanupKnowledgeTitle(t, store, testKnowledgeTitle)
}

// hitAtK 判断前 K 个结果中是否包含指定 title 的分块
func hitAtK(results []types.KnowledgeChunk, relevantTitle string, k int) bool {
	for i, r := range results {
		if i >= k {
			break
		}
		if r.Title == relevantTitle {
			return true
		}
	}
	return false
}

// precisionAtK 前 K 个结果中 title 匹配的比例
func precisionAtK(results []types.KnowledgeChunk, relevantTitle string, k int) float64 {
	if k <= 0 || len(results) == 0 {
		return 0
	}
	hit := 0
	for i, r := range results {
		if i >= k {
			break
		}
		if r.Title == relevantTitle {
			hit++
		}
	}
	limit := k
	if len(results) < k {
		limit = len(results)
	}
	return float64(hit) / float64(limit)
}

// reciprocalRank 第一个相关结果排名的倒数（未命中返回 0）
func reciprocalRank(results []types.KnowledgeChunk, relevantTitle string) float64 {
	for i, r := range results {
		if r.Title == relevantTitle {
			return 1.0 / float64(i+1)
		}
	}
	return 0
}

// lcs 最长公共子序列长度（用于 ROUGE-L）
func lcs(a, b []string) int {
	m, n := len(a), len(b)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] > dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}
	return dp[m][n]
}

// tokenizeChinese 简单分词：按字符拆开（适合中文），英文按空格
func tokenizeChinese(text string) []string {
	var tokens []string
	for _, r := range []rune(text) {
		tokens = append(tokens, string(r))
	}
	return tokens
}

// rougeL 计算 ROUGE-L F1 分数
func rougeL(hypothesis, reference string) float64 {
	h := tokenizeChinese(hypothesis)
	r := tokenizeChinese(reference)
	if len(h) == 0 || len(r) == 0 {
		return 0
	}
	l := lcs(h, r)
	precision := float64(l) / float64(len(h))
	recall := float64(l) / float64(len(r))
	if precision+recall == 0 {
		return 0
	}
	return 2 * precision * recall / (precision + recall)
}

// TestRAGSaveAndRetrieve 验证向量写入 + 检索全链路
func TestRAGSaveAndRetrieve(t *testing.T) {
	store := newTestStore(t)
	defer store.Pool.Close()

	// 确保清理旧测试数据，结束后也清理
	cleanupTestData(t, store)
	t.Cleanup(func() { cleanupTestData(t, store) })

	// 1. 写入测试知识
	testContent := "goroutine是Go语言的轻量级线程，由Go运行时调度器管理。" +
		"一个goroutine初始栈仅约2KB，可动态扩缩，而操作系统线程通常需要1~8MB固定栈。" +
		"goroutine之间通过channel通信，遵循CSP并发模型，避免共享内存竞争。" +
		"Go调度器使用M:N模型，将M个goroutine复用到N个OS线程上运行。"

	err := store.SaveKnowledge(testKnowledgeTitle, testContent, testCfg)
	if err != nil {
		t.Fatalf("SaveKnowledge 失败: %v", err)
	}
	t.Log("✓ 知识写入成功")

	// 2. 用语义相关的问题检索
	queries := []struct {
		query    string
		minScore float64
	}{
		{"goroutine和线程有什么区别？", 0.6},
		{"Go语言并发模型是什么", 0.55}, // text-embedding-v1 对此类短语相似度略低
		{"goroutine的内存开销", 0.6},
	}

	for _, tc := range queries {
		results, err := store.RetrieveKnowledge(tc.query, 3)
		if err != nil {
			t.Errorf("RetrieveKnowledge 失败 [%s]: %v", tc.query, err)
			continue
		}

		t.Logf("查询: %q", tc.query)
		for i, r := range results {
			t.Logf("  [%d] title=%-30s similarity=%.4f", i+1, r.Title, r.Similarity)
		}

		if len(results) == 0 {
			t.Errorf("查询 %q 未返回任何结果", tc.query)
			continue
		}

		found := false
		for _, r := range results {
			if r.Title == testKnowledgeTitle && r.Similarity >= tc.minScore {
				found = true
				t.Logf("  ✓ 命中测试数据，similarity=%.4f >= %.1f", r.Similarity, tc.minScore)
				break
			}
		}
		if !found {
			t.Errorf("查询 %q: 未能以 similarity >= %.1f 检索到测试知识片段", tc.query, tc.minScore)
		}
	}
}

// TestRAGSimilarityThreshold 验证 0.6 阈值过滤效果
func TestRAGSimilarityThreshold(t *testing.T) {
	store := newTestStore(t)
	defer store.Pool.Close()

	unrelatedQuery := "今天天气怎么样"
	results, err := store.RetrieveKnowledge(unrelatedQuery, 3)
	if err != nil {
		t.Fatalf("RetrieveKnowledge 失败: %v", err)
	}

	t.Logf("不相关查询 %q 的检索结果：", unrelatedQuery)
	highSimilarityCount := 0
	for i, r := range results {
		t.Logf("  [%d] title=%-30s similarity=%.4f", i+1, r.Title, r.Similarity)
		if r.Similarity >= 0.6 {
			highSimilarityCount++
		}
	}

	fmt.Printf("相似度 >= 0.6 的片段数量: %d（期望为 0 或很少）\n", highSimilarityCount)
}

func TestSplitterComparison(t *testing.T) {
	store := newTestStore(t)
	defer store.Pool.Close()

	openaiCfg := openai.DefaultConfig(testOpenAIKey)
	openaiCfg.BaseURL = testOpenAIBase
	client := openai.NewClientWithConfig(openaiCfg)

	cases := []struct {
		name     string
		content  string
		splitter utils.Splitter
		query    string
	}{
		{
			name:     "知识库-FixedSize",
			content:  testMarkdownDoc,
			splitter: &utils.FixedSizeSplitter{ChunkSize: 200},
			query:    "goroutine和线程有什么区别",
		},
		{
			name:     "知识库-Markdown",
			content:  testMarkdownDoc,
			splitter: &utils.MarkdownSplitter{FallbackChunkSize: 200},
			query:    "goroutine和线程有什么区别",
		},
		{
			name:    "知识库-Agentic",
			content: testMarkdownDoc,
			splitter: &utils.AgenticSplitter{
				Client: client,
				Model:  "qwen3.5-122b-a10b",
			},
			query: "goroutine和线程有什么区别",
		},
		{
			name:     "简历-FixedSize",
			content:  testResumeText,
			splitter: &utils.FixedSizeSplitter{ChunkSize: 100},
			query:    "候选人有什么分布式系统经验",
		},
		{
			name:     "简历-Markdown",
			content:  testResumeText,
			splitter: &utils.MarkdownSplitter{FallbackChunkSize: 100},
			query:    "候选人有什么分布式系统经验",
		},
		{
			name:    "简历-Agentic",
			content: testResumeText,
			splitter: &utils.AgenticSplitter{
				Client: client,
				Model:  "qwen3.5-122b-a10b",
			},
			query: "候选人有什么分布式系统经验",
		},
	}

	testTitle := "__splitter_comparison_test__"
	defer cleanupKnowledgeTitle(t, store, testTitle)

	t.Log("\n| 策略名称             | 分块数 | Top1相似度 | Hit@3 | P@3    | RR     |")
	t.Log("|----------------------|--------|-----------|-------|--------|--------|")
	for _, tc := range cases {
		cleanupKnowledgeTitle(t, store, testTitle)

		err := store.SaveKnowledge(testTitle, tc.content, testCfg, tc.splitter)
		if err != nil {
			t.Errorf("[%s] 写入失败: %v", tc.name, err)
			continue
		}

		var chunkCount int
		if err := store.Pool.QueryRow(context.Background(),
			"SELECT COUNT(*) FROM knowledge_base WHERE title = $1", testTitle).Scan(&chunkCount); err != nil {
			t.Errorf("[%s] 查询分块数失败: %v", tc.name, err)
			continue
		}

		results, err := store.RetrieveKnowledge(tc.query, 3)
		if err != nil {
			t.Errorf("[%s] 检索失败: %v", tc.name, err)
			continue
		}

		hit := hitAtK(results, testTitle, 3)
		p3 := precisionAtK(results, testTitle, 3)
		rr := reciprocalRank(results, testTitle)
		top1Sim := 0.0
		if len(results) > 0 {
			top1Sim = results[0].Similarity
		}

		t.Logf("| %-20s | %5d | %.4f | %v | %.4f | %.4f |",
			tc.name, chunkCount, top1Sim, hit, p3, rr)
	}
}

func TestSplitterMultiQuery(t *testing.T) {
	store := newTestStore(t)
	defer store.Pool.Close()

	openaiCfg := openai.DefaultConfig(testOpenAIKey)
	openaiCfg.BaseURL = testOpenAIBase
	client := openai.NewClientWithConfig(openaiCfg)

	splitters := []struct {
		name     string
		splitter utils.Splitter
	}{
		{"FixedSize(200)", &utils.FixedSizeSplitter{ChunkSize: 200}},
		{"Markdown(200)", &utils.MarkdownSplitter{FallbackChunkSize: 200}},
		{"Agentic(qwen3.5-122b-a10b)", &utils.AgenticSplitter{Client: client, Model: "qwen3.5-122b-a10b"}},
	}

	testTitles := datasetTitles(knowledgeEvalDatasets)
	defer cleanupKnowledgeTitles(t, store, testTitles...)
	queries := flattenRetrievalCases(knowledgeEvalDatasets)

	t.Log("\n| 策略名称            | 平均 Hit@3 | 平均 MRR  | 平均 Top1 相似度 |")
	t.Log("|---------------------|-----------|-----------|-----------------|")

	for _, sp := range splitters {
		cleanupKnowledgeTitles(t, store, testTitles...)

		if err := saveDatasetsWithSplitter(t, store, knowledgeEvalDatasets, sp.splitter); err != nil {
			t.Errorf("[%s] 写入失败: %v", sp.name, err)
			continue
		}

		totalHit, totalRR, totalSim := 0.0, 0.0, 0.0
		for _, q := range queries {
			results, err := store.RetrieveKnowledge(q.query, 3)
			if err != nil {
				t.Logf("[%s] query=%q 检索失败: %v", sp.name, q.query, err)
				continue
			}
			if hitAtK(results, q.relevantTitle, 3) {
				totalHit++
			}
			totalRR += reciprocalRank(results, q.relevantTitle)
			if len(results) > 0 {
				totalSim += results[0].Similarity
			}
		}

		n := float64(len(queries))
		t.Logf("| %-20s | %.4f     | %.4f     | %.4f           |",
			sp.name, totalHit/n, totalRR/n, totalSim/n)
	}
}

func TestRetrievalThresholdSensitivity(t *testing.T) {
	store := newTestStore(t)
	defer store.Pool.Close()

	testTitle := "__threshold_test__"
	defer cleanupKnowledgeTitle(t, store, testTitle)
	cleanupKnowledgeTitle(t, store, testTitle)

	if err := store.SaveKnowledge(testTitle, testMarkdownDoc2, testCfg,
		&utils.MarkdownSplitter{FallbackChunkSize: 200}); err != nil {
		t.Fatalf("写入失败: %v", err)
	}

	thresholds := []float64{0.40, 0.45, 0.50, 0.55, 0.60, 0.65, 0.70}

	t.Log("\n| 阈值  | TP | FP | FN | TN | Precision | Recall | F1     |")
	t.Log("|-------|----|----|----|----|-----------|--------|--------|")

	for _, threshold := range thresholds {
		tp, fp, fn, tn := 0, 0, 0, 0

		for _, q := range thresholdPositiveQueries {
			results, err := store.RetrieveKnowledge(q, 3)
			if err != nil {
				t.Logf("正例 query=%q 检索失败: %v", q, err)
				continue
			}
			retrieved := false
			for _, r := range results {
				if r.Similarity >= threshold {
					retrieved = true
					break
				}
			}
			if retrieved {
				tp++
			} else {
				fn++
			}
		}

		for _, q := range thresholdNegativeQueries {
			results, err := store.RetrieveKnowledge(q, 3)
			if err != nil {
				t.Logf("负例 query=%q 检索失败: %v", q, err)
				continue
			}
			retrieved := false
			for _, r := range results {
				if r.Similarity >= threshold {
					retrieved = true
					break
				}
			}
			if retrieved {
				fp++
			} else {
				tn++
			}
		}

		precision, recall, f1 := 0.0, 0.0, 0.0
		if tp+fp > 0 {
			precision = float64(tp) / float64(tp+fp)
		}
		if tp+fn > 0 {
			recall = float64(tp) / float64(tp+fn)
		}
		if precision+recall > 0 {
			f1 = 2 * precision * recall / (precision + recall)
		}

		t.Logf("| %.2f  | %2d | %2d | %2d | %2d | %.4f    | %.4f | %.4f |",
			threshold, tp, fp, fn, tn, precision, recall, f1)
	}
}

func TestChunkSizeImpact(t *testing.T) {
	store := newTestStore(t)
	defer store.Pool.Close()

	chunkSizes := []int{100, 200, 500, 800, 1000}
	queries := []string{
		"goroutine 和线程的区别",
		"channel 通信机制",
		"Go 内存模型",
	}

	testTitle := "__chunksize_test__"
	defer cleanupKnowledgeTitle(t, store, testTitle)

	t.Log("\n| ChunkSize | 分块数 | 平均 Hit@3 | 平均 Top1 相似度 | 平均分块长度(rune) |")
	t.Log("|-----------|--------|-----------|-----------------|------------------|")

	for _, size := range chunkSizes {
		cleanupKnowledgeTitle(t, store, testTitle)

		sp := &utils.FixedSizeSplitter{ChunkSize: size}
		if err := store.SaveKnowledge(testTitle, testMarkdownDoc2, testCfg, sp); err != nil {
			t.Errorf("ChunkSize=%d 写入失败: %v", size, err)
			continue
		}

		var chunkCount int
		if err := store.Pool.QueryRow(context.Background(),
			"SELECT COUNT(*) FROM knowledge_base WHERE title = $1", testTitle).Scan(&chunkCount); err != nil {
			t.Errorf("ChunkSize=%d 查询分块数失败: %v", size, err)
			continue
		}

		var avgLen float64
		if err := store.Pool.QueryRow(context.Background(),
			"SELECT AVG(LENGTH(content)) FROM knowledge_base WHERE title = $1", testTitle).Scan(&avgLen); err != nil {
			t.Errorf("ChunkSize=%d 查询平均分块长度失败: %v", size, err)
			continue
		}

		totalHit, totalSim := 0.0, 0.0
		for _, q := range queries {
			results, err := store.RetrieveKnowledge(q, 3)
			if err != nil {
				t.Logf("ChunkSize=%d query=%q 检索失败: %v", size, q, err)
				continue
			}
			if hitAtK(results, testTitle, 3) {
				totalHit++
			}
			if len(results) > 0 {
				totalSim += results[0].Similarity
			}
		}

		n := float64(len(queries))
		t.Logf("| %9d | %6d | %.4f     | %.4f           | %.1f              |",
			size, chunkCount, totalHit/n, totalSim/n, avgLen)
	}
}

func TestTopKImpact(t *testing.T) {
	store := newTestStore(t)
	defer store.Pool.Close()

	testTitle := "__topk_test__"
	defer cleanupKnowledgeTitle(t, store, testTitle)
	cleanupKnowledgeTitle(t, store, testTitle)

	if err := store.SaveKnowledge(testTitle, testMarkdownDoc2, testCfg,
		&utils.MarkdownSplitter{FallbackChunkSize: 200}); err != nil {
		t.Fatalf("写入失败: %v", err)
	}

	queries := []string{
		"goroutine 和线程的区别",
		"channel 通信机制",
		"Go 内存模型",
		"GC 停顿",
		"goroutine 初始栈",
	}

	topKValues := []int{1, 2, 3, 5, 10}

	t.Log("\n| TopK | 平均 Hit@K | 平均最高相似度 | 估算 Token 数 |")
	t.Log("|------|----------|------------|------------|")

	for _, k := range topKValues {
		totalHit, totalSim, totalTokens := 0.0, 0.0, 0.0
		for _, q := range queries {
			results, err := store.RetrieveKnowledge(q, k)
			if err != nil {
				t.Logf("TopK=%d query=%q 检索失败: %v", k, q, err)
				continue
			}
			if hitAtK(results, testTitle, k) {
				totalHit++
			}
			if len(results) > 0 {
				totalSim += results[0].Similarity
			}
			for _, r := range results {
				totalTokens += float64(len([]rune(r.Content))) / 1.5
			}
		}

		n := float64(len(queries))
		t.Logf("| %4d | %.4f     | %.4f       | %.0f          |",
			k, totalHit/n, totalSim/n, totalTokens/n)
	}
}

// TestResumeMultiQuery 简历场景多查询检索对比（FixedSize / Markdown / Agentic）
func TestResumeMultiQuery(t *testing.T) {
	store := newTestStore(t)
	defer store.Pool.Close()

	openaiCfg := openai.DefaultConfig(testOpenAIKey)
	openaiCfg.BaseURL = testOpenAIBase
	client := openai.NewClientWithConfig(openaiCfg)

	splitters := []struct {
		name     string
		splitter utils.Splitter
	}{
		{"FixedSize(100)", &utils.FixedSizeSplitter{ChunkSize: 100}},
		{"Markdown(100)", &utils.MarkdownSplitter{FallbackChunkSize: 100}},
		{"Agentic(qwen3.5-122b-a10b)", &utils.AgenticSplitter{Client: client, Model: "qwen3.5-122b-a10b"}},
	}

	testTitles := datasetTitles(resumeEvalDatasets)
	defer cleanupKnowledgeTitles(t, store, testTitles...)
	queries := flattenRetrievalCases(resumeEvalDatasets)

	t.Log("\n| 策略名称            | 平均 Hit@3 | 平均 MRR  | 平均 Top1 相似度 |")
	t.Log("|---------------------|-----------|-----------|-----------------|")

	for _, sp := range splitters {
		cleanupKnowledgeTitles(t, store, testTitles...)

		if err := saveDatasetsWithSplitter(t, store, resumeEvalDatasets, sp.splitter); err != nil {
			t.Errorf("[%s] 写入失败: %v", sp.name, err)
			continue
		}

		totalHit, totalRR, totalSim := 0.0, 0.0, 0.0
		for _, q := range queries {
			results, err := store.RetrieveKnowledge(q.query, 3)
			if err != nil {
				t.Logf("[%s] query=%q 检索失败: %v", sp.name, q.query, err)
				continue
			}
			if hitAtK(results, q.relevantTitle, 3) {
				totalHit++
			}
			totalRR += reciprocalRank(results, q.relevantTitle)
			if len(results) > 0 {
				totalSim += results[0].Similarity
			}
		}

		n := float64(len(queries))
		t.Logf("| %-20s | %.4f     | %.4f     | %.4f           |",
			sp.name, totalHit/n, totalRR/n, totalSim/n)
	}
}
