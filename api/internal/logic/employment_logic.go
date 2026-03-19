package logic

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ai-gozero-agent/api/internal/model"
	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type EmploymentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

var allowedEmploymentEvidenceExts = map[string]bool{
	".pdf":  true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
}

const maxEmploymentEvidenceSize = 10 << 20

func NewEmploymentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EmploymentLogic {
	return &EmploymentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EmploymentLogic) List(userId int64) ([]*types.EmploymentResp, error) {
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return nil, err
	}
	list, err := l.svcCtx.EmploymentModel.FindByStudentId(l.ctx, s.ID)
	if err != nil {
		return nil, err
	}

	var resp []*types.EmploymentResp
	for _, e := range list {
		evidences, err := l.svcCtx.EmploymentEvidenceModel.FindByEmploymentID(l.ctx, e.ID)
		if err != nil {
			return nil, err
		}
		resp = append(resp, toEmploymentResp(e, evidences))
	}
	return resp, nil
}

func (l *EmploymentLogic) Create(userId int64, req *types.CreateEmploymentReq) (int64, error) {
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return 0, err
	}
	if req.Status == "" {
		req.Status = "seeking"
	}
	offerDate, err := parseDate(req.OfferDate)
	if err != nil {
		return 0, err
	}
	entryDate, err := parseDate(req.EntryDate)
	if err != nil {
		return 0, err
	}

	return l.svcCtx.EmploymentModel.Insert(l.ctx, &model.Employment{
		StudentID:          s.ID,
		Status:             req.Status,
		VerificationStatus: "pending",
		CompanyName:        req.CompanyName,
		Position:           req.Position,
		SalaryRange:        req.SalaryRange,
		City:               req.City,
		OfferDate:          offerDate,
		EntryDate:          entryDate,
		Notes:              req.Notes,
	})
}

func (l *EmploymentLogic) Update(userId, id int64, req *types.UpdateEmploymentReq) error {
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return err
	}
	if id <= 0 {
		return errors.New("invalid id")
	}
	if req.Status == "" {
		req.Status = "seeking"
	}
	offerDate, err := parseDate(req.OfferDate)
	if err != nil {
		return err
	}
	entryDate, err := parseDate(req.EntryDate)
	if err != nil {
		return err
	}

	return l.svcCtx.EmploymentModel.UpdateByStudent(l.ctx, &model.Employment{
		ID:          id,
		StudentID:   s.ID,
		Status:      req.Status,
		CompanyName: req.CompanyName,
		Position:    req.Position,
		SalaryRange: req.SalaryRange,
		City:        req.City,
		OfferDate:   offerDate,
		EntryDate:   entryDate,
		Notes:       req.Notes,
	})
}

func (l *EmploymentLogic) Delete(userId, id int64) error {
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return err
	}
	if id <= 0 {
		return errors.New("invalid id")
	}
	return l.svcCtx.EmploymentModel.DeleteByStudent(l.ctx, id, s.ID)
}

func (l *EmploymentLogic) UploadEvidence(userId, employmentID int64, file multipart.File, header *multipart.FileHeader) (*types.EmploymentEvidenceResp, error) {
	if header.Size > maxEmploymentEvidenceSize {
		return nil, errors.New("文件大小超过限制（最大 10MB）")
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedEmploymentEvidenceExts[ext] {
		return nil, errors.New("当前仅支持 PDF/PNG/JPG/WEBP 格式")
	}

	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return nil, err
	}

	employment, err := l.svcCtx.EmploymentModel.FindById(l.ctx, employmentID)
	if err != nil {
		return nil, err
	}
	if employment.StudentID != s.ID {
		return nil, errors.New("forbidden")
	}

	uploadDir := strings.TrimSpace(l.svcCtx.Config.UploadDir)
	if uploadDir == "" {
		return nil, errors.New("uploadDir not configured")
	}

	evidenceDir := filepath.Join(uploadDir, "employment-evidences")
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		return nil, err
	}

	safeName := filepath.Base(header.Filename)
	safeName = strings.ReplaceAll(safeName, " ", "_")
	if safeName == "." || safeName == string(filepath.Separator) {
		safeName = "evidence" + ext
	}

	outName := strings.Join([]string{
		formatInt64(employmentID),
		formatInt64(time.Now().UnixNano()),
		safeName,
	}, "_")
	outPath := filepath.Join(evidenceDir, outName)

	out, err := os.Create(outPath)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		return nil, err
	}

	fileURL := "/uploads/employment-evidences/" + outName
	mimeType := strings.TrimSpace(header.Header.Get("Content-Type"))
	evidence := &model.EmploymentEvidence{
		EmploymentID: employmentID,
		FileURL:      fileURL,
		FileName:     safeName,
		MimeType:     mimeType,
		UploadedBy:   userId,
	}
	id, err := l.svcCtx.EmploymentEvidenceModel.Insert(l.ctx, evidence)
	if err != nil {
		return nil, err
	}
	evidence.ID = id
	evidence.CreatedAt = time.Now()

	if err := l.svcCtx.EmploymentModel.ResetVerification(l.ctx, employmentID); err != nil {
		return nil, err
	}

	return toEmploymentEvidenceResp(evidence), nil
}

func (l *EmploymentLogic) DeleteEvidence(userId, employmentID, evidenceID int64) error {
	s, err := l.svcCtx.StudentModel.FindByUserId(l.ctx, userId)
	if err != nil {
		return err
	}

	employment, err := l.svcCtx.EmploymentModel.FindById(l.ctx, employmentID)
	if err != nil {
		return err
	}
	if employment.StudentID != s.ID {
		return errors.New("forbidden")
	}

	evidence, err := l.svcCtx.EmploymentEvidenceModel.FindByID(l.ctx, evidenceID)
	if err != nil {
		return err
	}
	if evidence.EmploymentID != employmentID {
		return model.ErrNotFound
	}

	if err := l.svcCtx.EmploymentEvidenceModel.DeleteByEmploymentID(l.ctx, evidenceID, employmentID); err != nil {
		return err
	}
	if err := l.svcCtx.EmploymentModel.ResetVerification(l.ctx, employmentID); err != nil {
		return err
	}

	uploadDir := strings.TrimSpace(l.svcCtx.Config.UploadDir)
	if uploadDir != "" && evidence.FileURL != "" {
		relativePath := strings.TrimPrefix(evidence.FileURL, "/uploads/")
		filePath := filepath.Join(uploadDir, relativePath)
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			l.Logger.Errorf("删除就业佐证文件失败: %v", err)
		}
	}

	return nil
}

func toEmploymentResp(e *model.Employment, evidences []*model.EmploymentEvidence) *types.EmploymentResp {
	var reviewedAt string
	if e.ReviewedAt != nil {
		reviewedAt = e.ReviewedAt.Format(time.RFC3339)
	}

	resp := &types.EmploymentResp{
		ID:                 e.ID,
		Status:             e.Status,
		VerificationStatus: e.VerificationStatus,
		ReviewComment:      e.ReviewComment,
		CompanyName:        e.CompanyName,
		Position:           e.Position,
		SalaryRange:        e.SalaryRange,
		City:               e.City,
		Notes:              e.Notes,
		ReviewedAt:         reviewedAt,
		CreatedAt:          e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          e.UpdatedAt.Format(time.RFC3339),
	}

	if e.OfferDate != nil {
		resp.OfferDate = e.OfferDate.Format("2006-01-02")
	}
	if e.EntryDate != nil {
		resp.EntryDate = e.EntryDate.Format("2006-01-02")
	}
	for _, evidence := range evidences {
		resp.Evidences = append(resp.Evidences, toEmploymentEvidenceResp(evidence))
	}

	return resp
}

func toEmploymentEvidenceResp(evidence *model.EmploymentEvidence) *types.EmploymentEvidenceResp {
	return &types.EmploymentEvidenceResp{
		ID:        evidence.ID,
		FileURL:   evidence.FileURL,
		FileName:  evidence.FileName,
		MimeType:  evidence.MimeType,
		CreatedAt: evidence.CreatedAt.Format(time.RFC3339),
	}
}

func formatInt64(i int64) string {
	return strconv.FormatInt(i, 10)
}

func parseDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, errors.New("invalid date: " + s)
	}
	return &t, nil
}
