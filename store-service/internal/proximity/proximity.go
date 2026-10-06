// Package proximity saves the rough location of listings and requests in
// the proximity service; it is a copy of the backend's. Positions are
// snapped to ~555 m cells before they leave this service; the cell rule is
// shared with the backend, the frontend and the proximity service through
// shared/geo-cell-cases.json.
package proximity

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"
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
	return &Client{baseURL: baseURL, apiKey: apiKey, client: &http.Client{Timeout: 300 * time.Millisecond}}
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
