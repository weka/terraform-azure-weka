package common

import (
	"net/http"
	"strings"

	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

func FuncNameHandler(fieldKey string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := zerolog.Ctx(r.Context())
			log.UpdateContext(func(c zerolog.Context) zerolog.Context {
				pathParts := strings.Split(r.URL.Path, "/")
				funcName := pathParts[len(pathParts)-1]
				return c.Str(fieldKey, funcName)
			})
			next.ServeHTTP(w, r)
		})
	}
}

func AzureFuncInvocationIdHandler(fieldKey string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ref := r.Header.Get("X-Azure-Functions-Invocationid"); ref != "" {
				log := zerolog.Ctx(r.Context())
				log.UpdateContext(func(c zerolog.Context) zerolog.Context {
					return c.Str(fieldKey, ref)
				})
			}
			next.ServeHTTP(w, r)
		})
	}
}

func LoggingMiddleware(next http.HandlerFunc) http.Handler {
	logger := NewLogger()
	c := alice.New()
	c = c.Append(hlog.NewHandler(*logger.Logger))
	c = c.Append(FuncNameHandler("function"))
	c = c.Append(AzureFuncInvocationIdHandler("InvocationId"))
	// get final handler
	return c.ThenFunc(next)
}
