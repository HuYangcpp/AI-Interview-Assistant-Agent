package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"ai-gozero-agent/api/internal/logic"
	"ai-gozero-agent/api/internal/model"
	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// InterviewChatSSEHandler 面试对话 SSE 流
func InterviewChatSSEHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setSSEHeader(w)
		flusher, _ := w.(http.Flusher)

		userId, _ := r.Context().Value("userId").(int64)
		if userId <= 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		// resolve studentId from userId
		student, err := svcCtx.StudentModel.FindByUserId(r.Context(), userId)
		if err != nil {
			status := http.StatusForbidden
			if err == model.ErrNotFound {
				status = http.StatusNotFound
			}
			httpx.WriteJson(w, status, map[string]any{"error": err.Error()})
			return
		}

		var req types.InterviewChatReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		req.Message = strings.TrimSpace(req.Message)
		if req.SessionId <= 0 || req.Message == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": "sessionId/message required"})
			return
		}

		session, err := svcCtx.InterviewSessionModel.FindById(r.Context(), req.SessionId)
		if err != nil {
			status := http.StatusBadRequest
			if err == model.ErrNotFound {
				status = http.StatusNotFound
			}
			httpx.WriteJson(w, status, map[string]any{"error": err.Error()})
			return
		}
		if session.StudentID != student.ID {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}
		if session.Status != "ongoing" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": "session completed"})
			return
		}

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		l := logic.NewChatLogic(ctx, svcCtx)
		respChan, err := l.Chat(session, req.Message)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case resp, ok := <-respChan:
				if !ok {
					_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
					flusher.Flush()
					return
				}

				if resp.IsLast {
					_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
					flusher.Flush()
					return
				}

				safeContent := strings.ReplaceAll(resp.Content, "\n", "\\n")
				safeContent = strings.ReplaceAll(safeContent, "\r", "\\r")
				if _, err := fmt.Fprintf(w, "data: %s\n\n", safeContent); err != nil {
					logx.Errorf("sse write error: %v", err)
					return
				}
				flusher.Flush()
			}
		}
	}
}
