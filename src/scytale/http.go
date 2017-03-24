package main

import (
	"github.com/Comcast/webpa-common/logging"
	"io/ioutil"
	"net/http"
	"time"
)

type Send func(inFunc func(workerID int)) error

// Below is the struct that will implement our ServeHTTP method
type ServerHandler struct {
	logging.Logger
	scytaleHandler RequestHandler
	scytaleHealth  HealthTracker
	doJob          Send
}

func (sh *ServerHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	sh.Info("Receiving incoming post...")

	timeStamps := ScytaleTimestamps{
		TimeReceived: time.Now(),
	}

	myPayload, err := ioutil.ReadAll(request.Body)
	if err != nil {
		statusMsg := "Unable to retrieve the request body: " + err.Error() + ".\n"
		response.WriteHeader(http.StatusBadRequest)
		response.Write([]byte(statusMsg))
		return
	}

	var contentType string
	if value, ok := request.Header["Content-Type"]; ok {
		if len(value) == 1 {
			contentType = value[0]
			switch contentType {
			case "application/json":
			case "application/wrp":
			default:
				response.WriteHeader(http.StatusBadRequest)
				response.Write([]byte("Only Content-Type values of \"application/json\" or \"application/wrp\" are supported.\n"))
			}
		} else {
			response.WriteHeader(http.StatusBadRequest)
			response.Write([]byte("Content-Type cannot have more than one specification.\n"))
		}
	} else {
		response.WriteHeader(http.StatusBadRequest)
		response.Write([]byte("Content-Type must be set in the header.\n"))
	}

	if contentType == "" {
		return
	}

	targetURL := request.URL.String()

	scytaleRequest := ScytaleRequest{
		Payload:     myPayload,
		ContentType: contentType,
		TargetURL:   targetURL,
		Timestamps:  timeStamps,
	}

	scytaleRequest.Timestamps.TimeAccepted = time.Now()

	err = sh.doJob(func(workerID int) { sh.scytaleHandler.HandleRequest(workerID, scytaleRequest) })
	if err != nil {
		// return a 408
		response.WriteHeader(http.StatusRequestTimeout)
		response.Write([]byte("Unable to handle request at this time.\n"))
	} else {
		// return a 202
		response.WriteHeader(http.StatusAccepted)
		response.Write([]byte("Request placed on to queue.\n"))
		sh.scytaleHealth.IncrementBucket(len(myPayload))
	}
}

type ProfileHandler struct {
	logging.Logger
}

// TODO: temporarily adding this to check and see if we're getting what we expect
func (ph *ProfileHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()

	ph.Info("Receiving request for server stats...")

	response.WriteHeader(http.StatusOK)
	response.Write([]byte("Placeholder.\n"))
}
