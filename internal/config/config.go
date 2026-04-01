package config

import "os"

type Config struct {
	Port              string
	CarveBaseURL      string
	OmarBaseURL       string
	GrafanaBaseURL    string
	PrometheusBaseURL string
}

func Load() Config {
	return Config{
		Port:              getenv("GATEWAY_PORT", "18080"),
		CarveBaseURL:      getenv("CARVE_BASE_URL", "http://carve:8080"),
		OmarBaseURL:       getenv("OMAR_BASE_URL", "http://omar:8080"),
		GrafanaBaseURL:    getenv("GRAFANA_BASE_URL", "http://grafana.otel-demo.svc.cluster.local:80"),
		PrometheusBaseURL: getenv("PROMETHEUS_BASE_URL", "http://prometheus.otel-demo.svc.cluster.local:9090"),
	}
}

func getenv(k, d string) string {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	return v
}
