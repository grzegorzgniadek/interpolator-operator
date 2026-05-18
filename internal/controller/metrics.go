package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	interCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "inter_count",
			Help: "Count of inter resources",
		},
	)
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(interCount)
}
