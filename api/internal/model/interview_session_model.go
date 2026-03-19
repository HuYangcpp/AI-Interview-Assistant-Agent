package model

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InterviewSession struct {
	ID              int64
	ChatID          string
	StudentID       int64
	Title           string
	InterviewType   string
	CustomPrompt    string
	Status          string
	TotalQuestions  int
	DurationSeconds int
	Score           *float64
	ScoreDetail     []byte
	AISummary       string
	CreatedAt       time.Time
	CompletedAt     *time.Time
}

type InterviewSessionModel struct {
	pool *pgxpool.Pool
}

func NewInterviewSessionModel(pool *pgxpool.Pool) *InterviewSessionModel {
	return &InterviewSessionModel{pool: pool}
}

func (m *InterviewSessionModel) Insert(ctx context.Context, s *InterviewSession) (int64, error) {
	const query = `INSERT INTO interview_sessions (chat_id, student_id, title, interview_type, custom_prompt)
	VALUES ($1,$2,$3,$4,$5)
	RETURNING id`

	var id int64
	err := m.pool.QueryRow(ctx, query, s.ChatID, s.StudentID, s.Title, s.InterviewType, s.CustomPrompt).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (m *InterviewSessionModel) FindById(ctx context.Context, id int64) (*InterviewSession, error) {
	const query = `SELECT id, chat_id, student_id, title, interview_type, COALESCE(custom_prompt,''), status,
	       total_questions, duration_seconds, score::float8, score_detail, ai_summary, created_at, completed_at
	FROM interview_sessions WHERE id=$1`

	var s InterviewSession
	err := m.pool.QueryRow(ctx, query, id).Scan(
		&s.ID,
		&s.ChatID,
		&s.StudentID,
		&s.Title,
		&s.InterviewType,
		&s.CustomPrompt,
		&s.Status,
		&s.TotalQuestions,
		&s.DurationSeconds,
		&s.Score,
		&s.ScoreDetail,
		&s.AISummary,
		&s.CreatedAt,
		&s.CompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (m *InterviewSessionModel) FindByStudentId(ctx context.Context, studentId int64, limit int) ([]*InterviewSession, error) {
	if limit <= 0 {
		limit = 50
	}
	const query = `SELECT id, chat_id, student_id, title, interview_type, COALESCE(custom_prompt,''), status,
	       total_questions, duration_seconds, score::float8, score_detail, ai_summary, created_at, completed_at
	FROM interview_sessions WHERE student_id=$1
	ORDER BY id DESC
	LIMIT $2`

	rows, err := m.pool.Query(ctx, query, studentId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*InterviewSession
	for rows.Next() {
		var s InterviewSession
		if err := rows.Scan(
			&s.ID,
			&s.ChatID,
			&s.StudentID,
			&s.Title,
			&s.InterviewType,
			&s.CustomPrompt,
			&s.Status,
			&s.TotalQuestions,
			&s.DurationSeconds,
			&s.Score,
			&s.ScoreDetail,
			&s.AISummary,
			&s.CreatedAt,
			&s.CompletedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, &s)
	}

	return list, rows.Err()
}

func (m *InterviewSessionModel) IncrementQuestions(ctx context.Context, id int64) error {
	const query = `UPDATE interview_sessions SET total_questions=total_questions+1 WHERE id=$1`
	_, err := m.pool.Exec(ctx, query, id)
	return err
}

func (m *InterviewSessionModel) MarkCompleted(ctx context.Context, id int64, durationSeconds int) error {
	const query = `UPDATE interview_sessions
	SET status='completed', duration_seconds=$2, completed_at=NOW()
	WHERE id=$1 AND status='ongoing'`
	ct, err := m.pool.Exec(ctx, query, id, durationSeconds)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *InterviewSessionModel) DeleteByIdAndStudentId(ctx context.Context, id, studentId int64) error {
	const query = `DELETE FROM interview_sessions WHERE id=$1 AND student_id=$2`
	ct, err := m.pool.Exec(ctx, query, id, studentId)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *InterviewSessionModel) UpdateScore(ctx context.Context, id int64, overallScore float64, scoreDetail []byte, aiSummary string) error {
	const query = `UPDATE interview_sessions
	SET score=$2, score_detail=$3, ai_summary=$4
	WHERE id=$1`
	_, err := m.pool.Exec(ctx, query, id, overallScore, scoreDetail, aiSummary)
	return err
}
