package logic

import (
	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	stateKeyPrefix = "chat_state:"
	stateTTL       = 24 * time.Hour
)

var stateTagRegex = regexp.MustCompile(`\[STATE:(QUESTION|FOLLOWUP|EVALUATE|END)\]`)

var validStateTags = []string{
	"[STATE:QUESTION]",
	"[STATE:FOLLOWUP]",
	"[STATE:EVALUATE]",
	"[STATE:END]",
}

type StateManager struct {
	svcCtx *svc.ServiceContext
}

func NewStateManager(svcCtx *svc.ServiceContext) *StateManager {
	return &StateManager{svcCtx: svcCtx}
}

// 获取当前状态（带初始化）
func (sm *StateManager) GetOrInitState(chatId string) (string, error) {
	key := stateKeyPrefix + chatId

	// 尝试获取状态
	state, err := sm.svcCtx.Redis.Get(context.Background(), key).Result()
	if err == nil {
		return state, nil
	}

	// 如果状态不存在或出错，初始化状态
	if err == redis.Nil {
		if err := sm.svcCtx.Redis.Set(
			context.Background(),
			key,
			types.StateStart,
			stateTTL,
		).Err(); err != nil {
			return types.StateStart, fmt.Errorf("初始化状态失败: %w", err)
		}
		return types.StateStart, nil
	}

	return types.StateStart, fmt.Errorf("获取状态失败: %w", err)
}

// 强制设置状态
func (sm *StateManager) SetState(chatId, state string) error {
	key := stateKeyPrefix + chatId
	if err := sm.svcCtx.Redis.Set(
		context.Background(),
		key,
		state,
		stateTTL,
	).Err(); err != nil {
		return fmt.Errorf("设置状态失败: %w", err)
	}
	return nil
}

// 评估并更新状态（更智能的规则）
func (sm *StateManager) EvaluateAndUpdateState(chatId, aiResponse string) (string, error) {
	currentState, err := sm.GetOrInitState(chatId)
	if err != nil {
		return currentState, err
	}

	newState := sm.determineNewState(currentState, aiResponse)

	if newState != currentState {
		if err := sm.SetState(chatId, newState); err != nil {
			return currentState, err
		}
	}

	return newState, nil
}

// 状态转移决策逻辑
func (sm *StateManager) determineNewState(currentState, aiResponse string) string {
	matches := stateTagRegex.FindStringSubmatch(aiResponse)
	if len(matches) < 2 {
		return currentState
	}

	switch matches[1] {
	case "QUESTION":
		return types.StateQuestion
	case "FOLLOWUP":
		return types.StateFollowUp
	case "EVALUATE":
		return types.StateEvaluate
	case "END":
		return types.StateEnd
	default:
		return currentState
	}
}

func stripStateTags(content string) string {
	return stateTagRegex.ReplaceAllString(content, "")
}

func isStateTagPrefix(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}
	for _, tag := range validStateTags {
		if strings.HasPrefix(tag, trimmed) {
			return true
		}
	}
	return false
}
