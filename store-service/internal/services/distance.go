package services

import (
	"sort"

	"store-service/internal/proximity"
)

// Distancer measures rough distances to posts as buckets (an index into
// proximity.Buckets): from a position, or from another stored post. Posts
// without a stored location are left out.
type Distancer interface {
	Distances(kind string, at proximity.Location, ids []uint) (map[uint]int, error)
	DistancesFrom(kind, fromKind string, fromID uint, ids []uint) (map[uint]int, error)
}

// rankByDistance orders ids (given newest first) nearest bucket first,
// keeping newest first within a bucket; ids without a bucket go last.
func rankByDistance(ids []uint, buckets map[uint]int) {
	unknown := len(proximity.Buckets)
	rank := func(id uint) int {
		if b, ok := buckets[id]; ok {
			return b
		}
		return unknown
	}
	sort.SliceStable(ids, func(a, b int) bool { return rank(ids[a]) < rank(ids[b]) })
}

// pageOf cuts one page out of ranked ids (pages count from 1).
func pageOf(ids []uint, page, perPage int) []uint {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	start := (page - 1) * perPage
	if start > len(ids) {
		start = len(ids)
	}
	end := start + perPage
	if end > len(ids) {
		end = len(ids)
	}
	return ids[start:end]
}

// tagFor is the distance tag for id, or "" when it has no location.
func tagFor(buckets map[uint]int, id uint) string {
	if b, ok := buckets[id]; ok {
		return proximity.BucketLabel(b)
	}
	return ""
}
