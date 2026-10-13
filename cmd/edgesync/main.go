package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"edgesync/internal/raftkv"
	"edgesync/internal/web"
	"edgesync/pkg/kv"
	"edgesync/pkg/lsm"
)

func main() {
	addr := flag.String("http", "", "HTTP listen address (for example :8080)")
	dataDir := flag.String("data", "./data/solo", "directory for durable data")
	seed := flag.Bool("seed", false, "start this node as a Raft cluster seed")
	id := flag.String("id", "node-1", "node ID")
	peers := flag.String("peers", "", "comma-separated Raft peer addresses")
	flag.Parse()

	var store kv.Store
	if *seed || *peers != "" || *addr != "" {
		if err := os.MkdirAll(*dataDir, 0o755); err != nil {
			log.Fatalf("create data directory: %v", err)
		}
		engine, err := lsm.Open(filepath.Clean(*dataDir))
		if err != nil {
			log.Fatalf("open LSM engine: %v", err)
		}
		defer engine.Close()
		cluster, err := raftkv.NewCluster(*id, *addr, *peers, engine)
		if err != nil {
			log.Fatalf("create cluster: %v", err)
		}
		store = cluster
	} else {
		store = kv.NewMemoryStore()
	}

	if *addr != "" {
		srv := &http.Server{
			Addr:              *addr,
			Handler:           web.NewServer(store),
			ReadHeaderTimeout: 5 * time.Second,
		}
		done := make(chan os.Signal, 1)
		signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-done
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(ctx)
		}()
		log.Printf("EdgeSync node %s listening on %s", *id, *addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server: %v", err)
		}
		return
	}

	if err := runREPL(store); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
