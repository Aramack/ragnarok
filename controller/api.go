package controller

import (
  "encoding/json"
  "fmt"
  "net/http"
  "runtime"
  "sync"
  "strconv"

  "github.com/aramack/ragnarok/model"
  "github.com/gorilla/mux"
)


// healthcheckStatus is used to determine if the server should pass or fail health checks.
var HealthcheckStatus bool = false

// apiTrafficManager is a structure for keeping track of the running traffic pools
var apiTrafficManager TrafficManager

type Traffic struct {
  ID          int      `json:"id,omitempty"`
  ThreadCount int      `json:"threadcount,omitempty"`
  URL         []string `json:"url,omitempty"`
}

type TrafficManager struct {
  CurrentID   int
  TrafficList []*Traffic
  sync.Mutex
}

// registerTraffic takes a user defined traffic request and registers it in the master class.
func registerTraffic(traffic *Traffic) {
  apiTrafficManager.Lock()
  traffic.ID = apiTrafficManager.CurrentID
  apiTrafficManager.CurrentID++
  //@todo: Figure out how to add the new struct to the manager's list
  apiTrafficManager.TrafficList = append(apiTrafficManager.TrafficList, traffic)
  apiTrafficManager.Unlock()
}


// trafficDispatcher takes the traffic object created by the user an creates a threadpool to handle it.
// It returns the ID of the consumer via the consumerIDChannel
func trafficDispatcher(traffic *Traffic, consumerIDChannel chan<- int) {
  loadBalancerChannel := make(chan string)
  finishedChannel := make(chan bool)
  // Register the user defined traffic struct with the manager.
  registerTraffic(traffic)

  consumerIDChannel <- traffic.ID
  close(consumerIDChannel)

  go model.HttpLoadBalancer(loadBalancerChannel, finishedChannel, traffic.ThreadCount)
  fmt.Println(traffic.ThreadCount)
  for _, url := range traffic.URL {
    loadBalancerChannel <- url
  }

  //Close the channel to the load balancer to indicate no more requests incoming.
  close(loadBalancerChannel)
  //Wait for the load balancer to tell us it has handled all the requests.
  <-finishedChannel
}


// ApiTrafficCreate reads the request, creates a dispatcher and writes the result to w.
func ApiTrafficCreate(w http.ResponseWriter, req *http.Request){
  //Create traffic struct based upon the JSON POST data:
  var traffic Traffic
  _ = json.NewDecoder(req.Body).Decode(&traffic)

  //Create a traffic dispatcher:
  workerIDChannel := make(chan int)
  go trafficDispatcher(&traffic, workerIDChannel)

  //Wait for the traffic dispatcher to register its workerID
  workerID, _ := <- workerIDChannel

  //Return the workerID to the consumerID:
  returnJSON := make(map[string]int)
  returnJSON["workerID"] = workerID
  json.NewEncoder(w).Encode(returnJSON)
}

// ApiTraffic returns the traffic object that was returned
func ApiTraffic(w http.ResponseWriter, req *http.Request){
  params := mux.Vars(req)
  requestedID, err := strconv.Atoi(params["id"])
  if err != nil {
    //Bad request
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusBadRequest)
    responseMessage, _ := json.Marshal(map[string]string{"error": "Invalid traffic ID"})
    w.Write(responseMessage)
    return
  }
  //This will fail until manager registration is working:
  for _, traffic := range apiTrafficManager.TrafficList {
    if traffic.ID == requestedID {
      returnJSON := make(map[string]int)
      returnJSON["threadcount"] = traffic.ThreadCount
      json.NewEncoder(w).Encode(returnJSON)
      return
    }
  }
  //Not found:
  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(http.StatusNotFound)
}

// Healthcheck to check if the service is running correctly.
func Healthcheck(w http.ResponseWriter, req *http.Request){
  w.Header().Set("Connection", "close")
  w.Header().Set("Server", runtime.Version())
  if (!HealthcheckStatus) {
    w.WriteHeader(http.StatusServiceUnavailable)
  } else {
    w.WriteHeader(http.StatusOK)
  }
}

func HealthcheckAction(w http.ResponseWriter, req *http.Request){
  params := mux.Vars(req)
  action := params["action"]
  w.Header().Set("Connection", "close")
  w.Header().Set("Server", runtime.Version())
  if (action == "up") {
    HealthcheckStatus = true
  } else if (action == "down") {
    HealthcheckStatus = false
  }
  if (!HealthcheckStatus) {
    w.WriteHeader(http.StatusServiceUnavailable)
  } else {
    w.WriteHeader(http.StatusOK)
  }
}

// init function for the api controller.
func init() {
  apiTrafficManager = TrafficManager {CurrentID: 0, TrafficList: []*Traffic{}, }
}
