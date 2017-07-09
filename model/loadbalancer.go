package model

import (
  "fmt"
  "net/http"
  "time"
)

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
func HttpLoadBalancer(
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

