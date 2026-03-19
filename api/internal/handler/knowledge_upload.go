package handler

import (
	"ai-gozero-agent/api/internal/logic"
	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"
	"errors"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func KnowledgeUploadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value("role").(string)
		if role != "admin" {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
			return
		}

		// 获取文件
		file, header, err := r.FormFile("file")
		if err != nil {
			httpx.Error(w, err)
			return
		}
		defer file.Close()

		// 验证PDF
		if header.Header.Get("Content-Type") != "application/pdf" {
			httpx.Error(w, errors.New("仅支持PDF文件"))
			return
		}

		// 提取文本
		content, err := svcCtx.PdfClient.ExtractText(file, header.Filename)
		if err != nil {
			httpx.Error(w, err)
			return
		}

		// 获取标题（使用文件名）
		title := header.Filename
		// 调用Logic保存知识
		l := logic.NewKnowledgeUploadLogic(r.Context(), svcCtx)
		resp, err := l.KnowledgeUpload(&types.KnowledgeUploadReq{
			Title:        title,
			Content:      content,
			SplitterType: r.FormValue("splitter_type"),
		})
		if err != nil {
			httpx.Error(w, err)
		} else {
			httpx.OkJson(w, resp)
		}
	}
}
