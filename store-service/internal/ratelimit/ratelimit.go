// Package ratelimit caps how often one account may do an action (post a
// listing, book, ...), counted in memory: this service runs as one
// instance, and a restart forgetting the counts is fine for limits.  A copy
// of the backend's internal/ratelimit; chat has the same in JavaScript.
package ratelimit

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Code is the error code a refused request gets, in every service.
const Code = "rate_limited"

// Rule allows Limit actions per Per.
type Rule struct {
	Limit int
	Per   time.Duration
}

type key struct {
	user   uint
	action string
}

// Limiter counts each account's recent actions against its rules.
type Limiter struct {
	rules map[string][]Rule
	now   func() time.Time

	mu      sync.Mutex
	hits    map[key][]time.Time
	refused map[key]bool // logged once per run of refusals
	calls   int
}

// New limits each action by its rules (all must allow).
func New(rules map[string][]Rule) *Limiter {
	return &Limiter{rules: rules, now: time.Now, hits: map[key][]time.Time{}, refused: map[key]bool{}}
}

// Allow records an action for user and reports whether it's allowed; if
// not, how long until it would be.  A refused action doesn't count.
func (l *Limiter) Allow(user uint, action string) (bool, time.Duration) {
	rules := l.rules[action]
	if len(rules) == 0 {
		return true, 0
	}
	now := l.now()
	k := key{user, action}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweep(now)
	hits := recent(l.hits[k], now, longest(rules))
	var wait time.Duration
	for _, r := range rules {
		within := recent(hits, now, r.Per)
		if len(within) >= r.Limit {
			// Free again when the oldest hit that counts leaves the window
			if w := within[len(within)-r.Limit].Add(r.Per).Sub(now); w > wait {
				wait = w
			}
		}
	}
	if wait > 0 {
		l.hits[k] = hits
		return false, wait
	}
	l.hits[k] = append(hits, now)
	return true, 0
}

// Middleware refuses a signed-in account's request over its limit for
// action with 429 (code rate_limited, retry_after seconds), and logs the
// first refusal of a run as `limit event=hit` (Loki, for the daily summary
// and /caught).  It goes after the session check, which sets the account.
func (l *Limiter) Middleware(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.GetUint("userID")
		ok, wait := l.Allow(user, action)
		if ok {
			l.setRefused(user, action, false)
			c.Next()
			return
		}
		if l.setRefused(user, action, true) {
			log.Printf("limit event=hit action=%s user=%d username=%q email=%q", action, user,
				c.GetString("username"), firstNonEmpty(c.GetString("email"), c.GetString("userEmail")))
		}
		seconds := int(math.Ceil(wait.Seconds()))
		c.Header("Retry-After", fmt.Sprint(seconds))
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error": "You're doing that a lot. Try again in " + inWords(wait) + ".",
			"code":  Code, "retry_after": seconds})
	}
}

// setRefused records whether user's last try at action was refused, and
// reports whether this starts a run of refusals.
func (l *Limiter) setRefused(user uint, action string, refused bool) bool {
	k := key{user, action}
	l.mu.Lock()
	defer l.mu.Unlock()
	was := l.refused[k]
	if refused {
		l.refused[k] = true
	} else {
		delete(l.refused, k)
	}
	return refused && !was
}

// sweep drops idle accounts now and then, so memory follows active ones.
func (l *Limiter) sweep(now time.Time) {
	l.calls++
	if l.calls%1000 != 0 {
		return
	}
	for k, hits := range l.hits {
		if len(recent(hits, now, longest(l.rules[k.action]))) == 0 {
			delete(l.hits, k)
			delete(l.refused, k)
		}
	}
}

// recent returns the hits within per of now (hits are oldest first).
func recent(hits []time.Time, now time.Time, per time.Duration) []time.Time {
	for i, t := range hits {
		if now.Sub(t) < per {
			return hits[i:]
		}
	}
	return nil
}

func longest(rules []Rule) time.Duration {
	var d time.Duration
	for _, r := range rules {
		if r.Per > d {
			d = r.Per
		}
	}
	return d
}

func inWords(d time.Duration) string {
	if d < time.Minute {
		return "a minute"
	}
	m := int(math.Ceil(d.Minutes()))
	if m == 1 {
		return "1 minute"
	}
	return fmt.Sprintf("%d minutes", m)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
