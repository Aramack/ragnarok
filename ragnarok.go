// A simple application for load testing how much traffic a new service can handle.
package main

import (
  "net/http"
  "github.com/aramack/ragnarok/controller"
  "github.com/gorilla/mux"
)

func main() {
  apiRouter := mux.NewRouter()
  apiRouter.HandleFunc("/api/traffic", controller.ApiTrafficCreate).Methods("POST")
  apiRouter.HandleFunc("/api/traffic/{id:[0-9]+}", controller.ApiTraffic).Methods("GET")
  apiRouter.HandleFunc("/healthcheck", controller.Healthcheck).Methods("HEAD")
  apiRouter.HandleFunc("/healthcheck/{action:[a-z]+}", controller.HealthcheckAction).Methods("POST")
  controller.HealthcheckStatus = true
  http.ListenAndServe(":2626", apiRouter)
}
