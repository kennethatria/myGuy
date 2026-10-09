package proximity

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestCellSharedCases(t *testing.T) {
	raw, err := os.ReadFile("../../../shared/geo-cell-cases.json")
	require.NoError(t, err)
	var cases struct {
		Step  float64 `json:"step"`
		Cells []struct {
			Degrees float64 `json:"degrees"`
			Cell    int64   `json:"cell"`
		} `json:"cells"`
	}
	require.NoError(t, json.Unmarshal(raw, &cases))
	require.Equal(t, Step, cases.Step)
	for _, c := range cases.Cells {
		assert.Equal(t, c.Cell, Cell(c.Degrees), "cell of %v", c.Degrees)
	}
}

func f(v float64) *float64 { return &v }

func TestParse(t *testing.T) {
	none, err := Parse(nil, nil)
	assert.NoError(t, err)
	assert.Nil(t, none, "location is optional")

	got, err := Parse(f(0.3476), f(32.5842))
	require.NoError(t, err)
	assert.InDelta(t, 0.350, got.Lat, 1e-9)
	assert.InDelta(t, 32.585, got.Lng, 1e-9)

	for _, bad := range [][2]*float64{{f(1), nil}, {nil, f(1)}, {f(91), f(0)}, {f(0), f(-181)}, {f(math.NaN()), f(0)}} {
		_, err := Parse(bad[0], bad[1])
		assert.ErrorIs(t, err, ErrInvalidLocation)
	}
}

func TestClient(t *testing.T) {
	type call struct{ method, path, key, body string }
	calls := make(chan call, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		calls <- call{r.Method, r.URL.Path, r.Header.Get("X-Internal-API-Key"), string(body)}
		if r.URL.Path == "/internal/locations/task/9" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	c := New(server.URL, "k")

	next := func() call {
		select {
		case got := <-calls:
			return got
		case <-time.After(2 * time.Second):
			t.Fatal("no call made")
			return call{}
		}
	}

	c.Save("task", 7, Location{Lat: 0.35, Lng: 32.585})
	got := next()
	assert.Equal(t, call{"PUT", "/internal/locations/task/7", "k", `{"lat":0.35,"lng":32.585}`}, got)

	c.Delete("task", 7)
	got = next()
	assert.Equal(t, "DELETE", got.method)
	assert.Equal(t, "/internal/locations/task/7", got.path)

	c.Delete("task", 9) // a failure is only logged
	next()

	assert.Error(t, (&Client{baseURL: "http://127.0.0.1:1", client: &http.Client{Timeout: 50 * time.Millisecond}}).do("DELETE", "/x", nil))
}
func TestBucketLabel(t *testing.T) {
	assert.Equal(t, "<1 km", BucketLabel(0))
	assert.Equal(t, "10+ km", BucketLabel(4))
	assert.Equal(t, "", BucketLabel(5))
	assert.Equal(t, "", BucketLabel(-1))
}

func TestDistances(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, "/internal/distances/task", r.URL.Path)
		assert.Equal(t, "k", r.Header.Get("X-Internal-API-Key"))
		var in struct {
			Lat, Lng float64
			IDs      []uint
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		assert.Equal(t, 0.35, in.Lat)
		// every id but those divisible by 3 has a location, bucket = id % 5
		results := []map[string]interface{}{}
		for _, id := range in.IDs {
			if id%3 != 0 {
				results = append(results, map[string]interface{}{"id": id, "bucket": id % 5})
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"results": results})
	}))
	defer server.Close()
	c := New(server.URL, "k")

	ids := make([]uint, 0, 2500)
	for i := uint(1); i <= 2500; i++ {
		ids = append(ids, i)
	}
	got, err := c.Distances(context.Background(), "task", Location{Lat: 0.35, Lng: 32.585}, ids)

	require.NoError(t, err)
	assert.Equal(t, 3, calls, "sent in batches of 1000")
	assert.Equal(t, 1, got[1])
	assert.Equal(t, 2, got[7])
	_, has := got[3]
	assert.False(t, has, "ids without a location are left out")

	empty, err := c.Distances(context.Background(), "task", Location{}, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
	assert.Equal(t, 3, calls, "no call for no ids")
}

// A distance lookup carries the caller's trace, so it shows in that trace.
func TestDistancesSendTraceContext(t *testing.T) {
	prev := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer otel.SetTextMapPropagator(prev)

	traceparent := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceparent <- r.Header.Get("traceparent")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"results": []interface{}{}})
	}))
	defer server.Close()

	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled,
	}))
	_, err := New(server.URL, "k").Distances(ctx, "task", Location{}, []uint{1})

	require.NoError(t, err)
	assert.Contains(t, <-traceparent, traceID.String())
}

func TestDistancesErrors(t *testing.T) {
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer failing.Close()
	_, err := New(failing.URL, "k").Distances(context.Background(), "task", Location{}, []uint{1})
	assert.ErrorContains(t, err, "503")

	garbled := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer garbled.Close()
	_, err = New(garbled.URL, "k").Distances(context.Background(), "task", Location{}, []uint{1})
	assert.Error(t, err)

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
	}))
	defer slow.Close()
	start := time.Now()
	_, err = New(slow.URL, "k").Distances(context.Background(), "task", Location{}, []uint{1})
	assert.Error(t, err)
	assert.Less(t, time.Since(start), 450*time.Millisecond, "gives up after 300 ms")
}

func TestDistancesFrom(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/distances/item", r.URL.Path)
		var in map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&in)
		assert.Equal(t, map[string]interface{}{"kind": "request", "id": float64(31)}, in["from"])
		assert.Nil(t, in["lat"])
		_, _ = w.Write([]byte(`{"results":[{"id":51,"bucket":1}]}`))
	}))
	defer server.Close()

	got, err := New(server.URL, "k").DistancesFrom(context.Background(), "item", "request", 31, []uint{51, 52})
	require.NoError(t, err)
	assert.Equal(t, map[uint]int{51: 1}, got)
}
