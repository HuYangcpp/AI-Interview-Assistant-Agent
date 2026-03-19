package types

import "time"

type StudentCreateProfileChangeRequestReq struct {
	StudentNo      string `json:"studentNo"`
	Major          string `json:"major"`
	ClassName      string `json:"className"`
	GraduationYear int    `json:"graduationYear"`
	Reason         string `json:"reason"`
}

type StudentProfileChangeRequestItem struct {
	ID                      int64      `json:"id"`
	RequestedStudentNo      string     `json:"requestedStudentNo"`
	RequestedMajor          string     `json:"requestedMajor"`
	RequestedClassName      string     `json:"requestedClassName"`
	RequestedGraduationYear int        `json:"requestedGraduationYear"`
	Reason                  string     `json:"reason"`
	Status                  string     `json:"status"`
	ReviewComment           string     `json:"reviewComment"`
	CreatedAt               time.Time  `json:"createdAt"`
	ReviewedAt              *time.Time `json:"reviewedAt,omitempty"`
}

type AdminProfileChangeRequestItem struct {
	ID                      int64      `json:"id"`
	StudentId               int64      `json:"studentId"`
	Username                string     `json:"username"`
	RealName                string     `json:"realName"`
	CurrentStudentNo        string     `json:"currentStudentNo"`
	CurrentMajor            string     `json:"currentMajor"`
	CurrentClassName        string     `json:"currentClassName"`
	CurrentGraduationYear   int        `json:"currentGraduationYear"`
	RequestedStudentNo      string     `json:"requestedStudentNo"`
	RequestedMajor          string     `json:"requestedMajor"`
	RequestedClassName      string     `json:"requestedClassName"`
	RequestedGraduationYear int        `json:"requestedGraduationYear"`
	Reason                  string     `json:"reason"`
	Status                  string     `json:"status"`
	ReviewComment           string     `json:"reviewComment"`
	ReviewedByName          string     `json:"reviewedByName"`
	CreatedAt               time.Time  `json:"createdAt"`
	ReviewedAt              *time.Time `json:"reviewedAt,omitempty"`
}

type AdminReviewProfileChangeRequestReq struct {
	Status        string `json:"status"`
	ReviewComment string `json:"reviewComment"`
}
