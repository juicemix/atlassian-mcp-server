package domain

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Transport defines the interface for MCP transport mechanisms.
// Implementations handle communication between MCP clients and the server
// using either stdio or HTTP transport.
type Transport interface {
	// Start begins listening for incoming MCP messages.
	// Returns an error if the transport cannot be initialized.
	Start(ctx context.Context) error

	// Send transmits a JSON-RPC response to the client.
	// Returns an error if the response cannot be sent.
	Send(response *Response) error

	// Receive returns a channel for incoming JSON-RPC requests.
	// The channel is closed when the transport is shut down.
	Receive() <-chan *Request

	// Close gracefully shuts down the transport.
	// Returns an error if shutdown fails.
	Close() error
}

// StdioTransport implements Transport using stdin/stdout for communication.
// It reads newline-delimited JSON-RPC messages from stdin and writes
// responses to stdout.
type StdioTransport struct {
	reader  *bufio.Reader
	writer  *bufio.Writer
	reqChan chan *Request
	mu      sync.Mutex
	closed  bool
}

// NewStdioTransport creates a new StdioTransport instance.
// By default, it uses os.Stdin and os.Stdout, but custom readers/writers
// can be provided for testing.
func NewStdioTransport() *StdioTransport {
	return NewStdioTransportWithIO(os.Stdin, os.Stdout)
}

// NewStdioTransportWithIO creates a new StdioTransport with custom IO streams.
// This is primarily used for testing.
func NewStdioTransportWithIO(reader io.Reader, writer io.Writer) *StdioTransport {
	return &StdioTransport{
		reader:  bufio.NewReader(reader),
		writer:  bufio.NewWriter(writer),
		reqChan: make(chan *Request, 10), // Buffered channel to avoid blocking
	}
}

// Start begins reading JSON-RPC messages from stdin.
// It spawns a goroutine that continuously reads newline-delimited messages
// and sends them to the request channel.
func (t *StdioTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return fmt.Errorf("transport is closed")
	}
	t.mu.Unlock()

	go t.readLoop(ctx)
	return nil
}

// readLoop continuously reads from stdin and parses JSON-RPC requests.
func (t *StdioTransport) readLoop(ctx context.Context) {
	defer close(t.reqChan)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Read a line from stdin
			line, err := t.reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return
				}
				// Log error but continue reading
				continue
			}

			// Trim whitespace
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Check for embedded newlines (should not happen with ReadString('\n'))
			// but we validate the JSON doesn't contain literal newlines
			if strings.Contains(line, "\\n") {
				// This is okay - escaped newlines in JSON strings are valid
			}

			// Parse JSON-RPC request
			var req Request
			if err := json.Unmarshal([]byte(line), &req); err != nil {
				// Send parse error response
				t.sendParseError(nil, err)
				continue
			}

			// Validate JSON-RPC version
			if req.JSONRPC != "2.0" {
				t.sendInvalidRequest(req.ID, "invalid jsonrpc version")
				continue
			}

			// Send request to channel
			select {
			case t.reqChan <- &req:
			case <-ctx.Done():
				return
			}
		}
	}
}

// Send writes a JSON-RPC response to stdout.
// The response is serialized as a single line of JSON followed by a newline.
func (t *StdioTransport) Send(response *Response) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return fmt.Errorf("transport is closed")
	}

	// Ensure JSONRPC version is set
	if response.JSONRPC == "" {
		response.JSONRPC = "2.0"
	}

	// Serialize response to JSON
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Check for embedded newlines in the JSON output
	// This should not happen with standard JSON marshaling, but we validate
	jsonStr := string(data)
	if strings.Contains(jsonStr, "\n") {
		return fmt.Errorf("response contains embedded newlines")
	}

	// Write to stdout with newline
	if _, err := t.writer.WriteString(jsonStr + "\n"); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	// Flush to ensure immediate delivery
	if err := t.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush response: %w", err)
	}

	return nil
}

// Receive returns the channel for incoming JSON-RPC requests.
func (t *StdioTransport) Receive() <-chan *Request {
	return t.reqChan
}

// Close gracefully shuts down the transport.
func (t *StdioTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}

	t.closed = true
	// Note: We don't close reqChan here because it's closed by readLoop
	return nil
}

// sendParseError sends a parse error response.
func (t *StdioTransport) sendParseError(id interface{}, err error) {
	response := &Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    ParseError,
			Message: "Parse error",
			Data:    err.Error(),
		},
	}
	// Ignore error since we're already handling an error
	_ = t.Send(response)
}

// sendInvalidRequest sends an invalid request error response.
func (t *StdioTransport) sendInvalidRequest(id interface{}, reason string) {
	response := &Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    InvalidRequest,
			Message: "Invalid Request",
			Data:    reason,
		},
	}
	// Ignore error since we're already handling an error
	_ = t.Send(response)
}

// HTTPTransport implements Transport using HTTP with SSE for communication.
// It exposes two endpoints:
// 1. SSE endpoint (GET) for server-to-client messages
// 2. HTTP POST endpoint for client-to-server messages
type HTTPTransport struct {
	host    string
	port    int
	server  *http.Server
	reqChan chan *Request
	mu      sync.Mutex
	closed  bool
	// Session management for SSE connections
	sessions   map[string]*sseSession
	sessionsMu sync.RWMutex
}

// sseSession represents an active SSE connection
type sseSession struct {
	id            string
	messageChan   chan *Response
	clientWriter  http.ResponseWriter
	clientFlusher http.Flusher
	done          chan struct{}
}

// NewHTTPTransport creates a new HTTPTransport instance.
func NewHTTPTransport(host string, port int) *HTTPTransport {
	return &HTTPTransport{
		host:     host,
		port:     port,
		reqChan:  make(chan *Request, 10),
		sessions: make(map[string]*sseSession),
	}
}

// Start begins the HTTP server and starts listening for incoming requests.
func (t *HTTPTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return fmt.Errorf("transport is closed")
	}
	t.mu.Unlock()

	// Create HTTP server with handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", t.handleSSE)             // SSE endpoint for server-to-client
	mux.HandleFunc("/mcp/message", t.handleMessage) // POST endpoint for client-to-server

	addr := fmt.Sprintf("%s:%d", t.host, t.port)
	t.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server in a goroutine
	go func() {
		if err := t.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Log error but don't fail - server might be stopped gracefully
		}
	}()

	// Monitor context for cancellation
	go func() {
		<-ctx.Done()
		t.Close()
	}()

	return nil
}

// handleSSE handles SSE connections (GET requests) for server-to-client messages.
func (t *HTTPTransport) handleSSE(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[HTTP] %s %s from %s\n", r.Method, r.URL.Path, r.RemoteAddr)

	// Only accept GET requests for SSE
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Create a new session
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())
	session := &sseSession{
		id:            sessionID,
		messageChan:   make(chan *Response, 10),
		clientWriter:  w,
		clientFlusher: flusher,
		done:          make(chan struct{}),
	}

	// Register session
	t.sessionsMu.Lock()
	t.sessions[sessionID] = session
	t.sessionsMu.Unlock()

	// Send endpoint event to tell client where to send messages
	messageEndpoint := fmt.Sprintf("/mcp/message?sessionId=%s", sessionID)
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", messageEndpoint)
	flusher.Flush()

	fmt.Printf("[SSE] Session %s established\n", sessionID)

	// Keep connection alive and send messages
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			// Client disconnected
			fmt.Printf("[SSE] Session %s disconnected\n", sessionID)
			t.sessionsMu.Lock()
			delete(t.sessions, sessionID)
			t.sessionsMu.Unlock()
			close(session.done)
			return
		case <-session.done:
			// Session closed
			return
		case response := <-session.messageChan:
			// Send response as SSE message event
			data, err := json.Marshal(response)
			if err != nil {
				fmt.Printf("[SSE] Failed to marshal response: %v\n", err)
				continue
			}
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(data))
			flusher.Flush()
		case <-ticker.C:
			// Send keep-alive comment
			fmt.Fprintf(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}

// handleMessage handles HTTP POST requests for client-to-server messages.
func (t *HTTPTransport) handleMessage(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[HTTP] %s %s from %s\n", r.Method, r.URL.Path, r.RemoteAddr)

	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get session ID from query parameter
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId parameter", http.StatusBadRequest)
		return
	}

	// Verify session exists
	t.sessionsMu.RLock()
	session, exists := t.sessions[sessionID]
	t.sessionsMu.RUnlock()

	if !exists {
		http.Error(w, "Invalid session", http.StatusBadRequest)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse JSON-RPC request
	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		t.sendErrorToSession(session, nil, ParseError, "Parse error", err.Error())
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		t.sendErrorToSession(session, req.ID, InvalidRequest, "Invalid Request", "invalid jsonrpc version")
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// Send request to processing channel
	select {
	case t.reqChan <- &req:
		// Request accepted
		w.WriteHeader(http.StatusAccepted)
	default:
		// Channel full
		t.sendErrorToSession(session, req.ID, InternalError, "Internal error", "request queue full")
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

// sendErrorToSession sends an error response to a specific session.
func (t *HTTPTransport) sendErrorToSession(session *sseSession, id interface{}, code int, message string, data interface{}) {
	response := &Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	select {
	case session.messageChan <- response:
	default:
		fmt.Printf("[SSE] Failed to send error to session %s: channel full\n", session.id)
	}
}

// Send transmits a JSON-RPC response to the client via SSE.
// For HTTP transport, this sends the response through all active SSE sessions.
func (t *HTTPTransport) Send(response *Response) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return fmt.Errorf("transport is closed")
	}
	t.mu.Unlock()

	// Ensure JSONRPC version is set
	if response.JSONRPC == "" {
		response.JSONRPC = "2.0"
	}

	// Send to all active sessions
	t.sessionsMu.RLock()
	defer t.sessionsMu.RUnlock()

	if len(t.sessions) == 0 {
		return fmt.Errorf("no active sessions")
	}

	for _, session := range t.sessions {
		select {
		case session.messageChan <- response:
			// Message sent successfully
		default:
			fmt.Printf("[SSE] Failed to send to session %s: channel full\n", session.id)
		}
	}

	return nil
}

// Receive returns the channel for incoming JSON-RPC requests.
func (t *HTTPTransport) Receive() <-chan *Request {
	return t.reqChan
}

// Close gracefully shuts down the HTTP server and all SSE sessions.
func (t *HTTPTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}

	t.closed = true

	// Close all sessions
	t.sessionsMu.Lock()
	for _, session := range t.sessions {
		close(session.done)
	}
	t.sessions = make(map[string]*sseSession)
	t.sessionsMu.Unlock()

	// Close the request channel
	close(t.reqChan)

	// Shutdown the HTTP server if it exists
	if t.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return t.server.Shutdown(ctx)
	}

	return nil
}
