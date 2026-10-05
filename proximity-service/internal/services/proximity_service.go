// Package services applies the proximity rules: three kinds of post, rough
// cells only, distances shared only as buckets.
package services

import (
	"context"
	"errors"
	"time"

	"proximity-service/internal/geo"
	"proximity-service/internal/repositories"
)

const (
	// Retention is how long a location is kept after it was last saved.
	Retention = 30 * 24 * time.Hour
	// MaxIDs caps one distances request; callers send more in batches.
	MaxIDs = 1000
)

// Kinds of post that carry a location.
var Kinds = []string{"task", "item", "request"}

var (
	ErrUnknownKind = errors.New("kind must be task, item or request")
	ErrInvalidID   = errors.New("id must be a positive integer")
	ErrTooManyIDs  = errors.New("at most 1000 ids per request")
	ErrNoOrigin    = errors.New("give either lat and lng, or from")
)

// Ref names one stored post, for distances measured from it.
type Ref struct {
	Kind string `json:"kind"`
	ID   uint64 `json:"id"`
}

// DistancesQuery measures from the viewer (Lat, Lng) or from a stored post
// (From) to each of IDs, all of kind Kind.
type DistancesQuery struct {
	Kind string
	Lat  *float64
	Lng  *float64
	From *Ref
	IDs  []uint64
}

// Result is one post's distance bucket (an index into geo.Buckets).
type Result struct {
	ID     uint64 `json:"id"`
	Bucket int    `json:"bucket"`
}

type ProximityService struct {
	repo repositories.LocationRepository
	now  func() time.Time
}

func NewProximityService(repo repositories.LocationRepository) *ProximityService {
	return &ProximityService{repo: repo, now: time.Now}
}

func validKind(kind string) bool {
	for _, k := range Kinds {
		if k == kind {
			return true
		}
	}
	return false
}

// SaveLocation stores a post's cell, replacing any earlier one.
func (s *ProximityService) SaveLocation(ctx context.Context, kind string, id uint64, lat, lng float64) error {
	if !validKind(kind) {
		return ErrUnknownKind
	}
	if id == 0 {
		return ErrInvalidID
	}
	at, err := geo.Snap(lat, lng)
	if err != nil {
		return err
	}
	return s.repo.Save(ctx, kind, id, at, s.now())
}

func (s *ProximityService) DeleteLocation(ctx context.Context, kind string, id uint64) error {
	if !validKind(kind) {
		return ErrUnknownKind
	}
	if id == 0 {
		return ErrInvalidID
	}
	return s.repo.Delete(ctx, kind, id)
}

// Distances returns a bucket for each id that has a stored location, in
// the order asked. Ids without one, and every id when the origin has no
// location, are left out.
func (s *ProximityService) Distances(ctx context.Context, q DistancesQuery) ([]Result, error) {
	if !validKind(q.Kind) {
		return nil, ErrUnknownKind
	}
	if len(q.IDs) > MaxIDs {
		return nil, ErrTooManyIDs
	}

	origin, ok, err := s.origin(ctx, q)
	if err != nil || !ok {
		return []Result{}, err
	}

	positions, err := s.repo.Positions(ctx, q.Kind, q.IDs)
	if err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(positions))
	for _, id := range q.IDs {
		if at, found := positions[id]; found {
			results = append(results, Result{ID: id, Bucket: geo.Bucket(geo.DistanceKm(origin, at))})
		}
	}
	return results, nil
}

// origin is the viewer's cell, or the stored cell of q.From; ok is false
// when q.From has no location.
func (s *ProximityService) origin(ctx context.Context, q DistancesQuery) (geo.Point, bool, error) {
	switch {
	case q.From != nil && q.Lat == nil && q.Lng == nil:
		if !validKind(q.From.Kind) {
			return geo.Point{}, false, ErrUnknownKind
		}
		found, err := s.repo.Positions(ctx, q.From.Kind, []uint64{q.From.ID})
		if err != nil {
			return geo.Point{}, false, err
		}
		at, ok := found[q.From.ID]
		return at, ok, nil
	case q.From == nil && q.Lat != nil && q.Lng != nil:
		at, err := geo.Snap(*q.Lat, *q.Lng)
		return at, err == nil, err
	default:
		return geo.Point{}, false, ErrNoOrigin
	}
}

// Cleanup removes locations saved more than Retention ago, for every kind.
func (s *ProximityService) Cleanup(ctx context.Context) (int64, error) {
	cutoff := s.now().Add(-Retention)
	var removed int64
	for _, kind := range Kinds {
		n, err := s.repo.RemoveSavedBefore(ctx, kind, cutoff)
		if err != nil {
			return removed, err
		}
		removed += n
	}
	return removed, nil
}

// Healthy reports whether Redis answers.
func (s *ProximityService) Healthy(ctx context.Context) error {
	return s.repo.Ping(ctx)
}
