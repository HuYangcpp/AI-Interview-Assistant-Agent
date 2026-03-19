package handler

import (
	"net/http"

	"ai-gozero-agent/api/internal/logic"
	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func AuthLoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LoginReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewAuthLogic(r.Context(), svcCtx)
		resp, err := l.Login(&req)
		if err != nil {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, resp)
	}
}

func AuthRegisterHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RegisterReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewAuthLogic(r.Context(), svcCtx)
		resp, err := l.Register(&req)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, resp)
	}
}

func AuthProfileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, ok := r.Context().Value("userId").(int64)
		if !ok || userId <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
			return
		}

		l := logic.NewAuthLogic(r.Context(), svcCtx)
		resp, err := l.Profile(userId)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, resp)
	}
}

func AuthPasswordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, ok := r.Context().Value("userId").(int64)
		if !ok || userId <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
			return
		}

		var req types.ChangePasswordReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewAuthLogic(r.Context(), svcCtx)
		if err := l.ChangePassword(userId, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, map[string]any{"msg": "ok"})
	}
}
