package aiworker

import (
	"time"

	"github.com/envoyproxy/ratelimit/src/settings"
	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	logger "github.com/sirupsen/logrus"
)

func NewTrackRequest(method string, url string, duration time.Duration, statusCode string) TrackRequest {
	return TrackRequest{
		method:     method,
		url:        url,
		duration:   duration,
		statusCode: statusCode,
	}
}

type aiWorker struct {
	requestQueue chan TrackRequest
	aiClient     appinsights.TelemetryClient
	quitChannel  chan struct{}
}

func NewAiWorker(s settings.Settings) AiWorker {

	ret := new(aiWorker)

	if s.ApplicationInsightInstrumentationKey != "" {
		ret.aiClient = appinsights.NewTelemetryClient(s.ApplicationInsightInstrumentationKey)
	}

	ret.requestQueue = make(chan TrackRequest)
	ret.quitChannel = make(chan struct{})

	return ret
}

func (w *aiWorker) GetRequestQueue() *chan TrackRequest {
	return &w.requestQueue
}

func (w *aiWorker) Start() {
	go func() {
		for {
			select {
			case request := <-w.requestQueue:
				logger.Info("Getting request from queue")
				w.aiClient.TrackRequest(request.method, request.url, request.duration, request.statusCode)
			case <-w.quitChannel:
				logger.Info("Application Insight Worker hit quit channel")
				return
			}
		}
	}()

	logger.Info("Started Application Insight Worker")
}

func (w *aiWorker) Stop() {
	close(w.quitChannel)
}
