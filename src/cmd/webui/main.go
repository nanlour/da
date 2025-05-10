package main

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/nanlour/da/src/web"
)

func main() {
	// Parse command line flags
	rpcAddress := flag.String("rpc", "", "RPC server address")
	baseDir := flag.String("basedir", "", "HTML template path")
	webPort := flag.Int("port", 8080, "Web UI server port")
	flag.Parse()

	templatesPath := filepath.Join(*baseDir, "templates")
	staticPath := filepath.Join(*baseDir, "static")

	// Create and start the web server
	server, err := web.NewWebServer(*rpcAddress, *webPort, templatesPath, staticPath)
	if err != nil {
		log.Fatalf("Failed to create web server: %v", err)
	}

	log.Printf("Starting web UI on http://0.0.0.0:%d", *webPort)
	log.Printf("Connecting to RPC server at %s", *rpcAddress)

	// Start the server
	if err := server.Start(); err != nil {
		log.Fatalf("Web server error: %v", err)
	}
}
