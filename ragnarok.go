// A simple application for load testing how much traffic a new service can handle.
package main

import (
  "encoding/json"
  "fmt"
  "net/http"
  "runtime"
  "strconv"
  "sync"
  "time"

  "github.com/gorilla/mux"
)

// healthcheckStatus is used to determine if the server should pass or fail health checks.
var healthcheckStatus bool = false

// trafficManager is a struct for keeping track of the running traffic pools
var trafficManager TrafficManager

type Traffic struct {
  ID          int      `json:"id,omitempty"`
  ThreadCount int      `json:"threadcount,omitempty"`
  URL         []string `json:"url,omitempty"`
}

type TrafficManager struct {
  CurrentID   int
  TrafficList []Traffic
  sync.Mutex
}

// httpGet sends an http get request to the url parameter.
func httpGet(url string) {
  fmt.Println("Requesting: " + url)
  request, err := http.NewRequest(
    "GET",
    url,
    nil,
  )
  if (err != nil) {
    return
  }
  client := &http.Client{
    Timeout: time.Second * 30,
  }
  resp, err := client.Do(request)
  if err != nil {
    return
  }
  defer resp.Body.Close()
}

// httpRequestWorker takes all URLs sent to it via the urlChan passes them to httpGet.
// returns when the urlChan is closed.
func httpRequestWorker(urlChan <-chan string, finishedChan chan<- bool) {
  for {
    url, more := <-urlChan
    if more {
      httpGet(url)
      fmt.Println("Request finished: " + url)
    } else {
      finishedChan <- true
      return
    }
  }
}

// httpLoadBalancer is a function that takes a list of URLs from urlChan
// and distributes them between workerPoolSize number of threads.
// returns when urlChan is closed.
func httpLoadBalancer(
    urlChan <-chan string,
    finishedChan chan<- bool,
    workerPoolSize int) {
  //create workers
  var workerUrlChannels = make([]chan string, workerPoolSize)
  var workerFinishedChannels = make([]chan bool, workerPoolSize)

  for i := range workerUrlChannels {
    workerUrlChannels[i] = make(chan string)
    workerFinishedChannels[i] = make(chan bool)
  }
  for index, workerUrlChannels := range workerUrlChannels{
    go httpRequestWorker(workerUrlChannels, workerFinishedChannels[index])
  }

  //
  moreUrls := true
  for moreUrls{
    for _, workerUrlChannel := range workerUrlChannels {
      url, more := <- urlChan
      if more {
        workerUrlChannel <- url
      } else {
        moreUrls = more
      }
    }
  }
  for index, workerFinishedChannel := range workerFinishedChannels {
    close(workerUrlChannels[index])
    <-workerFinishedChannel
  }
  finishedChan <- true
}

// registerTraffic takes a user defined traffic request and registers it in the master class.
func registerTraffic(traffic *Traffic) {
  trafficManager.Lock()
  traffic.ID = trafficManager.CurrentID
  trafficManager.CurrentID++
  //@todo: Figure out how to add the new struct to the manager's list
  //trafficManager.TrafficList = append(trafficManager.TrafficList, traffic)
  trafficManager.Unlock()
}

//func addTrafficToManager(t *)

// trafficDispatcher takes the traffic object created by the user an creates a threadpool to handle it.
// It returns the ID of the consumer via the consumerIDChannel
func trafficDispatcher(traffic *Traffic, consumerIDChannel chan<- int) {
  loadBalancerChannel := make(chan string)
  finishedChannel := make(chan bool)
  // Register the user defined traffic struct with the manager.
  registerTraffic(traffic)

  consumerIDChannel <- traffic.ID
  close(consumerIDChannel)

  go httpLoadBalancer(loadBalancerChannel, finishedChannel, traffic.ThreadCount)
  fmt.Println(traffic.ThreadCount)
  for _, url := range traffic.URL {
    loadBalancerChannel <- url
  }

  //Close the channel to the load balancer to indicate no more requests incoming.
  close(loadBalancerChannel)
  //Wait for the load balancer to tell us it has handled all the requests.
  <-finishedChannel
}


// apiTrafficCreate reads the request, creates a dispatcher and writes the result to w.
func apiTrafficCreate(w http.ResponseWriter, req *http.Request){
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

func apiTraffic(w http.ResponseWriter, req *http.Request){
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
  for _, traffic := range trafficManager.TrafficList {
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

// healthcheck to check if the service is running correctly.
func healthcheck(w http.ResponseWriter, req *http.Request){
  w.Header().Set("Connection", "close")
  w.Header().Set("Server", runtime.Version())
  if (!healthcheckStatus) {
    w.WriteHeader(http.StatusServiceUnavailable)
  } else {
    w.WriteHeader(http.StatusOK)
  }
}

func main() {
  trafficManager = TrafficManager {CurrentID: 0, TrafficList: []Traffic{}, }
  api := mux.NewRouter()
  api.HandleFunc("/api/traffic", apiTrafficCreate).Methods("POST")
  api.HandleFunc("/api/traffic/{id:[0-9]+}", apiTraffic).Methods("GET")
  api.HandleFunc("/healthcheck", healthcheck).Methods("HEAD")
  healthcheckStatus = true
  http.ListenAndServe(":2626", api)
}
