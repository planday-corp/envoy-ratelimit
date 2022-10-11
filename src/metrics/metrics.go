package metrics

import (
	"context"
	"fmt"
	logger "github.com/sirupsen/logrus"
	"reflect"
	"time"

	envoy_service_ratelimit_v3 "github.com/envoyproxy/go-control-plane/envoy/service/ratelimit/v3"
	aiworker "github.com/envoyproxy/ratelimit/src/ai_worker"
	stats "github.com/lyft/gostats"
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

		s := newServerMetrics(r.scope, info.FullMethod)
		s.totalRequests.Inc()
		resp, err := handler(ctx, req)
		s.responseTime.AddValue(float64(time.Since(start).Milliseconds()))

		logger.Infof("Req is of type - %v - %v", reflect.TypeOf(req), reflect.TypeOf(req).Kind())
		logger.Infof("Resp is of type - %v - %v", reflect.TypeOf(resp), reflect.TypeOf(resp).Kind())

		go func() {
			rlReq, reqOk := req.(envoy_service_ratelimit_v3.RateLimitRequest)
			rlResp, respOk := resp.(envoy_service_ratelimit_v3.RateLimitResponse)
			logger.Infof("Is Req Ok - %t", reqOk)
			logger.Infof("Is Resp Ok - %t", respOk)
			if reqOk && respOk {

				var statusCode string
				switch rlResp.OverallCode {
				case 200:
					statusCode = "200"
					break
				case 429:
					statusCode = "429"
					break
				default:
					statusCode = "500"
				}

				ipValue := ""

				for _, descriptor := range rlReq.Descriptors {
					for _, entry := range descriptor.Entries {
						if entry.Key == "IP" {
							ipValue = entry.Value
						}
					}
				}

				logger.Infof("Tracking request from ip %s", ipValue)

				if ipValue != "" {
					queue := *r.aiWorker.GetRequestQueue()
					queue <- aiworker.NewTrackRequest("POST", fmt.Sprintf("IP_%s", ipValue), time.Since(start), statusCode)
				}
			}
		}()

		return resp, err
	}
}
