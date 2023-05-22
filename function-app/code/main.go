package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
	"weka-deployment/functions/clusterize"
	"weka-deployment/functions/clusterize_finalization"
	"weka-deployment/functions/debug"
	"weka-deployment/functions/deploy"
	"weka-deployment/functions/fetch"
	"weka-deployment/functions/join_finalization"
	"weka-deployment/functions/protect"
	"weka-deployment/functions/report"
	"weka-deployment/functions/resize"
	"weka-deployment/functions/scale_down"
	"weka-deployment/functions/scale_up"
	"weka-deployment/functions/status"
	"weka-deployment/functions/terminate"
	"weka-deployment/functions/transient"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/hlog"
	"github.com/weka/go-cloud-lib/logging"
)

var (
	logger *logging.Logger
)

func init() {
	logger = logging.NewLogger()
}

func InitRouter(ctx context.Context) chi.Router {
	r := chi.NewRouter()
	useMiddlewares(r)
	return r
}

func useMiddlewares(r chi.Router) {
	r.Use(hlog.NewHandler(*logger.Logger))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Stringer("url", r.URL).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Send()
	}))
	r.Use(logging.FuncNameHandler("function"))
	r.Use(hlog.RequestIDHandler("req_id", "Request-Id"))
	r.Use(middleware.Recoverer)
}

func main() {
	ctx := context.Background()

	r := InitRouter(ctx)
	r.Post("/clusterize", clusterize.Handler)
	r.Post("/deploy", deploy.Handler)
	r.Post("/clusterize_finalization", clusterize_finalization.Handler)
	r.Post("/status", status.Handler)
	r.Post("/debug", debug.Handler)
	r.Post("/scale_up", scale_up.Handler)
	r.Post("/fetch", fetch.Handler)
	r.Post("/join_finalization", join_finalization.Handler)
	r.Post("/scale_down", scale_down.Handler)
	r.Post("/terminate", terminate.Handler)
	r.Post("/transient", transient.Handler)
	r.Post("/resize", resize.Handler)
	r.Post("/report", report.Handler)
	r.Post("/protect", protect.Handler)

	host, exists := os.LookupEnv("HTTP_SERVER_HOST")
	if !exists {
		host = "localhost"
		os.Setenv("HTTP_SERVER_HOST", host)
	}
	port, exists := os.LookupEnv("HTTP_SERVER_PORT")
	if !exists {
		port = "8080"
		os.Setenv("HTTP_SERVER_PORT", port)
	}
	logger.Info().Msgf("Go server Listening on: %v", port)
	err := http.ListenAndServe(fmt.Sprintf("%s:%s", host, port), r)
	if err != nil {
		logger.Fatal().Err(err).Send()
	}
}
