package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// UpdatesTotal counts completed updates by target, service, and result.
	UpdatesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bulwark_updates_total",
		Help: "Total number of updates performed",
	}, []string{"target", "service", "result"})

	// RollbacksTotal counts rollbacks by target and service.
	RollbacksTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bulwark_rollbacks_total",
		Help: "Total number of rollbacks performed",
	}, []string{"target", "service"})

	// ProbesTotal counts probe executions by type and result.
	ProbesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bulwark_probes_total",
		Help: "Total number of probe executions",
	}, []string{"type", "result"})

	// ProbeDuration observes probe execution durations.
	ProbeDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "bulwark_probe_duration_seconds",
		Help:    "Probe execution duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"type"})

	// DigestFetchDuration observes registry digest fetch times.
	DigestFetchDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "bulwark_digest_fetch_duration_seconds",
		Help:    "Time to fetch image digest from registry",
		Buckets: prometheus.DefBuckets,
	}, []string{"registry"})

	// DiscoveryDuration observes discovery scan durations.
	DiscoveryDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bulwark_discovery_duration_seconds",
		Help:    "Time to complete target discovery",
		Buckets: prometheus.DefBuckets,
	})

	// ManagedTargets tracks the current number of managed targets.
	ManagedTargets = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bulwark_managed_targets",
		Help: "Current number of managed targets",
	})

	// ManagedServices tracks the current number of managed services.
	ManagedServices = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bulwark_managed_services",
		Help: "Current number of managed services",
	})
)
