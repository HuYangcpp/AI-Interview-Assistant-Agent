package handler

import (
	"net/http"

	"ai-gozero-agent/api/internal/model"
	"ai-gozero-agent/api/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// InterviewSessionMessagesHandler 获取会话历史消息
func InterviewSessionMessagesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
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

		session, err := svcCtx.InterviewSessionModel.FindById(r.Context(), path.ID)
		if err != nil {
			status := http.StatusInternalServerError
			if err == model.ErrNotFound {
				status = http.StatusNotFound
			}
			httpx.WriteJson(w, status, map[string]any{"error": err.Error()})
			return
		}

		student, err := svcCtx.StudentModel.FindByUserId(r.Context(), userId)
		if err != nil || session.StudentID != student.ID {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		messages, err := svcCtx.VectorStore.GetMessages(r.Context(), session.ChatID, 100)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]any{"messages": messages})
	}
}
