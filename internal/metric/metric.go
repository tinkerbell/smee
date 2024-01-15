package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	DHCPTotal *prometheus.CounterVec

	DiscoverDuration    prometheus.ObserverVec
	HardwareDiscovers   *prometheus.CounterVec
	DiscoversInProgress *prometheus.GaugeVec

	JobDuration    prometheus.ObserverVec
	JobsTotal      *prometheus.CounterVec
	JobsInProgress *prometheus.GaugeVec
)

func Init() {
	DHCPTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dhcp_total",
		Help: "Number of DHCP Requests handled.",
	}, []string{"op", "type", "giaddr"})

	labelValues := []prometheus.Labels{
		{"op": "recv", "type": "DHCPACK", "giaddr": "0.0.0.0"},
		{"op": "recv", "type": "DHCPDECLINE", "giaddr": "0.0.0.0"},
		{"op": "recv", "type": "DHCPDISCOVER", "giaddr": "0.0.0.0"},
		{"op": "recv", "type": "DHCPINFORM", "giaddr": "0.0.0.0"},
		{"op": "recv", "type": "DHCPNAK", "giaddr": "0.0.0.0"},
		{"op": "recv", "type": "DHCPOFFER", "giaddr": "0.0.0.0"},
		{"op": "recv", "type": "DHCPRELEASE", "giaddr": "0.0.0.0"},
		{"op": "recv", "type": "DHCPREQUEST", "giaddr": "0.0.0.0"},
		{"op": "send", "type": "DHCPOFFER", "giaddr": "0.0.0.0"},
	}
	initCounterLabels(DHCPTotal, labelValues)

	labelValues = []prometheus.Labels{
		{"from": "dhcp"},
		{"from": "ip"},
	}

	DiscoverDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "discover_duration_seconds",
		Help:    "Duration taken to get a response for a newly discovered request.",
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
