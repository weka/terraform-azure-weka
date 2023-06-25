package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
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
	periodic "weka-deployment/periodic_tasks"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/hlog"
	"github.com/weka/go-cloud-lib/logging"
)

var (
	logger     *logging.Logger
	httpServer *http.Server

	scaleUpTask   periodic.ScaleUpPeriodicTask
	scaleDownTask periodic.ScaleDownPeriodicTask
)

func init() {
	logger = logging.NewLogger()

	scaleUpTask = periodic.NewScaleUpPeriodicTask(time.Second * 10)
	scaleDownTask = periodic.NewScaleDownPeriodicTask(time.Second * 10)
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
	r.Use(hlog.RequestIDHandler("req_id", middleware.RequestIDHeader))
	r.Use(hlog.RemoteAddrHandler("remote_addr"))
	r.Use(middleware.Recoverer)
}

func main() {
	ctx := context.Background()

	// run periodic tasks
	go scaleUpTask.Run(ctx)
	go scaleDownTask.Run(ctx)

	// run hrrp server
	go runHttpServer(ctx)

	// cleanup stuff on interrupt
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		<-c
		cleanup(ctx)
		wg.Done()
	}()

	wg.Wait()
}

func runHttpServer(ctx context.Context) {
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
	httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%s", host, port),
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  10 * time.Second,
	}
	logger.Info().Msgf("Go server Listening on: %v", port)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal().Msgf("http server failure: %v", err)
	}
	logger.Info().Msg("HTTP Server Stopped")
}

func cleanup(ctx context.Context) {
	logger.Info().Msg("Cleanup")
	// stop http server
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx)
	// stop periodic tasks
	scaleUpTask.Stop()
	scaleDownTask.Stop()
}
