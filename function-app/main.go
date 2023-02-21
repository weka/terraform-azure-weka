package main

import (
	"net/http"
	"os"
	"weka-deployment/common"
	"weka-deployment/functions/clusterize"
	"weka-deployment/functions/clusterize_finalization"
	"weka-deployment/functions/debug"
	"weka-deployment/functions/deploy"
	"weka-deployment/functions/fetch"
	"weka-deployment/functions/join_finalization"
	"weka-deployment/functions/resize"
	"weka-deployment/functions/scale_down"
	"weka-deployment/functions/scale_up"
	"weka-deployment/functions/status"
	"weka-deployment/functions/terminate"
	"weka-deployment/functions/transient"
)

var (
	logger *common.Logger
)

func init() {
	logger = common.NewLogger()
}

func main() {
	customHandlerPort, exists := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT")
	if !exists {
		customHandlerPort = "8080"
	}
	mux := http.NewServeMux()
	mux.Handle("/clusterize", common.LoggingMiddleware(clusterize.Handler))
	mux.Handle("/clusterize_finalization", common.LoggingMiddleware(clusterize_finalization.Handler))
	mux.Handle("/status", common.LoggingMiddleware(status.Handler))
	mux.Handle("/debug", common.LoggingMiddleware(debug.Handler))
	mux.Handle("/scale_up", common.LoggingMiddleware(scale_up.Handler))
	mux.Handle("/fetch", common.LoggingMiddleware(fetch.Handler))
	mux.Handle("/deploy", common.LoggingMiddleware(deploy.Handler))
	mux.Handle("/join_finalization", common.LoggingMiddleware(join_finalization.Handler))
	mux.Handle("/scale_down", common.LoggingMiddleware(scale_down.Handler))
	mux.Handle("/terminate", common.LoggingMiddleware(terminate.Handler))
	mux.Handle("/transient", common.LoggingMiddleware(transient.Handler))
	mux.Handle("/resize", common.LoggingMiddleware(resize.Handler))
	logger.Info().Msgf("Go server Listening on: %v", customHandlerPort)
	logger.Fatal().Err(http.ListenAndServe(":"+customHandlerPort, mux)).Send()
}
