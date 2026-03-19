package types

type KnowledgeUploadReq struct {
	Title        string `form:"title"`         // 知识标题
	Content      string `form:"content"`       // 知识内容（由后端提取，前端不用传）
	SplitterType string `form:"splitter_type"` // "fixed" | "markdown" | "agentic"
}

type KnowledgeUploadResp struct {
	Msg    string `json:"msg"`
	Chunks int    `json:"chunks"` // 保存的知识块数量
}
