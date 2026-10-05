// Package repositories keeps rough locations in Redis: one geo set per kind
// (geo:<kind>) and a sorted set of save times (saved:<kind>) for cleanup.
package repositories

import (
	"context"
	"strconv"
	"time"

	"proximity-service/internal/geo"

	"github.com/redis/go-redis/v9"
)

// LocationRepository stores one cell per post.
type LocationRepository interface {
	Save(ctx context.Context, kind string, id uint64, at geo.Point, savedAt time.Time) error
	Delete(ctx context.Context, kind string, id uint64) error
	// Positions returns the stored cell of each id that has one.
	Positions(ctx context.Context, kind string, ids []uint64) (map[uint64]geo.Point, error)
	// RemoveSavedBefore drops every location of kind saved before cutoff.
	RemoveSavedBefore(ctx context.Context, kind string, cutoff time.Time) (int64, error)
	Ping(ctx context.Context) error
}

type redisLocationRepository struct {
	rdb *redis.Client
}

func NewRedisLocationRepository(rdb *redis.Client) LocationRepository {
	return &redisLocationRepository{rdb: rdb}
}

func geoKey(kind string) string   { return "geo:" + kind }
func savedKey(kind string) string { return "saved:" + kind }

func member(id uint64) string { return strconv.FormatUint(id, 10) }

func (r *redisLocationRepository) Save(ctx context.Context, kind string, id uint64, at geo.Point, savedAt time.Time) error {
	_, err := r.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.GeoAdd(ctx, geoKey(kind), &redis.GeoLocation{Name: member(id), Longitude: at.Lng, Latitude: at.Lat})
		p.ZAdd(ctx, savedKey(kind), redis.Z{Score: float64(savedAt.Unix()), Member: member(id)})
		return nil
	})
	return err
}

func (r *redisLocationRepository) Delete(ctx context.Context, kind string, id uint64) error {
	_, err := r.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.ZRem(ctx, geoKey(kind), member(id))
		p.ZRem(ctx, savedKey(kind), member(id))
		return nil
	})
	return err
}

func (r *redisLocationRepository) Positions(ctx context.Context, kind string, ids []uint64) (map[uint64]geo.Point, error) {
	found := make(map[uint64]geo.Point, len(ids))
	if len(ids) == 0 {
		return found, nil
	}
	members := make([]string, len(ids))
	for i, id := range ids {
		members[i] = member(id)
	}
	positions, err := r.rdb.GeoPos(ctx, geoKey(kind), members...).Result()
	if err != nil {
		return nil, err
	}
	for i, pos := range positions {
		if pos == nil {
			continue
		}
		// Redis stores points to about 0.6 m; snap back to the exact cell
		found[ids[i]] = geo.Point{Lat: geo.Degrees(geo.Cell(pos.Latitude)), Lng: geo.Degrees(geo.Cell(pos.Longitude))}
	}
	return found, nil
}

func (r *redisLocationRepository) RemoveSavedBefore(ctx context.Context, kind string, cutoff time.Time) (int64, error) {
	old, err := r.rdb.ZRangeByScore(ctx, savedKey(kind), &redis.ZRangeBy{
		Min: "-inf",
		Max: "(" + strconv.FormatInt(cutoff.Unix(), 10),
	}).Result()
	if err != nil || len(old) == 0 {
		return 0, err
	}
	members := make([]interface{}, len(old))
	for i, m := range old {
		members[i] = m
	}
	_, err = r.rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.ZRem(ctx, geoKey(kind), members...)
		p.ZRem(ctx, savedKey(kind), members...)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return int64(len(old)), nil
}

func (r *redisLocationRepository) Ping(ctx context.Context) error {
	return r.rdb.Ping(ctx).Err()
}
