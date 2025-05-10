package main

import (
	"flag"
	"log"
	"path/filepath"
	"runtime"

	"github.com/nanlour/da/src/web"
)

func main() {
	// Parse command line flags
	rpcAddress := flag.String("rpc", "localhost:9001", "RPC server address")
	webPort := flag.Int("port", 8080, "Web UI server port")
	flag.Parse()

	// Get the base directory for templates and static files
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Failed to get current file path")
	}
	baseDir := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filename))))
	templatesPath := filepath.Join(baseDir, "src", "web", "templates")
	staticPath := filepath.Join(baseDir, "src", "web", "static")

	// Create and start the web server
	server, err := web.NewWebServer(*rpcAddress, *webPort, templatesPath, staticPath)
	if err != nil {
		log.Fatalf("Failed to create web server: %v", err)
	}

	log.Printf("Starting web UI on http://localhost:%d", *webPort)
	log.Printf("Connecting to RPC server at %s", *rpcAddress)

	// Start the server
	if err := server.Start(); err != nil {
		log.Fatalf("Web server error: %v", err)
	}
}
