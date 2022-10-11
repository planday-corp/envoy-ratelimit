package metrics

import (
	"context"
	"time"

	aiworker "github.com/envoyproxy/ratelimit/src/ai_worker"
	stats "github.com/lyft/gostats"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type serverMetrics struct {
	totalRequests stats.Counter
	responseTime  stats.Timer
}

// ServerReporter reports server-side metrics for ratelimit gRPC server
type ServerReporter struct {
	scope    stats.Scope
	aiWorker aiworker.AiWorker
}

func newServerMetrics(scope stats.Scope, fullMethod string) *serverMetrics {
	_, methodName := splitMethodName(fullMethod)
	ret := serverMetrics{}
	ret.totalRequests = scope.NewCounter(methodName + ".total_requests")
	ret.responseTime = scope.NewTimer(methodName + ".response_time")
	return &ret
}

// NewServerReporter returns a ServerReporter object.
func NewServerReporter(scope stats.Scope, aiWorker aiworker.AiWorker) *ServerReporter {
	return &ServerReporter{
		scope:    scope,
		aiWorker: aiWorker,
	}
}

// UnaryServerInterceptor is a gRPC server-side interceptor that provides server metrics for Unary RPCs.
func (r *ServerReporter) UnaryServerInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		defer func() {

			queue := *r.aiWorker.GetRequestQueue()
			queue <- aiworker.NewTrackRequest("POST", "/test", time.Since(start), "200")
		}()

		s := newServerMetrics(r.scope, info.FullMethod)
		s.totalRequests.Inc()
		resp, err := handler(ctx, req)
		logger.Info(resp)
		s.responseTime.AddValue(float64(time.Since(start).Milliseconds()))
		return resp, err
	}
}
