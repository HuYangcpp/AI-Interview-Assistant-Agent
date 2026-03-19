package types

type SuggestionItem struct {
	ID             int64  `json:"id"`
	SessionID      int64  `json:"sessionId,omitempty"`
	SessionTitle   string `json:"sessionTitle,omitempty"`
	SessionType    string `json:"sessionType,omitempty"`
	SessionCreated string `json:"sessionCreated,omitempty"`
	SuggestionType string `json:"suggestionType"`
	Content        string `json:"content"`
	IsRead         bool   `json:"isRead"`
	CreatedAt      string `json:"createdAt"`
}

type GenerateSuggestionsResp struct {
	Created int `json:"created"`
}
