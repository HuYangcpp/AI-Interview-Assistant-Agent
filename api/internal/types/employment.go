package types

type EmploymentEvidenceResp struct {
	ID        int64  `json:"id"`
	FileURL   string `json:"fileUrl"`
	FileName  string `json:"fileName"`
	MimeType  string `json:"mimeType"`
	CreatedAt string `json:"createdAt"`
}

type EmploymentResp struct {
	ID                 int64                     `json:"id"`
	Status             string                    `json:"status"`
	VerificationStatus string                    `json:"verificationStatus"`
	ReviewComment      string                    `json:"reviewComment"`
	CompanyName        string                    `json:"companyName"`
	Position           string                    `json:"position"`
	SalaryRange        string                    `json:"salaryRange"`
	City               string                    `json:"city"`
	OfferDate          string                    `json:"offerDate,omitempty"`
	EntryDate          string                    `json:"entryDate,omitempty"`
	Notes              string                    `json:"notes"`
	ReviewedAt         string                    `json:"reviewedAt,omitempty"`
	Evidences          []*EmploymentEvidenceResp `json:"evidences"`
	CreatedAt          string                    `json:"createdAt"`
	UpdatedAt          string                    `json:"updatedAt"`
}

type CreateEmploymentReq struct {
	Status      string `json:"status"`
	CompanyName string `json:"companyName"`
	Position    string `json:"position"`
	SalaryRange string `json:"salaryRange"`
	City        string `json:"city"`
	OfferDate   string `json:"offerDate,omitempty"`
	EntryDate   string `json:"entryDate,omitempty"`
	Notes       string `json:"notes"`
}

type EmploymentReviewResp struct {
	VerificationStatus string `json:"verificationStatus"`
	ReviewComment      string `json:"reviewComment"`
	ReviewedAt         string `json:"reviewedAt,omitempty"`
}

type UpdateEmploymentReq struct {
	Status      string `json:"status"`
	CompanyName string `json:"companyName"`
	Position    string `json:"position"`
	SalaryRange string `json:"salaryRange"`
	City        string `json:"city"`
	OfferDate   string `json:"offerDate,omitempty"`
	EntryDate   string `json:"entryDate,omitempty"`
	Notes       string `json:"notes"`
}
