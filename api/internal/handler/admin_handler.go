package handler

import (
	"net/http"

	"ai-gozero-agent/api/internal/logic"
	"ai-gozero-agent/api/internal/model"
	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

type studentIdPathReq struct {
	ID int64 `path:"id"`
}

func AdminStudentsListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AdminStudentListReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewAdminLogic(r.Context(), svcCtx)
		items, total, err := l.ListStudents(req.Page, req.Size, req.Keyword)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, &types.AdminStudentListResp{
			Items: items,
			Total: total,
			Page:  req.Page,
			Size:  req.Size,
		})
	}
}

func AdminStudentsDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var path studentIdPathReq
		if err := httpx.ParsePath(r, &path); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewAdminLogic(r.Context(), svcCtx)
		resp, err := l.GetStudent(path.ID)
		if err != nil {
			status := http.StatusInternalServerError
			if err == model.ErrNotFound {
				status = http.StatusNotFound
			}
			httpx.WriteJson(w, status, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, resp)
	}
}

func AdminStudentsCreateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AdminCreateStudentReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewAdminLogic(r.Context(), svcCtx)
		resp, err := l.CreateStudent(&req)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, resp)
	}
}

func AdminStudentsUpdateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var path studentIdPathReq
		if err := httpx.ParsePath(r, &path); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		var req types.AdminUpdateStudentReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewAdminLogic(r.Context(), svcCtx)
		if err := l.UpdateStudent(path.ID, &req); err != nil {
			status := http.StatusBadRequest
			if err == model.ErrNotFound {
				status = http.StatusNotFound
			}
			httpx.WriteJson(w, status, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, map[string]any{"msg": "ok"})
	}
}

func AdminStudentsDeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var path studentIdPathReq
		if err := httpx.Parse(r, &path); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewAdminLogic(r.Context(), svcCtx)
		if err := l.DeleteStudent(path.ID); err != nil {
			status := http.StatusBadRequest
			if err == model.ErrNotFound {
				status = http.StatusNotFound
			}
			httpx.WriteJson(w, status, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, map[string]any{"msg": "ok"})
	}
}

func AdminStudentsImportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, _, err := r.FormFile("file")
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": "file required"})
			return
		}
		defer file.Close()

		l := logic.NewAdminLogic(r.Context(), svcCtx)
		resp, err := l.ImportStudents(file)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, resp)
	}
}

func AdminProfileChangeRequestsListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewAdminLogic(r.Context(), svcCtx)
		resp, err := l.ListProfileChangeRequests(100)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, resp)
	}
}

func AdminProfileChangeRequestReviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		adminUserId, _ := r.Context().Value("userId").(int64)
		if adminUserId <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
			return
		}

		var path profileChangeRequestIdPathReq
		if err := httpx.ParsePath(r, &path); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		var req types.AdminReviewProfileChangeRequestReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewAdminLogic(r.Context(), svcCtx)
		if err := l.ReviewProfileChangeRequest(adminUserId, path.ID, &req); err != nil {
			status := http.StatusBadRequest
			if err == model.ErrNotFound {
				status = http.StatusNotFound
			}
			httpx.WriteJson(w, status, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, map[string]any{"msg": "ok"})
	}
}

type knowledgeIdPathReq struct {
	ID int64 `path:"id"`
}

type adminEmploymentIdPathReq struct {
	ID int64 `path:"id"`
}

type profileChangeRequestIdPathReq struct {
	ID int64 `path:"id"`
}

func AdminDashboardStatsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewAdminLogic(r.Context(), svcCtx)
		resp, err := l.GetDashboardStats()
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		httpx.OkJson(w, resp)
	}
}

func AdminDashboardEmploymentRateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewAdminLogic(r.Context(), svcCtx)
		resp, err := l.GetEmploymentRate()
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		httpx.OkJson(w, resp)
	}
}

func AdminDashboardMajorStatsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewAdminLogic(r.Context(), svcCtx)
		resp, err := l.GetMajorStats()
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		httpx.OkJson(w, resp)
	}
}

func AdminDashboardTrendHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewAdminLogic(r.Context(), svcCtx)
		resp, err := l.GetEmploymentTrend(12)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		httpx.OkJson(w, resp)
	}
}

func AdminKnowledgeListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewAdminLogic(r.Context(), svcCtx)
		resp, err := l.ListKnowledge(200)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		httpx.OkJson(w, resp)
	}
}

func AdminKnowledgeDeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var path knowledgeIdPathReq
		if err := httpx.ParsePath(r, &path); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewAdminLogic(r.Context(), svcCtx)
		if err := l.DeleteKnowledge(path.ID); err != nil {
			status := http.StatusBadRequest
			if err == model.ErrNotFound {
				status = http.StatusNotFound
			}
			httpx.WriteJson(w, status, map[string]any{"error": err.Error()})
			return
		}
		httpx.OkJson(w, map[string]any{"msg": "ok"})
	}
}

func AdminEmploymentsListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewAdminLogic(r.Context(), svcCtx)
		resp, err := l.ListEmployments(200)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		httpx.OkJson(w, resp)
	}
}

func AdminEmploymentStatusUpdateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var path adminEmploymentIdPathReq
		if err := httpx.ParsePath(r, &path); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		var req types.AdminUpdateEmploymentStatusReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewAdminLogic(r.Context(), svcCtx)
		if err := l.UpdateEmploymentStatus(path.ID, req.Status); err != nil {
			status := http.StatusBadRequest
			if err == model.ErrNotFound {
				status = http.StatusNotFound
			}
			httpx.WriteJson(w, status, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, map[string]any{"msg": "ok"})
	}
}

func AdminEmploymentReviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		var path adminEmploymentIdPathReq
		if err := httpx.ParsePath(r, &path); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		var req types.AdminReviewEmploymentReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewAdminLogic(r.Context(), svcCtx)
		if err := l.ReviewEmployment(path.ID, userId, req.VerificationStatus, req.ReviewComment); err != nil {
			status := http.StatusBadRequest
			if err == model.ErrNotFound {
				status = http.StatusNotFound
			}
			httpx.WriteJson(w, status, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, map[string]any{"msg": "ok"})
	}
}
