package web

import (
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
)

// WebServer represents the web interface for blockchain
type WebServer struct {
	client     *RPCClient
	port       int
	templates  *template.Template
	staticPath string
}

// NewWebServer creates a new web server instance
func NewWebServer(rpcAddress string, webPort int, templatesPath, staticPath string) (*WebServer, error) {
	client, err := NewRPCClient(rpcAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC server: %v", err)
	}

	// Parse all templates
	templates, err := template.ParseGlob(filepath.Join(templatesPath, "*.html"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %v", err)
	}

	return &WebServer{
		client:     client,
		port:       webPort,
		templates:  templates,
		staticPath: staticPath,
	}, nil
}

// Start begins listening for HTTP requests
func (s *WebServer) Start() error {
	// Set up routes
	http.HandleFunc("/", s.handleHome)
	http.HandleFunc("/send", s.handleSend)
	http.HandleFunc("/balance", s.handleBalance)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(s.staticPath))))
	http.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Server is running. Templates: %v", s.templates.DefinedTemplates())
	})

	// Start server
	addr := fmt.Sprintf("0.0.0.0:%d", s.port)
	log.Printf("Web UI server starting on http://%s", addr)
	return http.ListenAndServe(addr, nil)
}

// handleHome displays the home page with recent blocks and node info
func (s *WebServer) handleHome(w http.ResponseWriter, r *http.Request) {
	// Get the last 10 blocks
	blocks, err := s.client.GetLastTenBlocks()
	if err != nil {
		http.Error(w, "Failed to get blocks: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get node's address
	address, err := s.client.GetAddress()
	if err != nil {
		http.Error(w, "Failed to get node address: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Format blocks for display
	type DisplayBlock struct {
		Hash   string
		Height uint64
		From   string
		To     string
		Amount float64
	}

	displayBlocks := make([]DisplayBlock, len(blocks))
	for i, block := range blocks {
		hash := block.Hash()
		displayBlocks[i] = DisplayBlock{
			Hash:   hex.EncodeToString(hash[:]),
			Height: block.Height,
			From:   hex.EncodeToString(block.Txn.FromAddress[:]),
			To:     hex.EncodeToString(block.Txn.ToAddress[:]),
			Amount: block.Txn.Amount,
		}
	}

	data := struct {
		Blocks  []DisplayBlock
		Address string
	}{
		Blocks:  displayBlocks,
		Address: hex.EncodeToString(address[:]),
	}

	s.renderTemplate(w, "index_content", data)
}

// handleSend handles transaction sending requests
func (s *WebServer) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.renderTemplate(w, "send_content", nil)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseForm()

		// Parse destination address
		destHex := r.FormValue("destination")
		if len(destHex) != 64 { // 32 bytes as hex = 64 chars
			http.Error(w, "Invalid address format", http.StatusBadRequest)
			return
		}

		destBytes, err := hex.DecodeString(destHex)
		if err != nil || len(destBytes) != 32 {
			http.Error(w, "Invalid address format", http.StatusBadRequest)
			return
		}

		var destination [32]byte
		copy(destination[:], destBytes)

		// Parse amount
		amountStr := r.FormValue("amount")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || amount <= 0 {
			http.Error(w, "Invalid amount", http.StatusBadRequest)
			return
		}

		// Send transaction
		success, err := s.client.SendTxn(destination, amount)
		if err != nil {
			http.Error(w, "Failed to send transaction: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if !success {
			http.Error(w, "Transaction failed", http.StatusInternalServerError)
			return
		}

		// Redirect back to home page after successful transaction
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// handleBalance displays and queries account balances
func (s *WebServer) handleBalance(w http.ResponseWriter, r *http.Request) {
	var addressHex string
	var balance float64
	var err error

	if r.Method == http.MethodPost {
		r.ParseForm()
		addressHex = r.FormValue("address")

		// Validate address format
		if len(addressHex) != 64 {
			http.Error(w, "Invalid address format", http.StatusBadRequest)
			return
		}

		addressBytes, err := hex.DecodeString(addressHex)
		if err != nil || len(addressBytes) != 32 {
			http.Error(w, "Invalid address format", http.StatusBadRequest)
			return
		}

		var address [32]byte
		copy(address[:], addressBytes)

		// Query balance
		balance, err = s.client.GetBalanceByAddress(address)
		if err != nil {
			http.Error(w, "Failed to get balance: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	data := struct {
		Address string
		Balance float64
		Success bool
	}{
		Address: addressHex,
		Balance: balance,
		Success: r.Method == http.MethodPost && err == nil,
	}

	s.renderTemplate(w, "balance_content", data)
}

func (s *WebServer) renderTemplate(w http.ResponseWriter, contentTemplate string, data interface{}) {
	// Create a temporary wrapper template that includes the specified content template
	tmpl, err := s.templates.Clone()
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Define a "content" template that includes the requested content template
	_, err = tmpl.New("content").Parse("{{template \"" + contentTemplate + "\" .}}")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}
