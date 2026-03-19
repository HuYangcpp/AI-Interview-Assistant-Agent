package utils

import (
	"context"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// Splitter 分块策略接口
type Splitter interface {
	Split(text string) []string
}

// FixedSizeSplitter 使用固定字符数切块。
type FixedSizeSplitter struct {
	ChunkSize int
}

func (s *FixedSizeSplitter) Split(text string) []string {
	return SplitText(text, s.ChunkSize)
}

// MarkdownSplitter 按二级标题切割结构化文档。
type MarkdownSplitter struct {
	FallbackChunkSize int
}

func (s *MarkdownSplitter) Split(text string) []string {
	lines := strings.Split(text, "\n")

	var chunks []string
	var current strings.Builder
	hasH2 := false

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
			hasH2 = true
			if current.Len() > 0 {
				chunk := strings.TrimSpace(current.String())
				if chunk != "" {
					chunks = append(chunks, chunk)
				}
				current.Reset()
			}
		}
		current.WriteString(line)
		current.WriteString("\n")
	}

	if current.Len() > 0 {
		chunk := strings.TrimSpace(current.String())
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
	}

	if !hasH2 {
		fallback := &FixedSizeSplitter{ChunkSize: s.FallbackChunkSize}
		return fallback.Split(text)
	}

	return chunks
}

// AgenticSplitter 使用小模型判断话题切换边界。
type AgenticSplitter struct {
	Client *openai.Client
	Model  string
}

func (s *AgenticSplitter) Split(text string) []string {
	sentences := splitIntoSentences(text)
	if len(sentences) <= 1 {
		return sentences
	}

	var chunks []string
	current := sentences[0]

	for i := 1; i < len(sentences); i++ {
		if s.isTopicShift(current, sentences[i]) {
			chunks = append(chunks, strings.TrimSpace(current))
			current = sentences[i]
		} else {
			current += sentences[i]
		}
	}
	if strings.TrimSpace(current) != "" {
		chunks = append(chunks, strings.TrimSpace(current))
	}

	return chunks
}

func splitIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for _, r := range []rune(text) {
		current.WriteRune(r)
		if r == '。' || r == '！' || r == '？' || r == '.' || r == '!' || r == '?' || r == '\n' {
			sentence := strings.TrimSpace(current.String())
			if sentence != "" {
				sentences = append(sentences, sentence)
			}
			current.Reset()
		}
	}

	if remaining := strings.TrimSpace(current.String()); remaining != "" {
		sentences = append(sentences, remaining)
	}

	var result []string
	for _, sentence := range sentences {
		if len([]rune(sentence)) >= 5 {
			result = append(result, sentence)
		}
	}
	return result
}

func (s *AgenticSplitter) isTopicShift(sentA, sentB string) bool {
	if s == nil || s.Client == nil || s.Model == "" {
		return false
	}

	prompt := "判断以下两句话的话题是否发生了明显转换。只回答 yes 或 no，不要解释。\n句子A：" + sentA + "\n句子B：" + sentB

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := s.Client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: s.Model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		MaxTokens:   5,
		Temperature: 0,
	})
	if err != nil || len(resp.Choices) == 0 {
		return false
	}

	answer := strings.ToLower(strings.TrimSpace(resp.Choices[0].Message.Content))
	return strings.HasPrefix(answer, "yes")
}
