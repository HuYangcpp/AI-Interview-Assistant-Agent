package logic

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"ai-gozero-agent/api/internal/model"
	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"
	"ai-gozero-agent/api/internal/utils"

	"github.com/jackc/pgx/v5"
	"github.com/zeromicro/go-zero/core/logx"
)

type AdminLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type studentQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func NewAdminLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminLogic {
	return &AdminLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminLogic) ListStudents(page, size int64, keyword string) ([]*types.AdminStudentListItem, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	kw := strings.TrimSpace(keyword)
	kwLike := "%" + kw + "%"

	const countQuery = `
SELECT COUNT(1)
FROM students s
JOIN users u ON u.id = s.user_id
WHERE ($1 = '' OR u.username ILIKE $2 OR u.real_name ILIKE $2 OR s.student_no ILIKE $2 OR s.major ILIKE $2 OR s.class_name ILIKE $2)
`
	var total int64
	if err := l.svcCtx.DB.QueryRow(l.ctx, countQuery, kw, kwLike).Scan(&total); err != nil {
		return nil, 0, err
	}

	const listQuery = `
SELECT s.id, s.user_id, u.username, u.real_name, u.phone, u.email, u.status,
       s.student_no, s.major, s.class_name, s.graduation_year
FROM students s
JOIN users u ON u.id = s.user_id
WHERE ($1 = '' OR u.username ILIKE $2 OR u.real_name ILIKE $2 OR s.student_no ILIKE $2 OR s.major ILIKE $2 OR s.class_name ILIKE $2)
ORDER BY s.id DESC
LIMIT $3 OFFSET $4
`
	offset := (page - 1) * size
	rows, err := l.svcCtx.DB.Query(l.ctx, listQuery, kw, kwLike, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*types.AdminStudentListItem
	for rows.Next() {
		var item types.AdminStudentListItem
		if err := rows.Scan(
			&item.StudentId,
			&item.UserId,
			&item.Username,
			&item.RealName,
			&item.Phone,
			&item.Email,
			&item.Status,
			&item.StudentNo,
			&item.Major,
			&item.ClassName,
			&item.GraduationYear,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, &item)
	}

	return items, total, rows.Err()
}

func (l *AdminLogic) GetStudent(studentId int64) (*types.AdminStudentDetailResp, error) {
	const query = `
SELECT s.id, s.user_id, u.username, u.real_name, u.phone, u.email, u.status,
       s.student_no, s.major, s.class_name, s.graduation_year,
       s.skills, s.self_introduction, s.resume_url
FROM students s
JOIN users u ON u.id = s.user_id
WHERE s.id=$1
`

	var (
		rawSkills string
		resp      types.AdminStudentDetailResp
	)
	err := l.svcCtx.DB.QueryRow(l.ctx, query, studentId).Scan(
		&resp.StudentId,
		&resp.UserId,
		&resp.Username,
		&resp.RealName,
		&resp.Phone,
		&resp.Email,
		&resp.Status,
		&resp.StudentNo,
		&resp.Major,
		&resp.ClassName,
		&resp.GraduationYear,
		&rawSkills,
		&resp.SelfIntroduction,
		&resp.ResumeURL,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	resp.Skills = utils.ParseStringArrayJSON(rawSkills)
	return &resp, nil
}

func (l *AdminLogic) CreateStudent(req *types.AdminCreateStudentReq) (*types.AdminCreateStudentResp, error) {
	req.Username = strings.TrimSpace(req.Username)
	req.StudentNo = strings.TrimSpace(req.StudentNo)
	if req.Username == "" || req.StudentNo == "" {
		return nil, errors.New("username/studentNo required")
	}
	if req.Status == 0 {
		req.Status = 1
	}

	password := strings.TrimSpace(req.Password)
	initialPassword := ""
	if password == "" {
		var err error
		password, err = generateRandomPassword(8)
		if err != nil {
			return nil, err
		}
		initialPassword = password
	}

	tx, err := l.svcCtx.DB.Begin(l.ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(l.ctx)

	studentId, err := l.createStudentRecord(l.ctx, tx, req, password)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(l.ctx); err != nil {
		return nil, err
	}

	return &types.AdminCreateStudentResp{
		ID:              studentId,
		InitialPassword: initialPassword,
	}, nil
}

func generateRandomPassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	max := big.NewInt(int64(len(charset)))
	for i := range result {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("生成随机密码失败: %w", err)
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}

func (l *AdminLogic) createStudentRecord(ctx context.Context, q studentQueryer, req *types.AdminCreateStudentReq, password string) (int64, error) {
	hash, err := utils.HashPassword(password)
	if err != nil {
		return 0, err
	}
	skillsJSON, err := json.Marshal(req.Skills)
	if err != nil {
		return 0, err
	}

	var userId int64
	if err := q.QueryRow(ctx,
		`INSERT INTO users (username, password_hash, role, real_name, phone, email, status)
VALUES ($1,$2,'student',$3,$4,$5,$6) RETURNING id`,
		req.Username,
		hash,
		strings.TrimSpace(req.RealName),
		strings.TrimSpace(req.Phone),
		strings.TrimSpace(req.Email),
		req.Status,
	).Scan(&userId); err != nil {
		return 0, err
	}

	var studentId int64
	if err := q.QueryRow(ctx,
		`INSERT INTO students (user_id, student_no, major, class_name, graduation_year, skills)
VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		userId,
		req.StudentNo,
		strings.TrimSpace(req.Major),
		strings.TrimSpace(req.ClassName),
		req.GraduationYear,
		string(skillsJSON),
	).Scan(&studentId); err != nil {
		return 0, err
	}

	return studentId, nil
}

func (l *AdminLogic) UpdateStudent(studentId int64, req *types.AdminUpdateStudentReq) error {
	if studentId <= 0 {
		return errors.New("invalid id")
	}

	skillsJSON, err := json.Marshal(req.Skills)
	if err != nil {
		return err
	}

	tx, err := l.svcCtx.DB.Begin(l.ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(l.ctx)

	var userId int64
	if err := tx.QueryRow(l.ctx, `SELECT user_id FROM students WHERE id=$1`, studentId).Scan(&userId); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrNotFound
		}
		return err
	}

	if _, err := tx.Exec(l.ctx,
		`UPDATE users SET real_name=$2, phone=$3, email=$4, status=$5, updated_at=NOW() WHERE id=$1`,
		userId,
		strings.TrimSpace(req.RealName),
		strings.TrimSpace(req.Phone),
		strings.TrimSpace(req.Email),
		req.Status,
	); err != nil {
		return err
	}

	if _, err := tx.Exec(l.ctx,
		`UPDATE students
SET student_no=$2, major=$3, class_name=$4, graduation_year=$5, skills=$6, self_introduction=$7, updated_at=NOW()
WHERE id=$1`,
		studentId,
		strings.TrimSpace(req.StudentNo),
		strings.TrimSpace(req.Major),
		strings.TrimSpace(req.ClassName),
		req.GraduationYear,
		string(skillsJSON),
		strings.TrimSpace(req.SelfIntroduction),
	); err != nil {
		return err
	}

	return tx.Commit(l.ctx)
}

func (l *AdminLogic) ListProfileChangeRequests(limit int64) ([]*types.AdminProfileChangeRequestItem, error) {
	if limit <= 0 {
		limit = 100
	}

	const query = `
SELECT r.id, s.id, u.username, u.real_name,
       s.student_no, s.major, s.class_name, s.graduation_year,
       r.requested_student_no, r.requested_major, r.requested_class_name, r.requested_graduation_year,
       r.reason, r.status, r.review_comment,
       COALESCE(NULLIF(reviewer.real_name, ''), reviewer.username, ''),
       r.created_at, r.reviewed_at
FROM student_profile_change_requests r
JOIN students s ON s.id = r.student_id
JOIN users u ON u.id = s.user_id
LEFT JOIN users reviewer ON reviewer.id = r.reviewed_by
ORDER BY CASE r.status WHEN 'pending' THEN 0 WHEN 'approved' THEN 1 ELSE 2 END, r.created_at DESC
LIMIT $1
`

	rows, err := l.svcCtx.DB.Query(l.ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*types.AdminProfileChangeRequestItem
	for rows.Next() {
		item := new(types.AdminProfileChangeRequestItem)
		if err := rows.Scan(
			&item.ID,
			&item.StudentId,
			&item.Username,
			&item.RealName,
			&item.CurrentStudentNo,
			&item.CurrentMajor,
			&item.CurrentClassName,
			&item.CurrentGraduationYear,
			&item.RequestedStudentNo,
			&item.RequestedMajor,
			&item.RequestedClassName,
			&item.RequestedGraduationYear,
			&item.Reason,
			&item.Status,
			&item.ReviewComment,
			&item.ReviewedByName,
			&item.CreatedAt,
			&item.ReviewedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (l *AdminLogic) ReviewProfileChangeRequest(adminUserId, requestId int64, req *types.AdminReviewProfileChangeRequestReq) error {
	if requestId <= 0 {
		return errors.New("invalid id")
	}

	status := strings.TrimSpace(req.Status)
	if status != "approved" && status != "rejected" {
		return errors.New("invalid status")
	}

	tx, err := l.svcCtx.DB.Begin(l.ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(l.ctx)

	var studentId int64
	var requestedStudentNo string
	var requestedMajor string
	var requestedClassName string
	var requestedGraduationYear int
	var currentStatus string

	const lockQuery = `
SELECT student_id, requested_student_no, requested_major, requested_class_name,
       requested_graduation_year, status
FROM student_profile_change_requests
WHERE id=$1
FOR UPDATE
`
	if err := tx.QueryRow(l.ctx, lockQuery, requestId).Scan(
		&studentId,
		&requestedStudentNo,
		&requestedMajor,
		&requestedClassName,
		&requestedGraduationYear,
		&currentStatus,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrNotFound
		}
		return err
	}

	if currentStatus != "pending" {
		return errors.New("request already reviewed")
	}

	if status == "approved" {
		if _, err := tx.Exec(l.ctx,
			`UPDATE students
SET student_no=$2, major=$3, class_name=$4, graduation_year=$5, updated_at=NOW()
WHERE id=$1`,
			studentId,
			strings.TrimSpace(requestedStudentNo),
			strings.TrimSpace(requestedMajor),
			strings.TrimSpace(requestedClassName),
			requestedGraduationYear,
		); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(l.ctx,
		`UPDATE student_profile_change_requests
SET status=$2, review_comment=$3, reviewed_by=$4, reviewed_at=NOW(), updated_at=NOW()
WHERE id=$1`,
		requestId,
		status,
		strings.TrimSpace(req.ReviewComment),
		adminUserId,
	); err != nil {
		return err
	}

	return tx.Commit(l.ctx)
}

func (l *AdminLogic) DeleteStudent(studentId int64) error {
	if studentId <= 0 {
		return errors.New("invalid id")
	}

	tx, err := l.svcCtx.DB.BeginTx(l.ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	rollback := func() {
		_ = tx.Rollback(l.ctx)
	}

	var userId int64
	if err := tx.QueryRow(l.ctx, `SELECT user_id FROM students WHERE id=$1`, studentId).Scan(&userId); err != nil {
		rollback()
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrNotFound
		}
		return err
	}

	// 先清理关联（Phase 2 仅处理就业信息）
	if _, err := tx.Exec(l.ctx, `DELETE FROM interview_analyses WHERE session_id IN (SELECT id FROM interview_sessions WHERE student_id=$1)`, studentId); err != nil {
		rollback()
		return err
	}
	if _, err := tx.Exec(l.ctx, `DELETE FROM personalized_suggestions WHERE student_id=$1`, studentId); err != nil {
		rollback()
		return err
	}
	if _, err := tx.Exec(l.ctx, `DELETE FROM interview_sessions WHERE student_id=$1`, studentId); err != nil {
		rollback()
		return err
	}
	if _, err := tx.Exec(l.ctx, `DELETE FROM employments WHERE student_id=$1`, studentId); err != nil {
		rollback()
		return err
	}

	if _, err := tx.Exec(l.ctx, `DELETE FROM students WHERE id=$1`, studentId); err != nil {
		rollback()
		return err
	}
	if _, err := tx.Exec(l.ctx, `DELETE FROM users WHERE id=$1`, userId); err != nil {
		rollback()
		return err
	}

	if err := tx.Commit(l.ctx); err != nil {
		rollback()
		return err
	}

	return nil
}

func (l *AdminLogic) ImportStudents(file multipart.File) (resp *types.AdminImportStudentsResp, err error) {
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	tx, err := l.svcCtx.DB.Begin(l.ctx)
	if err != nil {
		return nil, fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(l.ctx)
		}
	}()

	successCount := 0
	for i, row := range rows {
		if i == 0 && len(row) > 0 && strings.EqualFold(strings.TrimSpace(row[0]), "username") {
			continue
		}
		if len(row) < 3 {
			return nil, fmt.Errorf("第 %d 行导入失败: 数据列不足", i+2)
		}

		username := strings.TrimSpace(row[0])
		password := strings.TrimSpace(row[1])
		studentNo := strings.TrimSpace(row[2])
		realName := ""
		if len(row) >= 4 {
			realName = strings.TrimSpace(row[3])
		}
		graduationYear := 0
		if len(row) >= 5 && strings.TrimSpace(row[4]) != "" {
			y, convErr := strconv.Atoi(strings.TrimSpace(row[4]))
			if convErr != nil {
				return nil, fmt.Errorf("第 %d 行导入失败: 毕业年份无效", i+2)
			}
			graduationYear = y
		}

		if username == "" || studentNo == "" {
			return nil, fmt.Errorf("第 %d 行导入失败: username/studentNo required", i+2)
		}

		if password == "" {
			password, err = generateRandomPassword(8)
			if err != nil {
				return nil, fmt.Errorf("第 %d 行导入失败: %w", i+2, err)
			}
		}

		if _, err = l.createStudentRecord(l.ctx, tx, &types.AdminCreateStudentReq{
			Username:       username,
			Password:       password,
			StudentNo:      studentNo,
			RealName:       realName,
			Status:         1,
			GraduationYear: graduationYear,
		}, password); err != nil {
			return nil, fmt.Errorf("第 %d 行导入失败: %w", i+2, err)
		}
		successCount++
	}

	if err = tx.Commit(l.ctx); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	return &types.AdminImportStudentsResp{Count: successCount}, nil
}

func (l *AdminLogic) GetDashboardStats() (*types.AdminDashboardStatsResp, error) {
	s, err := l.svcCtx.StatsModel.GetDashboardStats(l.ctx)
	if err != nil {
		return nil, err
	}
	return &types.AdminDashboardStatsResp{
		TotalStudents:    s.TotalStudents,
		TotalEmployments: s.TotalEmployments,
		EmployedStudents: s.EmployedStudents,
		EmploymentRate:   s.EmploymentRate,
	}, nil
}

func (l *AdminLogic) GetEmploymentRate() (*types.AdminEmploymentRateResp, error) {
	r, err := l.svcCtx.StatsModel.GetEmploymentRate(l.ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*types.AdminEmploymentRateItem, 0, len(r.Items))
	for _, item := range r.Items {
		items = append(items, &types.AdminEmploymentRateItem{Status: item.Status, Count: item.Count})
	}
	return &types.AdminEmploymentRateResp{
		TotalStudents:  r.TotalStudents,
		EmploymentRate: r.EmploymentRate,
		Items:          items,
	}, nil
}

func (l *AdminLogic) GetMajorStats() ([]*types.AdminMajorStatItem, error) {
	rows, err := l.svcCtx.StatsModel.GetMajorStats(l.ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*types.AdminMajorStatItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &types.AdminMajorStatItem{
			Major:            row.Major,
			TotalStudents:    row.TotalStudents,
			EmployedStudents: row.EmployedStudents,
		})
	}
	return items, nil
}

func (l *AdminLogic) GetEmploymentTrend(months int) ([]*types.AdminTrendPoint, error) {
	rows, err := l.svcCtx.StatsModel.GetEmploymentTrend(l.ctx, months)
	if err != nil {
		return nil, err
	}
	items := make([]*types.AdminTrendPoint, 0, len(rows))
	for _, row := range rows {
		items = append(items, &types.AdminTrendPoint{
			Month:            row.Month,
			EmployedStudents: row.EmployedStudents,
		})
	}
	return items, nil
}

func (l *AdminLogic) ListKnowledge(limit int) ([]*types.KnowledgeItem, error) {
	if limit <= 0 {
		limit = 200
	}
	const query = `SELECT id, title, created_at FROM knowledge_base ORDER BY id DESC LIMIT $1`
	rows, err := l.svcCtx.DB.Query(l.ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*types.KnowledgeItem
	for rows.Next() {
		var (
			id        int64
			title     string
			createdAt time.Time
		)
		if err := rows.Scan(&id, &title, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, &types.KnowledgeItem{
			ID:        id,
			Title:     title,
			CreatedAt: createdAt.Format(time.RFC3339),
		})
	}
	return items, rows.Err()
}

func (l *AdminLogic) DeleteKnowledge(id int64) error {
	if id <= 0 {
		return errors.New("invalid id")
	}
	const query = `DELETE FROM knowledge_base WHERE id=$1`
	ct, err := l.svcCtx.DB.Exec(l.ctx, query, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (l *AdminLogic) ListEmployments(limit int) ([]*types.AdminEmploymentItem, error) {
	if limit <= 0 {
		limit = 200
	}
	const query = `
SELECT e.id, e.student_id, s.student_no, u.real_name, s.major,
       e.status, e.verification_status, e.review_comment, e.company_name, e.position, e.city, e.salary_range,
       e.offer_date, e.entry_date, e.reviewed_at, e.updated_at, e.notes
FROM employments e
JOIN students s ON s.id = e.student_id
JOIN users u ON u.id = s.user_id
ORDER BY e.id DESC
LIMIT $1`

	rows, err := l.svcCtx.DB.Query(l.ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*types.AdminEmploymentItem
	for rows.Next() {
		var (
			item       types.AdminEmploymentItem
			offerDate  *time.Time
			entryDate  *time.Time
			reviewedAt *time.Time
			updatedAt  time.Time
		)
		if err := rows.Scan(
			&item.ID,
			&item.StudentID,
			&item.StudentNo,
			&item.RealName,
			&item.Major,
			&item.Status,
			&item.VerificationStatus,
			&item.ReviewComment,
			&item.CompanyName,
			&item.Position,
			&item.City,
			&item.SalaryRange,
			&offerDate,
			&entryDate,
			&reviewedAt,
			&updatedAt,
			&item.Notes,
		); err != nil {
			return nil, err
		}
		if offerDate != nil {
			item.OfferDate = offerDate.Format("2006-01-02")
		}
		if entryDate != nil {
			item.EntryDate = entryDate.Format("2006-01-02")
		}
		if reviewedAt != nil {
			item.ReviewedAt = reviewedAt.Format(time.RFC3339)
		}
		item.UpdatedAt = updatedAt.Format(time.RFC3339)

		evidences, err := l.svcCtx.EmploymentEvidenceModel.FindByEmploymentID(l.ctx, item.ID)
		if err != nil {
			return nil, err
		}
		for _, evidence := range evidences {
			item.Evidences = append(item.Evidences, &types.EmploymentEvidenceResp{
				ID:        evidence.ID,
				FileURL:   evidence.FileURL,
				FileName:  evidence.FileName,
				MimeType:  evidence.MimeType,
				CreatedAt: evidence.CreatedAt.Format(time.RFC3339),
			})
		}
		items = append(items, &item)
	}
	return items, rows.Err()
}

func (l *AdminLogic) ReviewEmployment(id int64, reviewerID int64, verificationStatus, reviewComment string) error {
	verificationStatus = strings.TrimSpace(verificationStatus)
	reviewComment = strings.TrimSpace(reviewComment)
	if id <= 0 {
		return errors.New("invalid id")
	}
	switch verificationStatus {
	case "pending", "approved", "rejected":
	default:
		return errors.New("invalid verificationStatus")
	}
	if verificationStatus == "rejected" && reviewComment == "" {
		return errors.New("驳回时请填写审核意见")
	}

	employment, err := l.svcCtx.EmploymentModel.FindById(l.ctx, id)
	if err != nil {
		return err
	}
	if verificationStatus == "approved" && employment.Status != "seeking" {
		evidences, err := l.svcCtx.EmploymentEvidenceModel.FindByEmploymentID(l.ctx, id)
		if err != nil {
			return err
		}
		if len(evidences) == 0 {
			return errors.New("请先上传就业佐证材料，再执行通过审核")
		}
	}

	if verificationStatus == "pending" {
		return l.svcCtx.EmploymentModel.UpdateReview(l.ctx, id, verificationStatus, reviewComment, nil, nil)
	}

	now := time.Now()
	return l.svcCtx.EmploymentModel.UpdateReview(l.ctx, id, verificationStatus, reviewComment, &reviewerID, &now)
}

func (l *AdminLogic) UpdateEmploymentStatus(id int64, status string) error {
	status = strings.TrimSpace(status)
	if id <= 0 || status == "" {
		return errors.New("invalid id/status")
	}
	const query = `UPDATE employments SET status=$2, updated_at=NOW() WHERE id=$1`
	ct, err := l.svcCtx.DB.Exec(l.ctx, query, id, status)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}
