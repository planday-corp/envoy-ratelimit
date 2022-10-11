package aiworker

import (
	"time"

	"github.com/envoyproxy/ratelimit/src/settings"
	"github.com/microsoft/ApplicationInsights-Go/appinsights"
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
	isRunning    bool
}

func NewAiWorker(s settings.Settings) AiWorker {

	ret := new(aiWorker)

	if s.ApplicationInsightInstrumentationKey != "" {
		ret.aiClient = appinsights.NewTelemetryClient(s.ApplicationInsightInstrumentationKey)
	}

	ret.requestQueue = make(chan TrackRequest)

	return ret
}

func (w *aiWorker) GetRequestQueue() *chan TrackRequest {
	return &w.requestQueue
}

func (w *aiWorker) Start() {
	w.isRunning = true
	for w.isRunning {
		request := <-w.requestQueue

		w.aiClient.TrackRequest(request.method, request.url, request.duration, request.statusCode)
	}
}

func (w *aiWorker) Stop() {
	w.isRunning = false
}
