package types

type StudentProfileResp struct {
	StudentId        int64    `json:"studentId"`
	UserId           int64    `json:"userId"`
	Username         string   `json:"username"`
	RealName         string   `json:"realName"`
	Phone            string   `json:"phone"`
	Email            string   `json:"email"`
	StudentNo        string   `json:"studentNo"`
	Major            string   `json:"major"`
	ClassName        string   `json:"className"`
	GraduationYear   int      `json:"graduationYear"`
	Skills           []string `json:"skills"`
	SelfIntroduction string   `json:"selfIntroduction"`
	ResumeURL        string   `json:"resumeUrl"`
	ResumeMode       string   `json:"resumeMode"`
	ResumeSummary    string   `json:"resumeSummary"`
}

type UpdateStudentProfileReq struct {
	RealName         string   `json:"realName"`
	Phone            string   `json:"phone"`
	Email            string   `json:"email"`
	Skills           []string `json:"skills"`
	SelfIntroduction string   `json:"selfIntroduction"`
}

type ResumeUploadResp struct {
	ResumeURL string `json:"resumeUrl"`
}
