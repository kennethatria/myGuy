# Proximity Service

Internal-only service that keeps a rough location (a cell about 555 m across) for gigs, marketplace listings and requests, and tells the backend and store-service how far each one is from a viewer, as a coarse bucket. It never stores or returns exact coordinates or distances. Design: *Proximity Service — Design (MVP / POC, Rev 4)*.

## API

Every `/internal` call needs `X-Internal-API-Key` (the shared `INTERNAL_API_KEY`). `kind` is `task`, `item` or `request`.

| Method and path | Body | Returns |
| --- | --- | --- |
| `PUT /internal/locations/:kind/:id` | `{"lat": 0.3476, "lng": 32.5842}` | `204`; stores the snapped cell, replacing any earlier one |
| `DELETE /internal/locations/:kind/:id` | none | `204` |
| `POST /internal/distances/:kind` | `{"lat": …, "lng": …, "ids": [41, 57]}` or `{"from": {"kind": "request", "id": 31}, "ids": […]}` | `{"results": [{"id": 41, "bucket": 0}]}`; ids without a location are left out |
| `GET /health` | none | `200`, or `503` when Redis is down |

Buckets: `0` = `<1 km`, `1` = `~2 km`, `2` = `~5 km`, `3` = `~10 km`, `4` = `10+ km`. A broken rule returns `400`; Redis trouble returns `503`, which callers treat as "sort newest first".

## Rules

- **Cells:** `cell = floor(degrees / 0.005 + 0.5)`, checked against `shared/geo-cell-cases.json` like every other component that rounds a position.
- **Storage:** Redis GEO sets `geo:<kind>`, and `saved:<kind>` sorted sets of save times. A daily job removes locations saved more than 30 days ago.
- **Limits:** at most 1,000 ids per distances call.

## Development

```bash
cp .env.example .env
go run cmd/api/main.go   # needs a Redis at REDIS_ADDR
go test ./internal/...   # tests use an in-memory Redis (miniredis); CI requires ≥70% coverage
```
