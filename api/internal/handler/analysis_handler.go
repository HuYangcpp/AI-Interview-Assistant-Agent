package handler

import (
	"net/http"

	"ai-gozero-agent/api/internal/logic"
	"ai-gozero-agent/api/internal/model"
	"ai-gozero-agent/api/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

type analysisSessionPathReq struct {
	ID int64 `path:"id"`
}

type analysisGeneratePathReq struct {
	SessionId int64 `path:"sessionId"`
}

func AnalysisSessionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		var path analysisSessionPathReq
		if err := httpx.Parse(r, &path); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewAnalysisLogic(r.Context(), svcCtx)
		resp, err := l.GetSessionAnalysis(userId, path.ID)
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

func AnalysisOverviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		l := logic.NewAnalysisLogic(r.Context(), svcCtx)
		resp, err := l.GetOverview(userId)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		httpx.OkJson(w, resp)
	}
}

func AnalysisTrendHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		l := logic.NewAnalysisLogic(r.Context(), svcCtx)
		resp, err := l.GetTrend(userId, 20)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		httpx.OkJson(w, resp)
	}
}

func AnalysisGenerateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		var path analysisGeneratePathReq
		if err := httpx.Parse(r, &path); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		l := logic.NewAnalysisLogic(r.Context(), svcCtx)
		if err := l.GenerateAnalysis(userId, path.SessionId); err != nil {
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
