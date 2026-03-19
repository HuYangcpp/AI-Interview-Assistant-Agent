package types

type AdminEmploymentItem struct {
	ID                 int64                     `json:"id"`
	StudentID          int64                     `json:"studentId"`
	StudentNo          string                    `json:"studentNo"`
	RealName           string                    `json:"realName"`
	Major              string                    `json:"major"`
	Status             string                    `json:"status"`
	VerificationStatus string                    `json:"verificationStatus"`
	ReviewComment      string                    `json:"reviewComment"`
	CompanyName        string                    `json:"companyName"`
	Position           string                    `json:"position"`
	City               string                    `json:"city"`
	SalaryRange        string                    `json:"salaryRange"`
	OfferDate          string                    `json:"offerDate,omitempty"`
	EntryDate          string                    `json:"entryDate,omitempty"`
	ReviewedAt         string                    `json:"reviewedAt,omitempty"`
	UpdatedAt          string                    `json:"updatedAt"`
	Notes              string                    `json:"notes"`
	Evidences          []*EmploymentEvidenceResp `json:"evidences"`
}

type AdminUpdateEmploymentStatusReq struct {
	Status string `json:"status"`
}

type AdminReviewEmploymentReq struct {
	VerificationStatus string `json:"verificationStatus"`
	ReviewComment      string `json:"reviewComment"`
}
