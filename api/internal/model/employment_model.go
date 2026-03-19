package model

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Employment struct {
	ID                 int64
	StudentID          int64
	Status             string
	VerificationStatus string
	ReviewComment      string
	ReviewerID         *int64
	CompanyName        string
	Position           string
	SalaryRange        string
	City               string
	OfferDate          *time.Time
	EntryDate          *time.Time
	ReviewedAt         *time.Time
	Notes              string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type EmploymentModel struct {
	pool *pgxpool.Pool
}

func NewEmploymentModel(pool *pgxpool.Pool) *EmploymentModel {
	return &EmploymentModel{pool: pool}
}

func (m *EmploymentModel) FindById(ctx context.Context, id int64) (*Employment, error) {
	const query = `SELECT id, student_id, status, verification_status, review_comment, reviewer_id,
company_name, position, salary_range, city, offer_date, entry_date, reviewed_at, notes, created_at, updated_at
FROM employments WHERE id=$1`

	var e Employment
	err := m.pool.QueryRow(ctx, query, id).Scan(
		&e.ID,
		&e.StudentID,
		&e.Status,
		&e.VerificationStatus,
		&e.ReviewComment,
		&e.ReviewerID,
		&e.CompanyName,
		&e.Position,
		&e.SalaryRange,
		&e.City,
		&e.OfferDate,
		&e.EntryDate,
		&e.ReviewedAt,
		&e.Notes,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (m *EmploymentModel) FindByStudentId(ctx context.Context, studentId int64) ([]*Employment, error) {
	const query = `SELECT id, student_id, status, verification_status, review_comment, reviewer_id,
company_name, position, salary_range, city, offer_date, entry_date, reviewed_at, notes, created_at, updated_at
FROM employments WHERE student_id=$1 ORDER BY id DESC`

	rows, err := m.pool.Query(ctx, query, studentId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Employment
	for rows.Next() {
		var e Employment
		if err := rows.Scan(
			&e.ID,
			&e.StudentID,
			&e.Status,
			&e.VerificationStatus,
			&e.ReviewComment,
			&e.ReviewerID,
			&e.CompanyName,
			&e.Position,
			&e.SalaryRange,
			&e.City,
			&e.OfferDate,
			&e.EntryDate,
			&e.ReviewedAt,
			&e.Notes,
			&e.CreatedAt,
			&e.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, &e)
	}

	return list, rows.Err()
}

func (m *EmploymentModel) Insert(ctx context.Context, e *Employment) (int64, error) {
	const query = `INSERT INTO employments (student_id, status, verification_status, company_name, position, salary_range, city, offer_date, entry_date, notes)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
RETURNING id`

	var id int64
	err := m.pool.QueryRow(ctx, query,
		e.StudentID,
		e.Status,
		e.VerificationStatus,
		e.CompanyName,
		e.Position,
		e.SalaryRange,
		e.City,
		e.OfferDate,
		e.EntryDate,
		e.Notes,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (m *EmploymentModel) UpdateByStudent(ctx context.Context, e *Employment) error {
	const query = `UPDATE employments
SET status=$3, company_name=$4, position=$5, salary_range=$6, city=$7, offer_date=$8, entry_date=$9, notes=$10,
    verification_status='pending', review_comment='', reviewer_id=NULL, reviewed_at=NULL, updated_at=NOW()
WHERE id=$1 AND student_id=$2`

	ct, err := m.pool.Exec(ctx, query,
		e.ID,
		e.StudentID,
		e.Status,
		e.CompanyName,
		e.Position,
		e.SalaryRange,
		e.City,
		e.OfferDate,
		e.EntryDate,
		e.Notes,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *EmploymentModel) DeleteByStudent(ctx context.Context, id, studentId int64) error {
	const query = `DELETE FROM employments WHERE id=$1 AND student_id=$2`
	ct, err := m.pool.Exec(ctx, query, id, studentId)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *EmploymentModel) ResetVerification(ctx context.Context, id int64) error {
	const query = `UPDATE employments
SET verification_status='pending', review_comment='', reviewer_id=NULL, reviewed_at=NULL, updated_at=NOW()
WHERE id=$1`
	ct, err := m.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *EmploymentModel) UpdateReview(ctx context.Context, id int64, verificationStatus, reviewComment string, reviewerID *int64, reviewedAt *time.Time) error {
	const query = `UPDATE employments
SET verification_status=$2, review_comment=$3, reviewer_id=$4, reviewed_at=$5, updated_at=NOW()
WHERE id=$1`
	ct, err := m.pool.Exec(ctx, query, id, verificationStatus, reviewComment, reviewerID, reviewedAt)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *EmploymentModel) DeleteByStudentId(ctx context.Context, studentId int64) error {
	const query = `DELETE FROM employments WHERE student_id=$1`
	_, err := m.pool.Exec(ctx, query, studentId)
	return err
}
