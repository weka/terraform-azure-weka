package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"weka-deployment/functions/clusterize"
	"weka-deployment/functions/clusterize_finalization"
	"weka-deployment/functions/debug"
	"weka-deployment/functions/deploy"
	"weka-deployment/functions/fetch"
	"weka-deployment/functions/join_finalization"
	"weka-deployment/functions/scale_down"
	"weka-deployment/functions/scale_up"
	"weka-deployment/functions/status"
	"weka-deployment/functions/terminate"
	"weka-deployment/functions/transient"
)

func main() {
	customHandlerPort, exists := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT")
	if !exists {
		customHandlerPort = "8080"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/clusterize", clusterize.Handler)
	mux.HandleFunc("/clusterize_finalization", clusterize_finalization.Handler)
	mux.HandleFunc("/status", status.Handler)
	mux.HandleFunc("/debug", debug.Handler)
	mux.HandleFunc("/scale_up", scale_up.Handler)
	mux.HandleFunc("/fetch", fetch.Handler)
	mux.HandleFunc("/deploy", deploy.Handler)
	mux.HandleFunc("/join_finalization", join_finalization.Handler)
	mux.HandleFunc("/scale_down", scale_down.Handler)
	mux.HandleFunc("/terminate", terminate.Handler)
	mux.HandleFunc("/transient", transient.Handler)
	fmt.Println("Go server Listening on: ", customHandlerPort)
	log.Fatal(http.ListenAndServe(":"+customHandlerPort, mux))
}
