package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"log"
	"net/http"
	"os"
)

var (
	metricsTokenCli string
	insecureMetrics bool
	metricsToken    string

	promWebhooksReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "webhook_proxy_webhooks_received_total",
		Help: "The total number of received valid webhooks payloads",
	})

	promOpenClientConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "webhook_proxy_open_client_connections",
		Help: "Momentary number of open client connections",
	})

	promTotalClientConnections = promauto.NewCounter(prometheus.CounterOpts{
		Name: "webhook_proxy_client_connections_total",
		Help: "The total number of served client connections",
	})

	promTimedOutClients = promauto.NewCounter(prometheus.CounterOpts{
		Name: "webhook_proxy_timed_out_clients_total",
		Help: "The total number of timed out webhook clients",
	})

	promTimedOutWebhooks = promauto.NewCounter(prometheus.CounterOpts{
		Name: "webhook_proxy_timed_out_webhooks_total",
		Help: "The total number of timed out webhook payloads",
	})

	promActiveTokens = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "webhook_proxy_active_tokens",
		Help: "Number of currently active stream tokens",
	})
)

func setupPrometheusAuth() {
	metricsToken = os.Getenv("PROXY_METRICS_TOKEN")

	// If set via flag, overwrite the env one
	if metricsTokenCli != "" {
		metricsToken = metricsTokenCli
		return
	}

	if metricsToken != "" {
		return
	}

	if insecureMetrics {
		log.Printf("IMPORTANT: metrics token not provided and insecure metrics endpoint allowed, bearer token will NOT be required\n")
		return
	}

	// Token empty and insecure metrics not allowed, generate random token or fail
	var err error
	log.Printf("IMPORTANT: metrics token not provided but insecure metrics endpoint is NOT allowed, generating random token\n")
	metricsToken, err = generateSecureToken(16)
	if err != nil {
		log.Panicf("generating random token for metrics endpoint failed: %v\n", err)
	}
	log.Printf("IMPORTANT: generated random metrics token is: %s\n", metricsToken)

}

func prometheusAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if metricsToken == "" {
			next.ServeHTTP(w, r)
			return
		}

		if r.Header.Get("Authorization") != "Bearer "+metricsToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
