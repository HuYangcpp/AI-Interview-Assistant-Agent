package middleware

import (
	"context"
	"errors"
	"net/http"

	"ai-gozero-agent/api/internal/utils"

	"github.com/golang-jwt/jwt/v4/request"
	"github.com/zeromicro/go-zero/rest/httpx"
)

type JwtMiddleware struct {
	AccessSecret string
}

func NewJwtMiddleware(accessSecret string) *JwtMiddleware {
	return &JwtMiddleware{AccessSecret: accessSecret}
}

func (m *JwtMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	extractor := request.MultiExtractor{
		request.BearerExtractor{},
		request.ArgumentExtractor{"token"},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := extractor.ExtractToken(r)
		if err != nil {
			respondUnauthorized(w)
			return
		}

		claims, err := utils.ParseJWTToken(m.AccessSecret, tokenString)
		if err != nil {
			respondUnauthorized(w)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, "userId", claims.UserId)
		ctx = context.WithValue(ctx, "role", claims.Role)

		next(w, r.WithContext(ctx))
	}
}

func respondUnauthorized(w http.ResponseWriter) {
	httpx.WriteJson(w, http.StatusUnauthorized, map[string]any{
		"error": errors.New("unauthorized").Error(),
	})
}
