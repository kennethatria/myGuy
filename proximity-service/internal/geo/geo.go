// Package geo holds the rough-location rules: positions snap to cells about
// 555 m across, and distances are only ever shared as coarse buckets. The
// cell rule is shared with the backend, store-service and frontend through
// shared/geo-cell-cases.json; change it in all of them.
package geo

import (
	"errors"
	"math"
)

// Step is a cell's size in degrees of latitude and longitude.
const Step = 0.005

const earthRadiusKm = 6371.0

// ErrOutOfRange means coordinates that can't be on Earth.
var ErrOutOfRange = errors.New("coordinates out of range")

// Cell snaps degrees to a whole-number cell index. floor(x/Step + 0.5)
// rounds halves up the same way everywhere; Go's math.Round and JavaScript's
// Math.round disagree on negative halves, and Uganda crosses the equator.
func Cell(degrees float64) int64 {
	return int64(math.Floor(degrees/Step + 0.5))
}

// Degrees is the centre of a cell.
func Degrees(cell int64) float64 {
	return float64(cell) * Step
}

// Point is a position snapped to its cell centre.
type Point struct {
	Lat, Lng float64
}

// Snap validates a position and returns its cell centre.
func Snap(lat, lng float64) (Point, error) {
	if math.IsNaN(lat) || math.IsNaN(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return Point{}, ErrOutOfRange
	}
	return Point{Lat: Degrees(Cell(lat)), Lng: Degrees(Cell(lng))}, nil
}

// DistanceKm is the great-circle distance between two points.
func DistanceKm(a, b Point) float64 {
	toRad := math.Pi / 180
	dLat := (b.Lat - a.Lat) * toRad
	dLng := (b.Lng - a.Lng) * toRad
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(a.Lat*toRad)*math.Cos(b.Lat*toRad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(h)))
}

// Buckets are the only distances anyone sees, nearest first.
var Buckets = []string{"<1 km", "~2 km", "~5 km", "~10 km", "10+ km"}

// Bucket maps a distance to its index in Buckets.
func Bucket(km float64) int {
	switch {
	case km < 1:
		return 0
	case km < 3:
		return 1
	case km < 7:
		return 2
	case km <= 10:
		return 3
	default:
		return 4
	}
}
