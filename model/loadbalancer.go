package model
import (
  "fmt"
  "net/http"
  "time"
)

var headers map[string]string

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

  // Iterate over the http requests headers map and add set each.
  for httpHeaderKey, httpHeaderValue := range headers {
    request.Header.Set(httpHeaderKey, httpHeaderValue)
  }

  client := &http.Client{
    Timeout: time.Second * 30,
  }
  //Golang will follow 300 redirects be default.
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

// HTTPLoadBalancer is a function that takes url strings from urlChan
// and distributes them between workerPoolSize number of threads.
// Returns when urlChan is closed. Custom http request headers can be specified
// in the httpRequestHeaders map parameter
func HTTPLoadBalancer(
    urlChan <-chan string,
    finishedChan chan<- bool,
    workerPoolSize int,
    httpRequestHeaders map[string]string) {
  headers = httpRequestHeaders
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

  //While the workerUrlChannels has new URLs, send them to the a worker.
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

  //Wait for the workers to finish.
  for index, workerFinishedChannel := range workerFinishedChannels {
    close(workerUrlChannels[index])
    <-workerFinishedChannel
  }
  finishedChan <- true
}

