// Package metrics publishes account and activity counts for Prometheus,
// read from the database on every scrape.  Counts only, by status at most:
// never ids, names, emails or text.  It listens on its own port, which nginx
// does not route and the firewall opens to the monitoring server alone.
package metrics

import (
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"
)

// gauge is one metric and the query behind it.  The query returns a single
// count, or (status, count) rows when the metric has a status label.
type gauge struct {
	desc     *prometheus.Desc
	byStatus bool
	query    string
	args     func() []any
}

type collector struct {
	db     *gorm.DB
	gauges []gauge
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	for _, g := range c.gauges {
		ch <- g.desc
	}
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	for _, g := range c.gauges {
		var args []any
		if g.args != nil {
			args = g.args()
		}
		rows, err := c.db.Raw(g.query, args...).Rows()
		if err != nil {
			ch <- prometheus.NewInvalidMetric(g.desc, err)
			continue
		}
		for rows.Next() {
			var status string
			var count float64
			if g.byStatus {
				err = rows.Scan(&status, &count)
			} else {
				err = rows.Scan(&count)
			}
			if err != nil {
				ch <- prometheus.NewInvalidMetric(g.desc, err)
				break
			}
			if g.byStatus {
				ch <- prometheus.MustNewConstMetric(g.desc, prometheus.GaugeValue, count, status)
			} else {
				ch <- prometheus.MustNewConstMetric(g.desc, prometheus.GaugeValue, count)
			}
		}
		rows.Close()
	}
}

func lastWeek() []any {
	since := time.Now().Add(-7 * 24 * time.Hour)
	return []any{since, since}
}

func newCollector(db *gorm.DB) *collector {
	return &collector{db: db, gauges: []gauge{
		{
			desc:  prometheus.NewDesc("myguy_accounts", "Accounts that exist.", nil, nil),
			query: "SELECT count(*) FROM users",
		},
		{
			desc: prometheus.NewDesc("myguy_gig_active_accounts_7d",
				"Accounts that posted or applied for a gig in the last 7 days.", nil, nil),
			query: "SELECT count(DISTINCT id) FROM (SELECT created_by AS id FROM tasks WHERE created_at > ? " +
				"UNION SELECT applicant_id FROM applications WHERE created_at > ?) AS active",
			args: lastWeek,
		},
		{
			desc:     prometheus.NewDesc("myguy_gigs", "Gigs by status.", []string{"status"}, nil),
			byStatus: true,
			query:    "SELECT status, count(*) FROM tasks GROUP BY status",
		},
		{
			desc:     prometheus.NewDesc("myguy_applications", "Gig applications by status.", []string{"status"}, nil),
			byStatus: true,
			query:    "SELECT status, count(*) FROM applications GROUP BY status",
		},
	}}
}

// Handler serves the counts in the Prometheus text format.
func Handler(db *gorm.DB) http.Handler {
	registry := prometheus.NewRegistry()
	registry.MustRegister(newCollector(db))
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError})
}

// Serve listens on addr (e.g. ":9464") in the background; a failure is
// logged and leaves the API running.
func Serve(addr string, db *gorm.DB) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", Handler(db))
	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("metrics server stopped: %v", err)
		}
	}()
}
