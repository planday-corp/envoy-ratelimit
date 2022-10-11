package aiworker

import (
	"sync"
	"time"

	"github.com/envoyproxy/ratelimit/src/settings"
	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	logger "github.com/sirupsen/logrus"
)

type concurrentSlice struct {
	sync.RWMutex
	items []TrackRequest
}

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
	ticker := time.NewTicker(5 * time.Second)
	cs := concurrentSlice{items: []TrackRequest{}}

	go func() {
		for {
			select {
			case <-ticker.C:
				cs.Lock()
				defer cs.Unlock()

				for _, item := range cs.items {
					w.aiClient.TrackRequest(item.method, item.url, item.duration, item.statusCode)
				}
			case <-w.quitChannel:
				ticker.Stop()
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case request := <-w.requestQueue:
				cs.Lock()
				defer cs.Unlock()

				cs.items = append(cs.items, request)
			case <-w.quitChannel:
				return
			}
		}
	}()

	logger.Info("Started Application Insight Worker")
}

func (w *aiWorker) Stop() {
	close(w.quitChannel)
}
