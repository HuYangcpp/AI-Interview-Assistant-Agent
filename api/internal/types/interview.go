package types

type CreateInterviewSessionReq struct {
	Title         string `json:"title"`
	InterviewType string `json:"interviewType"`
	CustomPrompt  string `json:"customPrompt,optional"`
}

type InterviewSessionItem struct {
	ID              int64    `json:"id"`
	Title           string   `json:"title"`
	InterviewType   string   `json:"interviewType"`
	Status          string   `json:"status"`
	TotalQuestions  int      `json:"totalQuestions"`
	MessageCount    int      `json:"messageCount"`
	DurationSeconds int      `json:"durationSeconds"`
	Score           *float64 `json:"score,omitempty"`
	CreatedAt       string   `json:"createdAt"`
	CompletedAt     string   `json:"completedAt,omitempty"`
}

type CreateInterviewSessionResp struct {
	Session InterviewSessionItem `json:"session"`
}

type InterviewSessionDetailResp struct {
	InterviewSessionItem
	ChatID      string `json:"chatId"`
	ScoreDetail any    `json:"scoreDetail,omitempty"`
	AISummary   string `json:"aiSummary"`
}

type InterviewChatReq struct {
	Message   string `form:"message"`
	SessionId int64  `form:"sessionId"`
}
