package middleware

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
)

type AdminMiddleware struct{}

func NewAdminMiddleware() *AdminMiddleware {
	return &AdminMiddleware{}
}

func (m *AdminMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value("role").(string)
		if role != "admin" {
			httpx.WriteJson(w, http.StatusForbidden, map[string]any{
				"error": "forbidden",
			})
			return
		}

		next(w, r)
	}
}
