package main

import (
	"net/http"
	"os"
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

	"github.com/weka/go-cloud-lib/logging"
)

var (
	logger *logging.Logger
)

func init() {
	logger = logging.NewLogger()
}

func main() {
	customHandlerPort, exists := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT")
	if !exists {
		customHandlerPort = "8080"
	}
	mux := http.NewServeMux()
	mux.Handle("/clusterize", logging.LoggingMiddleware(clusterize.Handler))
	mux.Handle("/clusterize_finalization", logging.LoggingMiddleware(clusterize_finalization.Handler))
	mux.Handle("/status", logging.LoggingMiddleware(status.Handler))
	mux.Handle("/debug", logging.LoggingMiddleware(debug.Handler))
	mux.Handle("/scale_up", logging.LoggingMiddleware(scale_up.Handler))
	mux.Handle("/fetch", logging.LoggingMiddleware(fetch.Handler))
	mux.Handle("/deploy", logging.LoggingMiddleware(deploy.Handler))
	mux.Handle("/join_finalization", logging.LoggingMiddleware(join_finalization.Handler))
	mux.Handle("/scale_down", logging.LoggingMiddleware(scale_down.Handler))
	mux.Handle("/terminate", logging.LoggingMiddleware(terminate.Handler))
	mux.Handle("/transient", logging.LoggingMiddleware(transient.Handler))
	mux.Handle("/resize", logging.LoggingMiddleware(resize.Handler))
	mux.Handle("/report", logging.LoggingMiddleware(report.Handler))
	mux.Handle("/protect", logging.LoggingMiddleware(protect.Handler))
	logger.Info().Msgf("Go server Listening on: %v", customHandlerPort)
	logger.Fatal().Err(http.ListenAndServe(":"+customHandlerPort, mux)).Send()
}
