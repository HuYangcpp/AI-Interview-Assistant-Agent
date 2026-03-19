package model

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Student struct {
	ID                int64
	UserID            int64
	StudentNo         string
	Major             string
	ClassName         string
	GraduationYear    int
	Skills            string
	SelfIntroduction  string
	ResumeURL         string
	ResumeMode        string
	ResumeLength      int
	ResumeSummaryJSON string
	ResumeSummaryText string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type StudentModel struct {
	pool *pgxpool.Pool
}

func NewStudentModel(pool *pgxpool.Pool) *StudentModel {
	return &StudentModel{pool: pool}
}

func (m *StudentModel) FindById(ctx context.Context, id int64) (*Student, error) {
	const query = `SELECT id, user_id, student_no, major, class_name, graduation_year, skills, self_introduction, resume_url, resume_mode, resume_length, resume_summary_json, resume_summary_text, created_at, updated_at
FROM students WHERE id=$1`

	var s Student
	err := m.pool.QueryRow(ctx, query, id).Scan(
		&s.ID,
		&s.UserID,
		&s.StudentNo,
		&s.Major,
		&s.ClassName,
		&s.GraduationYear,
		&s.Skills,
		&s.SelfIntroduction,
		&s.ResumeURL,
		&s.ResumeMode,
		&s.ResumeLength,
		&s.ResumeSummaryJSON,
		&s.ResumeSummaryText,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (m *StudentModel) FindByUserId(ctx context.Context, userId int64) (*Student, error) {
	const query = `SELECT id, user_id, student_no, major, class_name, graduation_year, skills, self_introduction, resume_url, resume_mode, resume_length, resume_summary_json, resume_summary_text, created_at, updated_at
FROM students WHERE user_id=$1`

	var s Student
	err := m.pool.QueryRow(ctx, query, userId).Scan(
		&s.ID,
		&s.UserID,
		&s.StudentNo,
		&s.Major,
		&s.ClassName,
		&s.GraduationYear,
		&s.Skills,
		&s.SelfIntroduction,
		&s.ResumeURL,
		&s.ResumeMode,
		&s.ResumeLength,
		&s.ResumeSummaryJSON,
		&s.ResumeSummaryText,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (m *StudentModel) FindByPage(ctx context.Context, page, size int64, keyword string) ([]*Student, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	var whereClause string
	var args []any
	if keyword != "" {
		whereClause = "WHERE student_no ILIKE $1 OR major ILIKE $1 OR class_name ILIKE $1"
		args = append(args, "%"+keyword+"%")
	}

	// total
	countQuery := "SELECT COUNT(1) FROM students " + whereClause
	var total int64
	if err := m.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// list
	offset := (page - 1) * size
	listQuery := `SELECT id, user_id, student_no, major, class_name, graduation_year, skills, self_introduction, resume_url, resume_mode, resume_length, resume_summary_json, resume_summary_text, created_at, updated_at
FROM students ` + whereClause + ` ORDER BY id DESC LIMIT $` + itoa(len(args)+1) + ` OFFSET $` + itoa(len(args)+2)
	args = append(args, size, offset)

	rows, err := m.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*Student
	for rows.Next() {
		var s Student
		if err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.StudentNo,
			&s.Major,
			&s.ClassName,
			&s.GraduationYear,
			&s.Skills,
			&s.SelfIntroduction,
			&s.ResumeURL,
			&s.ResumeMode,
			&s.ResumeLength,
			&s.ResumeSummaryJSON,
			&s.ResumeSummaryText,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, &s)
	}

	return items, total, rows.Err()
}

func (m *StudentModel) Insert(ctx context.Context, s *Student) (int64, error) {
	const query = `INSERT INTO students (user_id, student_no, major, class_name, graduation_year, skills, self_introduction, resume_url, resume_mode, resume_length, resume_summary_json, resume_summary_text)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
RETURNING id`

	var id int64
	err := m.pool.QueryRow(ctx, query,
		s.UserID,
		s.StudentNo,
		s.Major,
		s.ClassName,
		s.GraduationYear,
		s.Skills,
		s.SelfIntroduction,
		s.ResumeURL,
		s.ResumeMode,
		s.ResumeLength,
		s.ResumeSummaryJSON,
		s.ResumeSummaryText,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (m *StudentModel) Update(ctx context.Context, s *Student) error {
	const query = `UPDATE students
SET student_no=$2, major=$3, class_name=$4, graduation_year=$5, skills=$6, self_introduction=$7, resume_url=$8, resume_mode=$9, resume_length=$10, resume_summary_json=$11, resume_summary_text=$12, updated_at=NOW()
WHERE id=$1`

	_, err := m.pool.Exec(ctx, query,
		s.ID,
		s.StudentNo,
		s.Major,
		s.ClassName,
		s.GraduationYear,
		s.Skills,
		s.SelfIntroduction,
		s.ResumeURL,
		s.ResumeMode,
		s.ResumeLength,
		s.ResumeSummaryJSON,
		s.ResumeSummaryText,
	)
	return err
}

func (m *StudentModel) UpdateResumeProcessing(ctx context.Context, id int64, mode string, length int, summaryJSON, summaryText string) error {
	const query = `UPDATE students
SET resume_mode=$2, resume_length=$3, resume_summary_json=$4, resume_summary_text=$5, updated_at=NOW()
WHERE id=$1`
	_, err := m.pool.Exec(ctx, query, id, mode, length, summaryJSON, summaryText)
	return err
}

func (m *StudentModel) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM students WHERE id=$1`
	_, err := m.pool.Exec(ctx, query, id)
	return err
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	var buf [32]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}

	return string(buf[pos:])
}
