package main

import (
	"context"
	"errors"
	"flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	store          Store
	signalCh       chan os.Signal
	addrStr        string
	requestTimeout int
)

func main() {
	// Configure logging
	log.SetFlags(log.LstdFlags)
	log.SetOutput(os.Stdout)

	// Settings
	flag.IntVar(&requestTimeout, "timeout", 120, "maximum waiting time for webhook response in seconds. Client connection gets closed after that.")
	flag.BoolVar(&insecureMetrics, "allow-insecure-metrics", false, "whether to expose /metrics endpoint without requiring token")
	flag.StringVar(&metricsTokenCli, "metrics-token", "", "bearer token required for accessing /metrics endpoint")
	flag.StringVar(&addrStr, "addr", "0.0.0.0:8000", "address and port to listen on")
	flag.Parse()
	setupPrometheusAuth()

	// Configure graceful signal handling
	// `ctx` is passed to client stream handling for graceful connection closing
	signalCh = make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	// Initialize data store
	store = NewInMemStore()

	// token.go. Stores webhook's requests_ids and tokens assigned to them.
	// Tokens are required to connect to `/listen` endpoint and listen to the webhook responses.
	streamsTokens = sync.Map{}

	// Start http server
	server := startServer(ctx)
	go cleanup()

	// Wait for interrupt signal
	<-signalCh
	log.Printf("closing clients connections and shuting down (10s)...")

	// Close clients connections
	cancel()

	// Shutdown server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("error shutting down http server: %v\n", err)
	}
	log.Println("shutting down")
}

// cleanup removes webhook responses older than requestTimeout seconds
// those can only happen when they were delivered after the requestTimeout was exceeded
// in client stream connection, ie.
// client connects --> 120s passes --> client timeouts --> webhook delivered --> 120s passes --> delete webhook payload
func cleanup() {
	t := time.NewTicker(time.Duration(requestTimeout) * time.Second)
	for {
		<-t.C
		// Clean webhook payloads store
		log.Printf("cleaning up the store from webhook payloads older than %d seconds...", requestTimeout)
		reqs := store.GetOlderThan(time.Duration(requestTimeout) * time.Second)
		log.Printf("%d requests older than %d seconds, deleting", len(reqs), requestTimeout)
		promTimedOutWebhooks.Add(float64(len(reqs)))
		for _, req := range reqs {
			store.Delete(req)
		}

		// Clean listener tokens
		n := 0
		streamsTokens.Range(func(key, value interface{}) bool {
			token := value.(streamToken)
			if token.expiresAt < time.Now().Unix() {
				streamsTokens.Delete(key)
				n++
			}
			return true
		})
		log.Printf("%d expired streams tokens, deleting", n)
		promActiveTokens.Sub(float64(n))
	}
}

func startServer(ctx context.Context) *http.Server {
	// Configure http server
	addr, err := net.ResolveTCPAddr("tcp", addrStr)
	if err != nil {
		log.Fatalf("error resolving address: %v\n", err)
		return nil
	}
	server := &http.Server{
		Addr: addr.String(),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /webhook", handleIncomingWebhook)
	mux.HandleFunc("POST /token", handleCreateToken)
	mux.HandleFunc("GET /listen/{request_id}", handleClientStream(ctx))
	mux.Handle("/metrics", prometheusAuthMiddleware(promhttp.Handler()))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	})
	server.Handler = mux

	// Start server
	log.Println("starting server")
	go func() {
		log.Printf("listening on %s\n", addr.String())
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server error: %v\n", err)
		}
		log.Printf("stopped accepting new connections\n")
	}()
	return server
}
