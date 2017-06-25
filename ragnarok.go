// A simple application for load testing how much traffic a new service can handle.
package main

import (
  "fmt"
  "net/http"
  "os"
  "bufio"
  "encoding/json"

  "github.com/gorilla/mux"
)

type Traffic struct {
  ID          int      `json:"id,omitempty"`
  ThreadCount int      `json:"threadcount,omitempty"`
  URL         []string `json:"url,omitempty"`
}

func http_get(url string) {
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
    //Timeout: time.Second * 30,
  }
  resp, err := client.Do(request)
  if err != nil {
    return
  }
  defer resp.Body.Close()
}

func http_request_worker(url_chan <-chan string, finished_chan chan<- bool) {
  for {
    url, more := <-url_chan
    if more {
      http_get(url)
      fmt.Println("Request finished: " + url)
    } else {
      finished_chan <- true
      return
    }
  }
}

func http_load_balancer(
    url_chan <-chan string,
    finished_chan chan<- bool,
    worker_pool_size int) {
  //create workers
  var worker_url_channels = make([]chan string, worker_pool_size)
  var worker_finished_channels = make([]chan bool, worker_pool_size)

  for i := range worker_url_channels {
    worker_url_channels[i] = make(chan string)
    worker_finished_channels[i] = make(chan bool)
  }
  for index, worker_url_channel := range worker_url_channels{
    go http_request_worker(worker_url_channel, worker_finished_channels[index])
  }

  more_urls := true
  for more_urls{
    for _, worker_url_channel := range worker_url_channels {
      url, more := <- url_chan
      if more {
        worker_url_channel <- url
      } else {
        more_urls = more
      }
    }
  }
  for index, worker_finished_channel := range worker_finished_channels {
    close(worker_url_channels[index])
    <-worker_finished_channel
  }
  finished_chan <- true
}

func read_url_source(raw_url_chan chan<- string, file_path string) {
  file_handle, _ := os.Open(file_path)
  reader := bufio.NewScanner(file_handle)
  reader.Split(bufio.ScanLines)
  for reader.Scan() {
    raw_url_chan <- reader.Text()
  }
  file_handle.Close()
  close(raw_url_chan)
}

func apiTrafficCreate(w http.ResponseWriter, req *http.Request){
  //params := mux.Vars(req)
  var traffic Traffic
  _ = json.NewDecoder(req.Body).Decode(&traffic)

  loadBalancerChannel := make(chan string)
  finishedChannel := make(chan bool)

  go http_load_balancer(loadBalancerChannel, finishedChannel, traffic.ThreadCount)
  fmt.Println(traffic.ThreadCount)
  for _, url := range traffic.URL {
    loadBalancerChannel <- url
  }

  //Close the channel to the load balancer to indicate no more requests incoming.
  close(loadBalancerChannel)
  //Wait for the load balancer to tell us it has handled all the requests.
  <-finishedChannel

  returnJSON := make(map[string]int)
  returnJSON["consumerID"] = 1

  json.NewEncoder(w).Encode(returnJSON)
}

func main() {
  api := mux.NewRouter()
  api.HandleFunc("/api/traffic", apiTrafficCreate).Methods("POST")
  http.ListenAndServe(":2626", api)
}
