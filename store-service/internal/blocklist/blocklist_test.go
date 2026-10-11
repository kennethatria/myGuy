package blocklist

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
)

func TestList(t *testing.T) {
	ids := `{"user_ids":[3,7]}`
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/v1/blocked-users" || r.Header.Get("X-Internal-API-Key") != "key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte(ids))
	}))
	defer backend.Close()

	l := New(backend.URL, "key")
	if err := l.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !l.Blocked(3) || !l.Blocked(7) || l.Blocked(4) {
		t.Fatalf("blocked: 3 %v, 7 %v, 4 %v", l.Blocked(3), l.Blocked(7), l.Blocked(4))
	}
	got := l.IDs()
	sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
	if len(got) != 2 || got[0] != 3 || got[1] != 7 {
		t.Fatalf("IDs = %v", got)
	}

	// Unblocked in the backend: gone here on the next refresh
	ids = `{"user_ids":[]}`
	if err := l.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if l.Blocked(3) {
		t.Fatal("3 should no longer be blocked")
	}
}

func TestFailedRefreshKeepsTheList(t *testing.T) {
	up := true
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !up {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.Write([]byte(`{"user_ids":[5]}`))
	}))
	defer backend.Close()

	l := New(backend.URL, "key")
	if err := l.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	up = false
	if err := l.Refresh(context.Background()); err == nil {
		t.Fatal("expected an error")
	}
	if !l.Blocked(5) {
		t.Fatal("a failed refresh must keep blocks in force")
	}
}

func TestNotConfigured(t *testing.T) {
	l := New("", "key")
	if l != nil || l.Blocked(1) || l.IDs() != nil {
		t.Fatal("an unconfigured list blocks nobody")
	}
	l.RefreshEvery(context.Background(), 0) // returns at once
}
