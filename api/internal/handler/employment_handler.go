package handler

import (
	"net/http"

	"ai-gozero-agent/api/internal/logic"
	"ai-gozero-agent/api/internal/model"
	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

type employmentIdPathReq struct {
	ID int64 `path:"id"`
}

type employmentEvidencePathReq struct {
	ID         int64 `path:"id"`
	EvidenceID int64 `path:"evidenceId"`
}

func EmploymentListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		l := logic.NewEmploymentLogic(r.Context(), svcCtx)
		resp, err := l.List(userId)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, resp)
	}
}

func EmploymentCreateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		var req types.CreateEmploymentReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewEmploymentLogic(r.Context(), svcCtx)
		id, err := l.Create(userId, &req)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, map[string]any{"id": id})
	}
}

func EmploymentUpdateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		var path employmentIdPathReq
		if err := httpx.ParsePath(r, &path); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		var req types.UpdateEmploymentReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewEmploymentLogic(r.Context(), svcCtx)
		if err := l.Update(userId, path.ID, &req); err != nil {
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

func EmploymentDeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		var path employmentIdPathReq
		if err := httpx.ParsePath(r, &path); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewEmploymentLogic(r.Context(), svcCtx)
		if err := l.Delete(userId, path.ID); err != nil {
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

func EmploymentEvidenceUploadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		var path employmentIdPathReq
		if err := httpx.ParsePath(r, &path); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": "file required"})
			return
		}
		defer file.Close()

		l := logic.NewEmploymentLogic(r.Context(), svcCtx)
		resp, err := l.UploadEvidence(userId, path.ID, file, header)
		if err != nil {
			status := http.StatusBadRequest
			if err == model.ErrNotFound {
				status = http.StatusNotFound
			}
			if err.Error() == "forbidden" {
				status = http.StatusForbidden
			}
			httpx.WriteJson(w, status, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, resp)
	}
}

func EmploymentEvidenceDeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		var path employmentEvidencePathReq
		if err := httpx.ParsePath(r, &path); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewEmploymentLogic(r.Context(), svcCtx)
		if err := l.DeleteEvidence(userId, path.ID, path.EvidenceID); err != nil {
			status := http.StatusBadRequest
			if err == model.ErrNotFound {
				status = http.StatusNotFound
			}
			if err.Error() == "forbidden" {
				status = http.StatusForbidden
			}
			httpx.WriteJson(w, status, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, map[string]any{"msg": "ok"})
	}
}
