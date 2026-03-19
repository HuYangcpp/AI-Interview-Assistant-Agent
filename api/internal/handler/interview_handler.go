package handler

import (
	"net/http"

	"ai-gozero-agent/api/internal/logic"
	"ai-gozero-agent/api/internal/model"
	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

type sessionIdPathReq struct {
	ID int64 `path:"id"`
}

func InterviewSessionCreateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		var req types.CreateInterviewSessionReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewInterviewLogic(r.Context(), svcCtx)
		resp, err := l.CreateSession(userId, &req)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, resp)
	}
}

func InterviewSessionListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		l := logic.NewInterviewLogic(r.Context(), svcCtx)
		resp, err := l.ListSessions(userId)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, resp)
	}
}

func InterviewSessionDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		var path sessionIdPathReq
		if err := httpx.Parse(r, &path); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewInterviewLogic(r.Context(), svcCtx)
		resp, err := l.GetSession(userId, path.ID)
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

func InterviewSessionDeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		var path sessionIdPathReq
		if err := httpx.Parse(r, &path); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewInterviewLogic(r.Context(), svcCtx)
		if err := l.DeleteSession(userId, path.ID); err != nil {
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

func InterviewSessionEndHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		var path sessionIdPathReq
		if err := httpx.Parse(r, &path); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewInterviewLogic(r.Context(), svcCtx)
		if err := l.EndSession(userId, path.ID); err != nil {
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
