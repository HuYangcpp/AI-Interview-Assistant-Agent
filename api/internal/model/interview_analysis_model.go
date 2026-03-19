package model

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InterviewAnalysis struct {
	ID              int64
	SessionID       int64
	StudentID       int64
	OverallScore    float64
	TechnicalScore  float64
	ExpressionScore float64
	LogicScore      float64
	Strengths       string
	Weaknesses      string
	Suggestions     string
	DetailReport    string
	CreatedAt       time.Time
}

type InterviewAnalysisModel struct {
	pool *pgxpool.Pool
}

func NewInterviewAnalysisModel(pool *pgxpool.Pool) *InterviewAnalysisModel {
	return &InterviewAnalysisModel{pool: pool}
}

func (m *InterviewAnalysisModel) Insert(ctx context.Context, a *InterviewAnalysis) (int64, error) {
	const query = `INSERT INTO interview_analyses
(session_id, student_id, overall_score, technical_score, expression_score, logic_score, strengths, weaknesses, suggestions, detail_report)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
RETURNING id`

	var id int64
	err := m.pool.QueryRow(ctx, query,
		a.SessionID,
		a.StudentID,
		a.OverallScore,
		a.TechnicalScore,
		a.ExpressionScore,
		a.LogicScore,
		a.Strengths,
		a.Weaknesses,
		a.Suggestions,
		a.DetailReport,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (m *InterviewAnalysisModel) FindLatestBySessionId(ctx context.Context, sessionId int64) (*InterviewAnalysis, error) {
	const query = `SELECT ia.id, ia.session_id, COALESCE(ia.student_id, s.student_id),
       overall_score::float8, technical_score::float8, expression_score::float8, logic_score::float8,
       ia.strengths, ia.weaknesses, ia.suggestions, ia.detail_report, ia.created_at
FROM interview_analyses ia
JOIN interview_sessions s ON s.id = ia.session_id
WHERE ia.session_id=$1
ORDER BY ia.created_at DESC
LIMIT 1`

	var a InterviewAnalysis
	err := m.pool.QueryRow(ctx, query, sessionId).Scan(
		&a.ID,
		&a.SessionID,
		&a.StudentID,
		&a.OverallScore,
		&a.TechnicalScore,
		&a.ExpressionScore,
		&a.LogicScore,
		&a.Strengths,
		&a.Weaknesses,
		&a.Suggestions,
		&a.DetailReport,
		&a.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &a, nil
}

func (m *InterviewAnalysisModel) DeleteBySessionId(ctx context.Context, sessionId int64) error {
	const query = `DELETE FROM interview_analyses WHERE session_id = $1`
	_, err := m.pool.Exec(ctx, query, sessionId)
	return err
}

// FindRecentByStudentId returns the most recent analyses for the given student,
// resolved via JOIN on interview_sessions.
func (m *InterviewAnalysisModel) FindRecentByStudentId(ctx context.Context, studentId int64, limit int) ([]*InterviewAnalysis, error) {
	if limit <= 0 {
		limit = 5
	}
	const query = `SELECT ia.id, ia.session_id,
       COALESCE(ia.student_id, s.student_id),
       ia.overall_score::float8, ia.technical_score::float8, ia.expression_score::float8, ia.logic_score::float8,
       ia.strengths, ia.weaknesses, ia.suggestions, ia.detail_report, ia.created_at
FROM interview_analyses ia
JOIN interview_sessions s ON s.id = ia.session_id
WHERE s.student_id=$1
ORDER BY ia.created_at DESC
LIMIT $2`

	rows, err := m.pool.Query(ctx, query, studentId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*InterviewAnalysis
	for rows.Next() {
		var a InterviewAnalysis
		if err := rows.Scan(
			&a.ID,
			&a.SessionID,
			&a.StudentID,
			&a.OverallScore,
			&a.TechnicalScore,
			&a.ExpressionScore,
			&a.LogicScore,
			&a.Strengths,
			&a.Weaknesses,
			&a.Suggestions,
			&a.DetailReport,
			&a.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, &a)
	}
	return list, rows.Err()
}
