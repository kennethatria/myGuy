package proximity

import (
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
