package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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
