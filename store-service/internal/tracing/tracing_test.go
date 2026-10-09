package tracing

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
)

func TestInitTracerSendsSpansOverOTLP(t *testing.T) {
	paths := make(chan string, 1)
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case paths <- r.URL.Path:
		default:
		}
	}))
	defer collector.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", collector.URL)

	shutdown, err := InitTracer(context.Background(), "test-service")
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	_, span := otel.Tracer("test").Start(context.Background(), "ping")
	span.End()
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	select {
	case path := <-paths:
		if path != "/v1/traces" {
			t.Fatalf("spans sent to %q, want /v1/traces", path)
		}
	default:
		t.Fatal("no spans reached the collector")
	}
}

func TestInitTracerExportsNothingWhenDisabled(t *testing.T) {
	requests := make(chan struct{}, 1)
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case requests <- struct{}{}:
		default:
		}
	}))
	defer collector.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", collector.URL)
	t.Setenv("OTEL_TRACES_EXPORTER", "none")

	shutdown, err := InitTracer(context.Background(), "test-service")
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	_, span := otel.Tracer("test").Start(context.Background(), "ping")
	span.End()
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	select {
	case <-requests:
		t.Fatal("spans were exported although OTEL_TRACES_EXPORTER=none")
	default:
	}
}

func TestHealthChecksAreNotTraced(t *testing.T) {
	exported := make(chan string, 10)
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		exported <- string(body)
	}))
	defer collector.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", collector.URL)
	t.Setenv("OTEL_TRACES_EXPORTER", "")

	shutdown, err := InitTracer(context.Background(), "test-service")
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(otelgin.Middleware("test-service", otelgin.WithFilter(Traced)))
	router.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/api/v1/things", func(c *gin.Context) { c.Status(http.StatusOK) })
	for _, path := range []string{"/health", "/api/v1/things"} {
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	var all string
	for done := false; !done; {
		select {
		case body := <-exported:
			all += body
		default:
			done = true
		}
	}
	if !strings.Contains(all, "/api/v1/things") {
		t.Error("the API request was not traced")
	}
	if strings.Contains(all, "/health") {
		t.Error("the health check was traced")
	}
}
