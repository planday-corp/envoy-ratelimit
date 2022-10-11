package aiworker

import "time"

type TrackRequest struct {
	method     string
	url        string
	duration   time.Duration
	statusCode string
}

type AiWorker interface {
	Start()

	Stop()

	GetRequestQueue() *chan TrackRequest
}
