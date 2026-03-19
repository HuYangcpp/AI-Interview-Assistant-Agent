package handler

import (
	"net/http"

	"ai-gozero-agent/api/internal/logic"
	"ai-gozero-agent/api/internal/model"
	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

type suggestionIdPathReq struct {
	ID int64 `path:"id"`
}

func SuggestionListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		l := logic.NewSuggestionLogic(r.Context(), svcCtx)
		resp, err := l.List(userId)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, resp)
	}
}

func SuggestionGenerateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		l := logic.NewSuggestionLogic(r.Context(), svcCtx)
		if err := l.GenerateSuggestionsForUser(userId); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		httpx.OkJson(w, &types.GenerateSuggestionsResp{Created: 4})
	}
}

func SuggestionReadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		var path suggestionIdPathReq
		if err := httpx.Parse(r, &path); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewSuggestionLogic(r.Context(), svcCtx)
		if err := l.MarkRead(userId, path.ID); err != nil {
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
