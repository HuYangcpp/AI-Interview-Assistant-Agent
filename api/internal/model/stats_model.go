package model

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DashboardStats holds the top-level summary counts returned by the admin dashboard.
type DashboardStats struct {
	TotalStudents    int64
	TotalEmployments int64
	EmployedStudents int64
	EmploymentRate   float64
}

// EmploymentRateItem is one row in the by-status breakdown.
type EmploymentRateItem struct {
	Status string
	Count  int64
}

// EmploymentRate holds the full by-status breakdown.
type EmploymentRate struct {
	TotalStudents  int64
	EmploymentRate float64
	Items          []*EmploymentRateItem
}

// MajorStatItem is one row in the per-major breakdown.
type MajorStatItem struct {
	Major            string
	TotalStudents    int64
	EmployedStudents int64
}

// TrendPoint is one month-bucket in the employment trend series.
type TrendPoint struct {
	Month            string
	EmployedStudents int64
}

// StatsModel encapsulates all dashboard / statistics queries.
type StatsModel struct {
	pool *pgxpool.Pool
}

func NewStatsModel(pool *pgxpool.Pool) *StatsModel {
	return &StatsModel{pool: pool}
}

func (m *StatsModel) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	const totalStudentsQ = `SELECT COUNT(1) FROM students`
	const totalEmploymentsQ = `SELECT COUNT(1) FROM employments`
	const employedStudentsQ = `
WITH latest AS (
  SELECT DISTINCT ON (student_id) student_id, status, verification_status
  FROM employments
  ORDER BY student_id, updated_at DESC NULLS LAST, id DESC
)
SELECT COUNT(1)
FROM students s
LEFT JOIN latest l ON l.student_id = s.id
WHERE COALESCE(CASE WHEN l.verification_status = 'approved' THEN l.status ELSE 'seeking' END, 'seeking') <> 'seeking'
`

	var s DashboardStats
	if err := m.pool.QueryRow(ctx, totalStudentsQ).Scan(&s.TotalStudents); err != nil {
		return nil, err
	}
	if err := m.pool.QueryRow(ctx, totalEmploymentsQ).Scan(&s.TotalEmployments); err != nil {
		return nil, err
	}
	if err := m.pool.QueryRow(ctx, employedStudentsQ).Scan(&s.EmployedStudents); err != nil {
		return nil, err
	}
	if s.TotalStudents > 0 {
		s.EmploymentRate = float64(s.EmployedStudents) / float64(s.TotalStudents)
	}
	return &s, nil
}

func (m *StatsModel) GetEmploymentRate(ctx context.Context) (*EmploymentRate, error) {
	const totalStudentsQ = `SELECT COUNT(1) FROM students`
	const byStatusQ = `
WITH latest AS (
  SELECT DISTINCT ON (student_id) student_id, status, verification_status
  FROM employments
  ORDER BY student_id, updated_at DESC NULLS LAST, id DESC
)
SELECT COALESCE(CASE WHEN latest.verification_status = 'approved' THEN latest.status ELSE 'seeking' END, 'seeking') AS status,
       COUNT(1)
FROM students s
LEFT JOIN latest ON latest.student_id = s.id
GROUP BY 1
`

	var result EmploymentRate
	if err := m.pool.QueryRow(ctx, totalStudentsQ).Scan(&result.TotalStudents); err != nil {
		return nil, err
	}

	rows, err := m.pool.Query(ctx, byStatusQ)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employed int64
	for rows.Next() {
		var item EmploymentRateItem
		if err := rows.Scan(&item.Status, &item.Count); err != nil {
			return nil, err
		}
		result.Items = append(result.Items, &item)
		if item.Status != "seeking" {
			employed += item.Count
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if result.TotalStudents > 0 {
		result.EmploymentRate = float64(employed) / float64(result.TotalStudents)
	}
	return &result, nil
}

func (m *StatsModel) GetMajorStats(ctx context.Context) ([]*MajorStatItem, error) {
	const query = `
WITH latest AS (
  SELECT DISTINCT ON (student_id) student_id, status, verification_status
  FROM employments
  ORDER BY student_id, updated_at DESC NULLS LAST, id DESC
)
SELECT COALESCE(s.major,''), COUNT(1) AS total_students,
       COUNT(1) FILTER (
           WHERE COALESCE(CASE WHEN latest.verification_status = 'approved' THEN latest.status ELSE 'seeking' END, 'seeking') <> 'seeking'
       ) AS employed_students
FROM students s
LEFT JOIN latest ON latest.student_id = s.id
GROUP BY s.major
ORDER BY total_students DESC
`

	rows, err := m.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*MajorStatItem
	for rows.Next() {
		var item MajorStatItem
		if err := rows.Scan(&item.Major, &item.TotalStudents, &item.EmployedStudents); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}
	return items, rows.Err()
}

func (m *StatsModel) GetEmploymentTrend(ctx context.Context, months int) ([]*TrendPoint, error) {
	if months <= 0 {
		months = 12
	}
	const query = `
SELECT to_char(date_trunc('month', reviewed_at), 'YYYY-MM') AS month,
       COUNT(1) FILTER (WHERE status <> 'seeking') AS employed_records
FROM employments
WHERE verification_status = 'approved'
  AND reviewed_at IS NOT NULL
  AND reviewed_at >= NOW() - make_interval(months => $1)
GROUP BY 1
ORDER BY 1 ASC
`

	rows, err := m.pool.Query(ctx, query, months)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*TrendPoint
	for rows.Next() {
		var item TrendPoint
		if err := rows.Scan(&item.Month, &item.EmployedStudents); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}
	return items, rows.Err()
}
