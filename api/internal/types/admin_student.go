package types

type AdminStudentListReq struct {
	Page    int64  `form:"page,optional"`
	Size    int64  `form:"size,optional"`
	Keyword string `form:"keyword,optional"`
}

type AdminStudentListItem struct {
	StudentId      int64  `json:"studentId"`
	UserId         int64  `json:"userId"`
	Username       string `json:"username"`
	RealName       string `json:"realName"`
	Phone          string `json:"phone"`
	Email          string `json:"email"`
	Status         int16  `json:"status"`
	StudentNo      string `json:"studentNo"`
	Major          string `json:"major"`
	ClassName      string `json:"className"`
	GraduationYear int    `json:"graduationYear"`
}

type AdminStudentListResp struct {
	Items []*AdminStudentListItem `json:"items"`
	Total int64                   `json:"total"`
	Page  int64                   `json:"page"`
	Size  int64                   `json:"size"`
}

type AdminStudentDetailResp struct {
	AdminStudentListItem
	Skills           []string `json:"skills"`
	SelfIntroduction string   `json:"selfIntroduction"`
	ResumeURL        string   `json:"resumeUrl"`
}

type AdminCreateStudentReq struct {
	Username       string   `json:"username"`
	Password       string   `json:"password"`
	StudentNo      string   `json:"studentNo"`
	RealName       string   `json:"realName"`
	Phone          string   `json:"phone"`
	Email          string   `json:"email"`
	Status         int16    `json:"status"`
	Major          string   `json:"major"`
	ClassName      string   `json:"className"`
	GraduationYear int      `json:"graduationYear"`
	Skills         []string `json:"skills"`
}

type AdminCreateStudentResp struct {
	ID              int64  `json:"id"`
	InitialPassword string `json:"initialPassword,omitempty"`
}

type AdminUpdateStudentReq struct {
	RealName         string   `json:"realName"`
	Phone            string   `json:"phone"`
	Email            string   `json:"email"`
	Status           int16    `json:"status"`
	StudentNo        string   `json:"studentNo"`
	Major            string   `json:"major"`
	ClassName        string   `json:"className"`
	GraduationYear   int      `json:"graduationYear"`
	Skills           []string `json:"skills"`
	SelfIntroduction string   `json:"selfIntroduction"`
}

type AdminImportStudentsResp struct {
	Count int `json:"count"`
}
