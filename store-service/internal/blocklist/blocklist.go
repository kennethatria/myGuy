// Package blocklist keeps the accounts the backend has blocked (by email),
// read from its GET /internal/v1/blocked-users every minute.  The backend
// owns blocks; this service only refuses those accounts and hides their
// listings and requests.
package blocklist

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type List struct {
	url    string
	key    string
	client *http.Client

	mu  sync.RWMutex
	ids map[uint]bool
}

// New reads from backendURL (e.g. http://api:8080) with INTERNAL_API_KEY.
// Without either it returns nil: nobody is blocked here (and a warning is
// logged), as before blocks existed.
func New(backendURL, key string) *List {
	if backendURL == "" || key == "" {
		log.Println("WARNING: BACKEND_INTERNAL_URL or INTERNAL_API_KEY not set; blocked accounts won't be refused")
		return nil
	}
	return &List{url: backendURL + "/internal/v1/blocked-users", key: key,
		client: &http.Client{Timeout: 10 * time.Second}, ids: map[uint]bool{}}
}

// Blocked reports whether an account is blocked.
func (l *List) Blocked(id uint) bool {
	if l == nil {
		return false
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.ids[id]
}

// IDs returns the blocked accounts (to leave their posts off the boards).
func (l *List) IDs() []uint {
	if l == nil {
		return nil
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	ids := make([]uint, 0, len(l.ids))
	for id := range l.ids {
		ids = append(ids, id)
	}
	return ids
}

// Refresh reads the list once.
func (l *List) Refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Internal-API-Key", l.key)
	resp, err := l.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("blocked users: status %d", resp.StatusCode)
	}
	var body struct {
		UserIDs []uint `json:"user_ids"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return err
	}
	ids := make(map[uint]bool, len(body.UserIDs))
	for _, id := range body.UserIDs {
		ids[id] = true
	}
	l.mu.Lock()
	l.ids = ids
	l.mu.Unlock()
	return nil
}

// RefreshEvery keeps the list current until ctx ends.  When the backend
// can't be reached the last list stays in force, and the failure is logged.
func (l *List) RefreshEvery(ctx context.Context, interval time.Duration) {
	if l == nil {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := l.Refresh(ctx); err != nil {
			log.Printf("blocked accounts: refresh failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
