package types

type AnalysisDetailResp struct {
	SessionID          int64             `json:"sessionId"`
	StudentID          int64             `json:"studentId"`
	SessionTitle       string            `json:"sessionTitle"`
	OverallScore       float64           `json:"overallScore"`
	TechnicalScore     float64           `json:"technicalScore"`
	ExpressionScore    float64           `json:"expressionScore"`
	LogicScore         float64           `json:"logicScore"`
	Strengths          []string          `json:"strengths"`
	Weaknesses         []string          `json:"weaknesses"`
	Suggestions        string            `json:"suggestions"`
	DetailReport       string            `json:"detailReport"`
	SessionSuggestions []*SuggestionItem `json:"sessionSuggestions"`
	CreatedAt          string            `json:"createdAt"`
}

type AnalysisOverviewSession struct {
	ID            int64   `json:"id"`
	Title         string  `json:"title"`
	Date          string  `json:"date"`
	OverallScore  float64 `json:"overallScore"`
	InterviewType string  `json:"interviewType"`
}

type AnalysisOverviewResp struct {
	TotalCompleted int64                      `json:"totalCompleted"`
	AvgOverall     float64                    `json:"avgOverall"`
	AvgTechnical   float64                    `json:"avgTechnical"`
	AvgExpression  float64                    `json:"avgExpression"`
	AvgLogic       float64                    `json:"avgLogic"`
	Sessions       []*AnalysisOverviewSession `json:"sessions"`
}

type AnalysisTrendPoint struct {
	CreatedAt    string  `json:"createdAt"`
	OverallScore float64 `json:"overallScore"`
}
