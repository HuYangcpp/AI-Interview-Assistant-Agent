package handler

import (
	"net/http"

	"ai-gozero-agent/api/internal/logic"
	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func StudentProfileGetHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
			return
		}

		l := logic.NewStudentLogic(r.Context(), svcCtx)
		resp, err := l.GetProfile(userId)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, resp)
	}
}

func StudentProfileUpdateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
			return
		}

		var req types.UpdateStudentProfileReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewStudentLogic(r.Context(), svcCtx)
		if err := l.UpdateProfile(userId, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, map[string]any{"msg": "ok"})
	}
}

func StudentResumeUploadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": "file required"})
			return
		}
		defer file.Close()

		l := logic.NewStudentLogic(r.Context(), svcCtx)
		url, err := l.UploadResume(userId, file, header)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, &types.ResumeUploadResp{ResumeURL: url})
	}
}

func StudentProfileChangeRequestsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
			return
		}

		l := logic.NewStudentLogic(r.Context(), svcCtx)
		resp, err := l.ListProfileChangeRequests(userId)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, resp)
	}
}

func StudentProfileChangeRequestCreateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
			return
		}

		var req types.StudentCreateProfileChangeRequestReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewStudentLogic(r.Context(), svcCtx)
		if err := l.CreateProfileChangeRequest(userId, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, map[string]any{"msg": "ok"})
	}
}
