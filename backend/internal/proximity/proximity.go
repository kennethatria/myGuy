// Package proximity saves the rough location of gigs in the proximity
// service. Positions are snapped to ~555 m cells before they leave this
// service; the cell rule is shared with store-service, the frontend and the
// proximity service through shared/geo-cell-cases.json.
package proximity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Step is a cell's size in degrees.
const Step = 0.005

// ErrInvalidLocation means a position that is half given or not on Earth.
var ErrInvalidLocation = errors.New("location must have a valid lat and lng")

// Cell snaps degrees to a whole-number cell index; floor(x/Step + 0.5)
// treats negative halves the same in every language.
func Cell(degrees float64) int64 {
	return int64(math.Floor(degrees/Step + 0.5))
}

// Location is a position snapped to its cell centre.
type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// Parse turns an optional lat/lng pair from a request into a rough
// location: nil when neither is given, an error when only one is or either
// is out of range.
func Parse(lat, lng *float64) (*Location, error) {
	if lat == nil && lng == nil {
		return nil, nil
	}
	if lat == nil || lng == nil || math.IsNaN(*lat) || math.IsNaN(*lng) ||
		*lat < -90 || *lat > 90 || *lng < -180 || *lng > 180 {
		return nil, ErrInvalidLocation
	}
	return &Location{Lat: float64(Cell(*lat)) * Step, Lng: float64(Cell(*lng)) * Step}, nil
}

// Client calls the proximity service's internal API in the background. It
// is best effort: a slow or failing proximity service never holds up or
// fails the post that triggered the call.
type Client struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// New returns a Client for the proximity service at baseURL (e.g.
// http://proximity-service:8083), authenticated with INTERNAL_API_KEY.
func New(baseURL, apiKey string) *Client {
	return &Client{baseURL: baseURL, apiKey: apiKey, client: &http.Client{
		Timeout: 300 * time.Millisecond,
		// Sends the caller's trace context (traceparent), so a distance lookup
		// shows in the same trace as the request that needed it.
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}}
}

// Save stores the rough location of a post of kind (task, item, request).
func (c *Client) Save(kind string, id uint, at Location) {
	c.background(fmt.Sprintf("save %s %d", kind, id), func() error {
		body, err := json.Marshal(at)
		if err != nil {
			return err
		}
		return c.do(http.MethodPut, fmt.Sprintf("/internal/locations/%s/%d", kind, id), body)
	})
}

// Delete removes a post's location.
func (c *Client) Delete(kind string, id uint) {
	c.background(fmt.Sprintf("delete %s %d", kind, id), func() error {
		return c.do(http.MethodDelete, fmt.Sprintf("/internal/locations/%s/%d", kind, id), nil)
	})
}

func (c *Client) background(what string, call func() error) {
	go func() {
		if err := call(); err != nil {
			log.Printf("WARNING: proximity %s failed: %v", what, err)
		}
	}()
}

func (c *Client) do(method, path string, body []byte) error {
	req, err := http.NewRequest(method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-API-Key", c.apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("proximity service returned %d", resp.StatusCode)
	}
	return nil
}

// Buckets are the only distances anyone sees, nearest first; the proximity
// service returns an index into this list.
var Buckets = []string{"<1 km", "~2 km", "~5 km", "~10 km", "10+ km"}

// BucketLabel is the tag shown for a bucket index ("" if out of range).
func BucketLabel(bucket int) string {
	if bucket < 0 || bucket >= len(Buckets) {
		return ""
	}
	return Buckets[bucket]
}

// maxIDsPerCall is the proximity service's limit for one distances call.
const maxIDsPerCall = 1000

// Distances returns the distance bucket of each id that has a stored
// location, measured from at. It waits for the answer (300 ms at most per
// call); on any error callers fall back to their usual order.
func (c *Client) Distances(ctx context.Context, kind string, at Location, ids []uint) (map[uint]int, error) {
	buckets := make(map[uint]int, len(ids))
	for start := 0; start < len(ids); start += maxIDsPerCall {
		end := start + maxIDsPerCall
		if end > len(ids) {
			end = len(ids)
		}
		body, err := json.Marshal(map[string]interface{}{"lat": at.Lat, "lng": at.Lng, "ids": ids[start:end]})
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/internal/distances/%s", c.baseURL, kind), bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Internal-API-Key", c.apiKey)
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		var out struct {
			Results []struct {
				ID     uint `json:"id"`
				Bucket int  `json:"bucket"`
			} `json:"results"`
		}
		err = json.NewDecoder(resp.Body).Decode(&out)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("proximity service returned %d", resp.StatusCode)
		}
		if err != nil {
			return nil, err
		}
		for _, r := range out.Results {
			buckets[r.ID] = r.Bucket
		}
	}
	return buckets, nil
}
