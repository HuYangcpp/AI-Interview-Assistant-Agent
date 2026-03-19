package model

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PersonalizedSuggestion struct {
	ID             int64
	StudentID      int64
	SessionID      *int64
	SessionTitle   string
	SessionType    string
	SessionCreated time.Time
	SuggestionType string
	Content        string
	IsRead         bool
	CreatedAt      time.Time
}

type SuggestionModel struct {
	pool *pgxpool.Pool
}

func NewSuggestionModel(pool *pgxpool.Pool) *SuggestionModel {
	return &SuggestionModel{pool: pool}
}

func (m *SuggestionModel) FindByStudentId(ctx context.Context, studentId int64, limit int) ([]*PersonalizedSuggestion, error) {
	if limit <= 0 {
		limit = 100
	}
	const query = `SELECT ps.id, ps.student_id, ps.session_id,
       sess.title AS session_title,
       sess.interview_type AS session_type,
       sess.created_at AS session_created,
       ps.suggestion_type, ps.content, ps.is_read, ps.created_at
FROM personalized_suggestions ps
JOIN interview_sessions sess ON sess.id = ps.session_id
WHERE ps.student_id=$1 AND ps.session_id IS NOT NULL
ORDER BY ps.session_id DESC, CASE ps.suggestion_type
  WHEN 'career' THEN 1
  WHEN 'skill' THEN 2
  WHEN 'interview' THEN 3
  WHEN 'resume' THEN 4
  ELSE 99
END, ps.id DESC
LIMIT $2`

	rows, err := m.pool.Query(ctx, query, studentId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*PersonalizedSuggestion
	for rows.Next() {
		var s PersonalizedSuggestion
		var sessionID int64
		if err := rows.Scan(
			&s.ID,
			&s.StudentID,
			&sessionID,
			&s.SessionTitle,
			&s.SessionType,
			&s.SessionCreated,
			&s.SuggestionType,
			&s.Content,
			&s.IsRead,
			&s.CreatedAt,
		); err != nil {
			return nil, err
		}
		sid := sessionID
		s.SessionID = &sid
		list = append(list, &s)
	}
	return list, rows.Err()
}

func (m *SuggestionModel) Insert(ctx context.Context, s *PersonalizedSuggestion) (int64, error) {
	const query = `INSERT INTO personalized_suggestions (student_id, session_id, suggestion_type, content, is_read)
VALUES ($1,$2,$3,$4,$5) RETURNING id`

	var id int64
	if err := m.pool.QueryRow(ctx, query, s.StudentID, s.SessionID, s.SuggestionType, s.Content, s.IsRead).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (m *SuggestionModel) FindByStudentIdAndSessionId(ctx context.Context, studentId, sessionId int64) ([]*PersonalizedSuggestion, error) {
	const query = `SELECT ps.id, ps.student_id, ps.session_id,
       COALESCE(sess.title, '') AS session_title,
       COALESCE(sess.interview_type, '') AS session_type,
       COALESCE(sess.created_at, ps.created_at) AS session_created,
       ps.suggestion_type, ps.content, ps.is_read, ps.created_at
FROM personalized_suggestions ps
JOIN interview_sessions sess ON sess.id = ps.session_id
WHERE ps.student_id=$1 AND ps.session_id=$2
ORDER BY CASE ps.suggestion_type
  WHEN 'career' THEN 1
  WHEN 'skill' THEN 2
  WHEN 'interview' THEN 3
  WHEN 'resume' THEN 4
  ELSE 99
END, ps.id ASC`

	rows, err := m.pool.Query(ctx, query, studentId, sessionId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*PersonalizedSuggestion
	for rows.Next() {
		var s PersonalizedSuggestion
		var sessionID int64
		if err := rows.Scan(
			&s.ID,
			&s.StudentID,
			&sessionID,
			&s.SessionTitle,
			&s.SessionType,
			&s.SessionCreated,
			&s.SuggestionType,
			&s.Content,
			&s.IsRead,
			&s.CreatedAt,
		); err != nil {
			return nil, err
		}
		sid := sessionID
		s.SessionID = &sid
		list = append(list, &s)
	}
	return list, rows.Err()
}

func (m *SuggestionModel) MarkRead(ctx context.Context, id, studentId int64) error {
	const query = `UPDATE personalized_suggestions SET is_read=TRUE WHERE id=$1 AND student_id=$2`
	ct, err := m.pool.Exec(ctx, query, id, studentId)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *SuggestionModel) DeleteByStudentId(ctx context.Context, studentId int64) error {
	const query = `DELETE FROM personalized_suggestions WHERE student_id=$1`
	_, err := m.pool.Exec(ctx, query, studentId)
	return err
}

func (m *SuggestionModel) DeleteByStudentIdAndSessionId(ctx context.Context, studentId, sessionId int64) error {
	const query = `DELETE FROM personalized_suggestions WHERE student_id=$1 AND session_id=$2`
	_, err := m.pool.Exec(ctx, query, studentId, sessionId)
	return err
}
