package util

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusExporter exports metrics to Prometheus, it will starts a web server listening on the port 2112
func PrometheusExporter() {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
