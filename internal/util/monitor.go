package util

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func PrometheusExporter() {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
