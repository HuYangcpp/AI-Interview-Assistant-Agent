package model

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EmploymentEvidence struct {
	ID           int64
	EmploymentID int64
	FileURL      string
	FileName     string
	MimeType     string
	UploadedBy   int64
	CreatedAt    time.Time
}

type EmploymentEvidenceModel struct {
	pool *pgxpool.Pool
}

func NewEmploymentEvidenceModel(pool *pgxpool.Pool) *EmploymentEvidenceModel {
	return &EmploymentEvidenceModel{pool: pool}
}

func (m *EmploymentEvidenceModel) FindByEmploymentID(ctx context.Context, employmentID int64) ([]*EmploymentEvidence, error) {
	const query = `SELECT id, employment_id, file_url, file_name, mime_type, uploaded_by, created_at
FROM employment_evidences
WHERE employment_id=$1
ORDER BY id DESC`

	rows, err := m.pool.Query(ctx, query, employmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*EmploymentEvidence
	for rows.Next() {
		var item EmploymentEvidence
		if err := rows.Scan(
			&item.ID,
			&item.EmploymentID,
			&item.FileURL,
			&item.FileName,
			&item.MimeType,
			&item.UploadedBy,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, &item)
	}

	return list, rows.Err()
}

func (m *EmploymentEvidenceModel) Insert(ctx context.Context, item *EmploymentEvidence) (int64, error) {
	const query = `INSERT INTO employment_evidences (employment_id, file_url, file_name, mime_type, uploaded_by)
VALUES ($1,$2,$3,$4,$5)
RETURNING id`

	var id int64
	err := m.pool.QueryRow(ctx, query,
		item.EmploymentID,
		item.FileURL,
		item.FileName,
		item.MimeType,
		item.UploadedBy,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (m *EmploymentEvidenceModel) FindByID(ctx context.Context, id int64) (*EmploymentEvidence, error) {
	const query = `SELECT id, employment_id, file_url, file_name, mime_type, uploaded_by, created_at
FROM employment_evidences
WHERE id=$1`

	var item EmploymentEvidence
	err := m.pool.QueryRow(ctx, query, id).Scan(
		&item.ID,
		&item.EmploymentID,
		&item.FileURL,
		&item.FileName,
		&item.MimeType,
		&item.UploadedBy,
		&item.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (m *EmploymentEvidenceModel) DeleteByEmploymentID(ctx context.Context, evidenceID, employmentID int64) error {
	const query = `DELETE FROM employment_evidences WHERE id=$1 AND employment_id=$2`
	ct, err := m.pool.Exec(ctx, query, evidenceID, employmentID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
