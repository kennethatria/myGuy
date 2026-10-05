package geo

import (
	"encoding/json"
	"math"
	"os"
	"testing"

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

func TestSnap(t *testing.T) {
	p, err := Snap(0.3476, 32.5842)
	require.NoError(t, err)
	assert.InDelta(t, 0.350, p.Lat, 1e-9)
	assert.InDelta(t, 32.585, p.Lng, 1e-9)

	for _, bad := range [][2]float64{{91, 0}, {-91, 0}, {0, 181}, {0, -181}, {math.NaN(), 0}} {
		_, err := Snap(bad[0], bad[1])
		assert.ErrorIs(t, err, ErrOutOfRange, "%v", bad)
	}
}

func TestDistanceKm(t *testing.T) {
	kampala := Point{Lat: 0.315, Lng: 32.58}
	entebbe := Point{Lat: 0.05, Lng: 32.46}
	assert.InDelta(t, 32.4, DistanceKm(kampala, entebbe), 0.5)
	assert.Zero(t, DistanceKm(kampala, kampala))
	// one cell of latitude is about 555 m
	assert.InDelta(t, 0.556, DistanceKm(Point{0, 32}, Point{Step, 32}), 0.001)
}

func TestBucket(t *testing.T) {
	cases := map[float64]int{0: 0, 0.99: 0, 1: 1, 2.9: 1, 3: 2, 6.9: 2, 7: 3, 10: 3, 10.1: 4, 300: 4}
	for km, want := range cases {
		assert.Equal(t, want, Bucket(km), "%v km", km)
	}
	assert.Equal(t, "<1 km", Buckets[Bucket(0.2)])
	assert.Equal(t, "10+ km", Buckets[Bucket(50)])
}
