package observability

import (
	"net/http"
	"os"

	"github.com/SkyAPM/go2sky"
	skyhttp "github.com/SkyAPM/go2sky/plugins/http"
	"github.com/SkyAPM/go2sky/reporter"
)

// SkyWalking is dormant in the standard profile. It opens an OAP connection and
// creates SW8-compatible HTTP server/client spans only when the tracing profile
// supplies SW_AGENT_COLLECTOR_BACKEND_SERVICES.
type SkyWalking struct {
	client     *http.Client
	middleware func(http.Handler) http.Handler
	close      func()
}

func New(serviceName string, baseClient *http.Client) (*SkyWalking, error) {
	backend := os.Getenv("SW_AGENT_COLLECTOR_BACKEND_SERVICES")
	if backend == "" {
		return &SkyWalking{client: baseClient, middleware: func(handler http.Handler) http.Handler { return handler }}, nil
	}
	reporterInstance, err := reporter.NewGRPCReporter(backend)
	if err != nil {
		return nil, err
	}
	tracer, err := go2sky.NewTracer(serviceName, go2sky.WithReporter(reporterInstance))
	if err != nil {
		reporterInstance.Close()
		return nil, err
	}
	client, err := skyhttp.NewClient(tracer, skyhttp.WithClient(baseClient))
	if err != nil {
		reporterInstance.Close()
		return nil, err
	}
	middleware, err := skyhttp.NewServerMiddleware(tracer)
	if err != nil {
		reporterInstance.Close()
		return nil, err
	}
	return &SkyWalking{client: client, middleware: middleware, close: func() { reporterInstance.Close() }}, nil
}

func (s *SkyWalking) Client() *http.Client                   { return s.client }
func (s *SkyWalking) Wrap(handler http.Handler) http.Handler { return s.middleware(handler) }
func (s *SkyWalking) Close() {
	if s.close != nil {
		s.close()
	}
}
