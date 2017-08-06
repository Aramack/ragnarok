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
  ID          int               `json:"id,omitempty"`
  ThreadCount int               `json:"threadcount,omitempty"`
  URL         []string          `json:"url,omitempty"`
  Iteration   int               `json:"iteration,omitempty"`
  Headers     map[string]string `json:"headers,omitempty"`
}

type TrafficManager struct {
  CurrentID   int
  TrafficList []*Traffic
  sync.Mutex
}

// registerTraffic takes a user defined traffic request and registers it in the manager.
func registerTraffic(traffic *Traffic) {
  apiTrafficManager.Lock()
  traffic.ID = apiTrafficManager.CurrentID
  apiTrafficManager.CurrentID++
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

  go model.HTTPLoadBalancer(loadBalancerChannel, finishedChannel, traffic.ThreadCount, traffic.Headers)
  fmt.Println("New dispatcher running with " + strconv.Itoa(traffic.ThreadCount) + " threads")
  for itt := 0; itt < traffic.Iteration; itt++ {
    for _, url := range traffic.URL {
      loadBalancerChannel <- url
    }
  }


  //Close the channel to the load balancer to indicate no more requests incoming.
  close(loadBalancerChannel)
  //Wait for the load balancer to tell us it has handled all the requests.
  <-finishedChannel
}


// ApiTrafficCreate reads the request, creates a dispatcher and writes the result to w.
func ApiTrafficCreate(w http.ResponseWriter, req *http.Request){
  setResponseHeaders(w)

  //Create traffic struct based upon the JSON POST data:
  var traffic Traffic
  err := json.NewDecoder(req.Body).Decode(&traffic)
  if err != nil {
    //Bad request
    w.WriteHeader(http.StatusBadRequest)
    responseMessage, _ := json.Marshal(map[string]string{"error": "Malformed JSON", "errormsg": err.Error()})
    w.Write(responseMessage)
    return
  }

  //Create a traffic dispatcher:
  workerIDChannel := make(chan int)
  go trafficDispatcher(&traffic, workerIDChannel)

  //Wait for the traffic dispatcher to register its workerID
  workerID, _ := <- workerIDChannel

  //Return the workerID to the consumerID:
  returnJSON := make(map[string]int)
  returnJSON["workerID"] = workerID
  w.WriteHeader(http.StatusOK)
  json.NewEncoder(w).Encode(returnJSON)
}

// ApiTraffic returns the traffic object that was returned
func ApiTraffic(w http.ResponseWriter, req *http.Request){
  setResponseHeaders(w)
  params := mux.Vars(req)
  requestedID, err := strconv.Atoi(params["id"])
  if err != nil {
    //Bad request
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
  w.WriteHeader(http.StatusNotFound)
}

// Healthcheck to check if the service is running correctly.
func Healthcheck(w http.ResponseWriter, req *http.Request){
  setResponseHeaders(w)
  if (!HealthcheckStatus) {
    w.WriteHeader(http.StatusServiceUnavailable)
  } else {
    w.WriteHeader(http.StatusOK)
  }
}

// HealthcheckAction is used to change the status of the healthcheck.
func HealthcheckAction(w http.ResponseWriter, req *http.Request){
  setResponseHeaders(w)
  params := mux.Vars(req)
  action := params["action"]

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

// setResponseHeaders is used to set the default http response headers.
func setResponseHeaders(rw http.ResponseWriter){
  rw.Header().Set("Connection", "close") //Connection closed
  rw.Header().Set("Server", runtime.Version()) //Golang version
  rw.Header().Set("Content-Type", "application/json") //JSon Response
}

// init function for the api controller.
func init() {
  apiTrafficManager = TrafficManager {CurrentID: 0, TrafficList: []*Traffic{}, }
}
