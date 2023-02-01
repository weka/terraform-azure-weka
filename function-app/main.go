package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"weka-deployment/functions/clusterize"
	"weka-deployment/functions/clusterize_finalization"
	"weka-deployment/functions/status"
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
	fmt.Println("Go server Listening on: ", customHandlerPort)
	log.Fatal(http.ListenAndServe(":"+customHandlerPort, mux))
}
