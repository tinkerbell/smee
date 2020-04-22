package metrics

import (
	"github.com/packethost/pkg/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	DHCPTotal *prometheus.CounterVec

	CacherDuration           prometheus.ObserverVec
	CacherCacheHits          *prometheus.CounterVec
	CacherTotal              *prometheus.CounterVec
	CacherRequestsInProgress *prometheus.GaugeVec

	DiscoverDuration    prometheus.ObserverVec
	HardwareDiscovers   *prometheus.CounterVec
	DiscoversInProgress *prometheus.GaugeVec

	JobDuration    prometheus.ObserverVec
	JobsTotal      *prometheus.CounterVec
	JobsInProgress *prometheus.GaugeVec
)

func Init(_ log.Logger) {
	DHCPTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dhcp_total",
		Help: "Number of DHCP Requests received.",
	}, []string{"op", "giaddr"})

	labelValues := []prometheus.Labels{
		{"op": "DHCPACK", "giaddr": "0.0.0.0"},
		{"op": "DHCPDECLINE", "giaddr": "0.0.0.0"},
		{"op": "DHCPDISCOVER", "giaddr": "0.0.0.0"},
		{"op": "DHCPINFORM", "giaddr": "0.0.0.0"},
		{"op": "DHCPNAK", "giaddr": "0.0.0.0"},
		{"op": "DHCPOFFER", "giaddr": "0.0.0.0"},
		{"op": "DHCPRELEASE", "giaddr": "0.0.0.0"},
		{"op": "DHCPREQUEST", "giaddr": "0.0.0.0"},
		{"op": "reply", "giaddr": "0.0.0.0"},
	}
	initCounterLabels(DHCPTotal, labelValues)

	CacherDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "cacher_request_duration_seconds",
		Help:    "Duration of cacher requests",
		Buckets: prometheus.LinearBuckets(.01, .05, 10),
	}, []string{"from"})
	CacherCacheHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cacher_cache_hits",
		Help: "Number of requests which returned data from cacher.",
	}, []string{"from"})
	CacherTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cacher_total",
		Help: "Total number of requests to the cacher service.",
	}, []string{"from"})
	CacherRequestsInProgress = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cacher_requests_in_progress",
		Help: "Number of cacher requests that have yet to receive a response.",
	}, []string{"from"})

	labelValues = []prometheus.Labels{
		{"from": "dhcp"},
		{"from": "ip"},
	}
	initObserverLabels(CacherDuration, labelValues)
	initCounterLabels(CacherCacheHits, labelValues)
	initCounterLabels(CacherTotal, labelValues)
	initGaugeLabels(CacherRequestsInProgress, labelValues)

	DiscoverDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "discover_duration_seconds",
		Help:    "Duration taken to get a responce for a newly discovered request",
		Buckets: prometheus.LinearBuckets(.01, .05, 10),
	}, []string{"from"})
	HardwareDiscovers = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "discover_total",
		Help: "Number of discover requests requested.",
	}, []string{"from"})
	DiscoversInProgress = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "discover_in_progress",
		Help: "Number of discover requests that have yet to receive a response.",
	}, []string{"from"})

	initObserverLabels(DiscoverDuration, labelValues)
	initCounterLabels(HardwareDiscovers, labelValues)
	initGaugeLabels(DiscoversInProgress, labelValues)

	JobDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "jobs_duration_seconds",
		Help:    "Duration taken for a job to complete.",
		Buckets: prometheus.LinearBuckets(.01, .05, 10),
	}, []string{"from", "op"})
	JobsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "jobs_total",
		Help: "Number of jobs.",
	}, []string{"from", "op"})
	JobsInProgress = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "jobs_in_progress",
		Help: "Number of jobs waiting to complete.",
	}, []string{"from", "op"})

	labelValues = []prometheus.Labels{
		{"from": "dhcp", "op": "DHCPACK"},
		{"from": "dhcp", "op": "DHCPDECLINE"},
		{"from": "dhcp", "op": "DHCPDISCOVER"},
		{"from": "dhcp", "op": "DHCPINFORM"},
		{"from": "dhcp", "op": "DHCPNAK"},
		{"from": "dhcp", "op": "DHCPOFFER"},
		{"from": "dhcp", "op": "DHCPRELEASE"},
		{"from": "dhcp", "op": "DHCPREQUEST"},
		{"from": "http", "op": "file"},
		{"from": "http", "op": "hardware-components"},
		{"from": "http", "op": "phone-home"},
		{"from": "http", "op": "problem"},
		{"from": "http", "op": "event"},
		{"from": "tftp", "op": "read"},
	}

	initObserverLabels(JobDuration, labelValues)
	initCounterLabels(JobsTotal, labelValues)
	initGaugeLabels(JobsInProgress, labelValues)
}

func initCounterLabels(m *prometheus.CounterVec, l []prometheus.Labels) {
	for _, labels := range l {
		m.With(labels)
	}
}

func initGaugeLabels(m *prometheus.GaugeVec, l []prometheus.Labels) {
	for _, labels := range l {
		m.With(labels)
	}
}

func initObserverLabels(m prometheus.ObserverVec, l []prometheus.Labels) {
	for _, labels := range l {
		m.With(labels)
	}
}
