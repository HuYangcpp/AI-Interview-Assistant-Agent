package types

type AdminDashboardStatsResp struct {
	TotalStudents    int64   `json:"totalStudents"`
	TotalEmployments int64   `json:"totalEmployments"`
	EmployedStudents int64   `json:"employedStudents"`
	EmploymentRate   float64 `json:"employmentRate"`
}

type AdminEmploymentRateItem struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type AdminEmploymentRateResp struct {
	TotalStudents  int64                      `json:"totalStudents"`
	EmploymentRate float64                    `json:"employmentRate"`
	Items          []*AdminEmploymentRateItem `json:"items"`
}

type AdminMajorStatItem struct {
	Major            string `json:"major"`
	TotalStudents    int64  `json:"totalStudents"`
	EmployedStudents int64  `json:"employedStudents"`
}

type AdminTrendPoint struct {
	Month            string `json:"month"`
	EmployedStudents int64  `json:"employedStudents"`
}
