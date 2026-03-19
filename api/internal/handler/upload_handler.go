package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"ai-gozero-agent/api/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

type uploadFilePathReq struct {
	Category string `path:"category"`
	Filename string `path:"filename"`
}

var allowedUploadCategories = map[string]bool{
	"resumes":              true,
	"employment-evidences": true,
}

func UploadFileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var path uploadFilePathReq
		if err := httpx.Parse(r, &path); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}

		if !allowedUploadCategories[path.Category] {
			http.NotFound(w, r)
			return
		}

		filename := filepath.Base(path.Filename)
		if filename == "." || filename == "" || filename != path.Filename {
			http.NotFound(w, r)
			return
		}

		uploadDir := strings.TrimSpace(svcCtx.Config.UploadDir)
		if uploadDir == "" {
			http.NotFound(w, r)
			return
		}

		filePath := filepath.Join(uploadDir, path.Category, filename)
		if _, err := os.Stat(filePath); err != nil {
			http.NotFound(w, r)
			return
		}

		http.ServeFile(w, r, filePath)
	}
}
