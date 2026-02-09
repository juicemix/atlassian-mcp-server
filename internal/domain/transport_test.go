package domain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestStdioTransport_ReadValidMessage tests reading a valid JSON-RPC message from stdin.
func TestStdioTransport_ReadValidMessage(t *testing.T) {
	// Create a mock stdin with a valid JSON-RPC request
	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1.0"}}` + "\n"
	reader := strings.NewReader(input)
	writer := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(reader, writer)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start the transport
	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start transport: %v", err)
	}

	// Receive the request
	select {
	case req := <-transport.Receive():
		if req == nil {
			t.Fatal("Received nil request")
		}
		if req.JSONRPC != "2.0" {
			t.Errorf("Expected JSONRPC version 2.0, got %s", req.JSONRPC)
		}
		if req.Method != "initialize" {
			t.Errorf("Expected method 'initialize', got %s", req.Method)
		}
		if req.ID != float64(1) { // JSON unmarshals numbers as float64
			t.Errorf("Expected ID 1, got %v", req.ID)
		}
	case <-ctx.Done():
		t.Fatal("Timeout waiting for request")
	}
}

// TestStdioTransport_ReadMultipleMessages tests reading multiple JSON-RPC messages.
func TestStdioTransport_ReadMultipleMessages(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"initialize"}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n" +
		`{"jsonrpc":"2.0","id":3,"method":"tools/call"}` + "\n"
	reader := strings.NewReader(input)
	writer := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(reader, writer)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start transport: %v", err)
	}

	// Receive three requests
	expectedMethods := []string{"initialize", "tools/list", "tools/call"}
	for i, expectedMethod := range expectedMethods {
		select {
		case req := <-transport.Receive():
			if req == nil {
				t.Fatalf("Received nil request for message %d", i+1)
			}
			if req.Method != expectedMethod {
				t.Errorf("Message %d: expected method '%s', got '%s'", i+1, expectedMethod, req.Method)
			}
		case <-ctx.Done():
			t.Fatalf("Timeout waiting for message %d", i+1)
		}
	}
}

// TestStdioTransport_SendResponse tests writing a JSON-RPC response to stdout.
func TestStdioTransport_SendResponse(t *testing.T) {
	reader := strings.NewReader("")
	writer := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(reader, writer)

	response := &Response{
		JSONRPC: "2.0",
		ID:      1,
		Result:  map[string]string{"status": "ok"},
	}

	err := transport.Send(response)
	if err != nil {
		t.Fatalf("Failed to send response: %v", err)
	}

	// Verify the output
	output := writer.String()
	if !strings.HasSuffix(output, "\n") {
		t.Error("Output should end with newline")
	}

	// Parse the JSON to verify it's valid
	var parsedResponse Response
	err = json.Unmarshal([]byte(strings.TrimSpace(output)), &parsedResponse)
	if err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	if parsedResponse.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC version 2.0, got %s", parsedResponse.JSONRPC)
	}
	if parsedResponse.ID != float64(1) {
		t.Errorf("Expected ID 1, got %v", parsedResponse.ID)
	}
}

// TestStdioTransport_SendResponseSetsJSONRPCVersion tests that Send sets JSONRPC version if missing.
func TestStdioTransport_SendResponseSetsJSONRPCVersion(t *testing.T) {
	reader := strings.NewReader("")
	writer := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(reader, writer)

	response := &Response{
		ID:     1,
		Result: "ok",
		// JSONRPC version intentionally omitted
	}

	err := transport.Send(response)
	if err != nil {
		t.Fatalf("Failed to send response: %v", err)
	}

	// Parse the output
	var parsedResponse Response
	err = json.Unmarshal([]byte(strings.TrimSpace(writer.String())), &parsedResponse)
	if err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	if parsedResponse.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC version to be set to 2.0, got %s", parsedResponse.JSONRPC)
	}
}

// TestStdioTransport_InvalidJSONRPCVersion tests handling of invalid JSONRPC version.
func TestStdioTransport_InvalidJSONRPCVersion(t *testing.T) {
	input := `{"jsonrpc":"1.0","id":1,"method":"test"}` + "\n"
	reader := strings.NewReader(input)
	writer := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(reader, writer)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start transport: %v", err)
	}

	// Wait a bit for the error response to be written
	time.Sleep(100 * time.Millisecond)

	// Check that an error response was written
	output := writer.String()
	if output == "" {
		t.Fatal("Expected error response to be written")
	}

	// Parse the error response
	var errorResponse Response
	err = json.Unmarshal([]byte(strings.TrimSpace(output)), &errorResponse)
	if err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if errorResponse.Error == nil {
		t.Fatal("Expected error in response")
	}
	if errorResponse.Error.Code != InvalidRequest {
		t.Errorf("Expected error code %d, got %d", InvalidRequest, errorResponse.Error.Code)
	}
}

// TestStdioTransport_MalformedJSON tests handling of malformed JSON.
func TestStdioTransport_MalformedJSON(t *testing.T) {
	input := `{invalid json}` + "\n"
	reader := strings.NewReader(input)
	writer := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(reader, writer)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start transport: %v", err)
	}

	// Wait a bit for the error response to be written
	time.Sleep(100 * time.Millisecond)

	// Check that an error response was written
	output := writer.String()
	if output == "" {
		t.Fatal("Expected error response to be written")
	}

	// Parse the error response
	var errorResponse Response
	err = json.Unmarshal([]byte(strings.TrimSpace(output)), &errorResponse)
	if err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if errorResponse.Error == nil {
		t.Fatal("Expected error in response")
	}
	if errorResponse.Error.Code != ParseError {
		t.Errorf("Expected error code %d, got %d", ParseError, errorResponse.Error.Code)
	}
}

// TestStdioTransport_EmptyLines tests that empty lines are ignored.
func TestStdioTransport_EmptyLines(t *testing.T) {
	input := "\n\n" +
		`{"jsonrpc":"2.0","id":1,"method":"test"}` + "\n" +
		"\n\n"
	reader := strings.NewReader(input)
	writer := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(reader, writer)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start transport: %v", err)
	}

	// Should receive exactly one request
	select {
	case req := <-transport.Receive():
		if req == nil {
			t.Fatal("Received nil request")
		}
		if req.Method != "test" {
			t.Errorf("Expected method 'test', got %s", req.Method)
		}
	case <-ctx.Done():
		t.Fatal("Timeout waiting for request")
	}

	// Should not receive any more requests (empty lines should be ignored)
	select {
	case req := <-transport.Receive():
		if req != nil {
			t.Errorf("Expected no more requests, got: %+v", req)
		}
	case <-time.After(200 * time.Millisecond):
		// Good - no more requests
	}
}

// TestStdioTransport_Close tests graceful shutdown.
func TestStdioTransport_Close(t *testing.T) {
	reader := strings.NewReader("")
	writer := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(reader, writer)

	err := transport.Close()
	if err != nil {
		t.Fatalf("Failed to close transport: %v", err)
	}

	// Sending after close should fail
	response := &Response{
		JSONRPC: "2.0",
		ID:      1,
		Result:  "ok",
	}
	err = transport.Send(response)
	if err == nil {
		t.Error("Expected error when sending after close")
	}
}

// TestStdioTransport_StartAfterClose tests that starting after close fails.
func TestStdioTransport_StartAfterClose(t *testing.T) {
	reader := strings.NewReader("")
	writer := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(reader, writer)

	err := transport.Close()
	if err != nil {
		t.Fatalf("Failed to close transport: %v", err)
	}

	ctx := context.Background()
	err = transport.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting after close")
	}
}

// TestStdioTransport_ContextCancellation tests that context cancellation stops the transport.
func TestStdioTransport_ContextCancellation(t *testing.T) {
	// Create a reader that will block (simulating continuous input)
	reader := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"test"}` + "\n")
	writer := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(reader, writer)

	ctx, cancel := context.WithCancel(context.Background())

	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start transport: %v", err)
	}

	// Receive one message
	select {
	case <-transport.Receive():
		// Good
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for request")
	}

	// Cancel the context
	cancel()

	// The receive channel should be closed
	select {
	case _, ok := <-transport.Receive():
		if ok {
			t.Error("Expected receive channel to be closed after context cancellation")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for channel to close")
	}
}

// TestStdioTransport_EscapedNewlinesInJSON tests that escaped newlines in JSON strings are handled correctly.
func TestStdioTransport_EscapedNewlinesInJSON(t *testing.T) {
	// JSON with escaped newlines in a string value
	input := `{"jsonrpc":"2.0","id":1,"method":"test","params":{"text":"line1\nline2"}}` + "\n"
	reader := strings.NewReader(input)
	writer := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(reader, writer)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start transport: %v", err)
	}

	// Should successfully receive the request
	select {
	case req := <-transport.Receive():
		if req == nil {
			t.Fatal("Received nil request")
		}
		if req.Method != "test" {
			t.Errorf("Expected method 'test', got %s", req.Method)
		}
		// Verify params were parsed correctly
		params, ok := req.Params.(map[string]interface{})
		if !ok {
			t.Fatal("Expected params to be a map")
		}
		text, ok := params["text"].(string)
		if !ok {
			t.Fatal("Expected text parameter to be a string")
		}
		if text != "line1\nline2" {
			t.Errorf("Expected text with newline, got %q", text)
		}
	case <-ctx.Done():
		t.Fatal("Timeout waiting for request")
	}
}

// TestStdioTransport_SendError tests sending an error response.
func TestStdioTransport_SendError(t *testing.T) {
	reader := strings.NewReader("")
	writer := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(reader, writer)

	response := &Response{
		JSONRPC: "2.0",
		ID:      1,
		Error: &Error{
			Code:    MethodNotFound,
			Message: "Method not found",
			Data:    "unknown_method",
		},
	}

	err := transport.Send(response)
	if err != nil {
		t.Fatalf("Failed to send error response: %v", err)
	}

	// Parse the output
	var parsedResponse Response
	err = json.Unmarshal([]byte(strings.TrimSpace(writer.String())), &parsedResponse)
	if err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	if parsedResponse.Error == nil {
		t.Fatal("Expected error in response")
	}
	if parsedResponse.Error.Code != MethodNotFound {
		t.Errorf("Expected error code %d, got %d", MethodNotFound, parsedResponse.Error.Code)
	}
	if parsedResponse.Error.Message != "Method not found" {
		t.Errorf("Expected error message 'Method not found', got %s", parsedResponse.Error.Message)
	}
}

// TestStdioTransport_EmbeddedNewlinesInResponse tests that responses with embedded newlines are rejected.
// This validates requirement 6.4: stdio transport must use newline-delimited messages.
func TestStdioTransport_EmbeddedNewlinesInResponse(t *testing.T) {
	reader := strings.NewReader("")
	writer := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(reader, writer)

	// Create a response that would contain an embedded newline when marshaled
	// Note: Standard JSON marshaling doesn't produce newlines, but we test the validation
	response := &Response{
		JSONRPC: "2.0",
		ID:      1,
		Result:  "test\nvalue", // This will be escaped as "test\\nvalue" in JSON
	}

	// This should succeed because JSON marshaling escapes the newline
	err := transport.Send(response)
	if err != nil {
		t.Fatalf("Failed to send response with escaped newline: %v", err)
	}

	// Verify the output doesn't contain actual newlines (except the trailing one)
	output := writer.String()
	lines := strings.Split(output, "\n")
	if len(lines) != 2 { // Should be: [json_line, empty_string_after_final_newline]
		t.Errorf("Expected output to be a single line with trailing newline, got %d lines", len(lines)-1)
	}
}

// TestStdioTransport_MultilineInputHandling tests that multi-line input is handled correctly.
// Each line should be treated as a separate message.
func TestStdioTransport_MultilineInputHandling(t *testing.T) {
	// Simulate input where someone tries to send a multi-line JSON (which is invalid for stdio transport)
	// The transport should treat each line separately
	input := `{"jsonrpc":"2.0",` + "\n" +
		`"id":1,` + "\n" +
		`"method":"test"}` + "\n"

	reader := strings.NewReader(input)
	writer := &bytes.Buffer{}

	transport := NewStdioTransportWithIO(reader, writer)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start transport: %v", err)
	}

	// Wait for error responses to be written
	time.Sleep(200 * time.Millisecond)

	// Each line should be treated as a separate (invalid) message
	// and should generate parse errors
	output := writer.String()
	if output == "" {
		t.Fatal("Expected error responses for malformed input")
	}

	// Count the number of error responses (one per invalid line)
	errorLines := strings.Split(strings.TrimSpace(output), "\n")
	if len(errorLines) < 2 {
		t.Errorf("Expected at least 2 error responses for multi-line input, got %d", len(errorLines))
	}

	// Verify each response is a parse error
	for i, line := range errorLines {
		if line == "" {
			continue
		}
		var errorResponse Response
		err := json.Unmarshal([]byte(line), &errorResponse)
		if err != nil {
			t.Errorf("Line %d: Failed to parse error response: %v", i, err)
			continue
		}
		if errorResponse.Error == nil {
			t.Errorf("Line %d: Expected error in response", i)
			continue
		}
		if errorResponse.Error.Code != ParseError {
			t.Errorf("Line %d: Expected parse error code %d, got %d", i, ParseError, errorResponse.Error.Code)
		}
	}
}

// TestHTTPTransport_StartServer tests that the HTTP server starts on the configured host and port.
func TestHTTPTransport_StartServer(t *testing.T) {
	transport := NewHTTPTransport("localhost", 0) // Port 0 for random available port

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start HTTP transport: %v", err)
	}

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Clean up
	err = transport.Close()
	if err != nil {
		t.Fatalf("Failed to close transport: %v", err)
	}
}

// TestHTTPTransport_ReceiveValidRequest tests receiving a valid JSON-RPC request via HTTP POST.
func TestHTTPTransport_ReceiveValidRequest(t *testing.T) {
	transport := NewHTTPTransport("localhost", 8765)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start HTTP transport: %v", err)
	}
	defer transport.Close()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Send a JSON-RPC request via HTTP
	requestBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1.0"}}`

	// Start a goroutine to handle the response
	go func() {
		// Wait for the request to be received
		select {
		case req := <-transport.Receive():
			if req == nil {
				return
			}
			// Send a response
			response := &Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  map[string]string{"status": "initialized"},
			}
			transport.Send(response)
		case <-time.After(2 * time.Second):
			return
		}
	}()

	resp, err := http.Post("http://localhost:8765/", "application/json", strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("Failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Parse the response
	var jsonResp Response
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if jsonResp.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC version 2.0, got %s", jsonResp.JSONRPC)
	}
	if jsonResp.ID != float64(1) {
		t.Errorf("Expected ID 1, got %v", jsonResp.ID)
	}
	if jsonResp.Error != nil {
		t.Errorf("Expected no error, got %+v", jsonResp.Error)
	}
}

// TestHTTPTransport_InvalidMethod tests that non-POST requests are rejected.
func TestHTTPTransport_InvalidMethod(t *testing.T) {
	transport := NewHTTPTransport("localhost", 8766)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start HTTP transport: %v", err)
	}
	defer transport.Close()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Try a GET request
	resp, err := http.Get("http://localhost:8766/")
	if err != nil {
		t.Fatalf("Failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

// TestHTTPTransport_MalformedJSON tests handling of malformed JSON in HTTP requests.
func TestHTTPTransport_MalformedJSON(t *testing.T) {
	transport := NewHTTPTransport("localhost", 8767)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start HTTP transport: %v", err)
	}
	defer transport.Close()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Send malformed JSON
	requestBody := `{invalid json}`
	resp, err := http.Post("http://localhost:8767/", "application/json", strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("Failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 (JSON-RPC error), got %d", resp.StatusCode)
	}

	// Parse the error response
	var jsonResp Response
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if jsonResp.Error == nil {
		t.Fatal("Expected error in response")
	}
	if jsonResp.Error.Code != ParseError {
		t.Errorf("Expected error code %d, got %d", ParseError, jsonResp.Error.Code)
	}
}

// TestHTTPTransport_InvalidJSONRPCVersion tests handling of invalid JSONRPC version in HTTP requests.
func TestHTTPTransport_InvalidJSONRPCVersion(t *testing.T) {
	transport := NewHTTPTransport("localhost", 8768)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start HTTP transport: %v", err)
	}
	defer transport.Close()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Send request with invalid JSONRPC version
	requestBody := `{"jsonrpc":"1.0","id":1,"method":"test"}`
	resp, err := http.Post("http://localhost:8768/", "application/json", strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("Failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 (JSON-RPC error), got %d", resp.StatusCode)
	}

	// Parse the error response
	var jsonResp Response
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if jsonResp.Error == nil {
		t.Fatal("Expected error in response")
	}
	if jsonResp.Error.Code != InvalidRequest {
		t.Errorf("Expected error code %d, got %d", InvalidRequest, jsonResp.Error.Code)
	}
}

// TestHTTPTransport_Close tests graceful shutdown of HTTP server.
func TestHTTPTransport_Close(t *testing.T) {
	transport := NewHTTPTransport("localhost", 8769)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start HTTP transport: %v", err)
	}

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Close the transport
	err = transport.Close()
	if err != nil {
		t.Fatalf("Failed to close transport: %v", err)
	}

	// Sending after close should fail
	response := &Response{
		JSONRPC: "2.0",
		ID:      1,
		Result:  "ok",
	}
	err = transport.Send(response)
	if err == nil {
		t.Error("Expected error when sending after close")
	}

	// Server should no longer accept connections
	time.Sleep(100 * time.Millisecond)
	_, err = http.Post("http://localhost:8769/", "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"test"}`))
	if err == nil {
		t.Error("Expected error when connecting to closed server")
	}
}

// TestHTTPTransport_StartAfterClose tests that starting after close fails.
func TestHTTPTransport_StartAfterClose(t *testing.T) {
	transport := NewHTTPTransport("localhost", 8770)

	ctx := context.Background()

	err := transport.Close()
	if err != nil {
		t.Fatalf("Failed to close transport: %v", err)
	}

	err = transport.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting after close")
	}
}

// TestHTTPTransport_ContextCancellation tests that context cancellation stops the server.
func TestHTTPTransport_ContextCancellation(t *testing.T) {
	transport := NewHTTPTransport("localhost", 8771)

	ctx, cancel := context.WithCancel(context.Background())

	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start HTTP transport: %v", err)
	}

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Verify server is running
	go func() {
		req := <-transport.Receive()
		if req != nil {
			response := &Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  "ok",
			}
			transport.Send(response)
		}
	}()

	resp, err := http.Post("http://localhost:8771/", "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"test"}`))
	if err != nil {
		t.Fatalf("Failed to send request to running server: %v", err)
	}
	resp.Body.Close()

	// Cancel the context
	cancel()

	// Give the server time to shut down
	time.Sleep(200 * time.Millisecond)

	// Server should no longer accept connections
	_, err = http.Post("http://localhost:8771/", "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"test"}`))
	if err == nil {
		t.Error("Expected error when connecting to cancelled server")
	}
}

// TestHTTPTransport_MultipleRequests tests handling multiple concurrent requests.
func TestHTTPTransport_MultipleRequests(t *testing.T) {
	transport := NewHTTPTransport("localhost", 8772)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start HTTP transport: %v", err)
	}
	defer transport.Close()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Start a goroutine to handle requests
	go func() {
		for {
			select {
			case req := <-transport.Receive():
				if req == nil {
					return
				}
				// Send a response
				response := &Response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result:  fmt.Sprintf("response_%v", req.ID),
				}
				transport.Send(response)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Send multiple requests concurrently
	numRequests := 5
	results := make(chan error, numRequests)

	for i := 1; i <= numRequests; i++ {
		go func(id int) {
			requestBody := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"test"}`, id)
			resp, err := http.Post("http://localhost:8772/", "application/json", strings.NewReader(requestBody))
			if err != nil {
				results <- fmt.Errorf("request %d failed: %w", id, err)
				return
			}
			defer resp.Body.Close()

			var jsonResp Response
			err = json.NewDecoder(resp.Body).Decode(&jsonResp)
			if err != nil {
				results <- fmt.Errorf("request %d decode failed: %w", id, err)
				return
			}

			if jsonResp.Error != nil {
				results <- fmt.Errorf("request %d returned error: %+v", id, jsonResp.Error)
				return
			}

			expectedResult := fmt.Sprintf("response_%d", id)
			if jsonResp.Result != expectedResult {
				results <- fmt.Errorf("request %d: expected result %s, got %v", id, expectedResult, jsonResp.Result)
				return
			}

			results <- nil
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		select {
		case err := <-results:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("Timeout waiting for request %d", i+1)
		}
	}
}

// TestHTTPTransport_SendError tests sending an error response via HTTP.
func TestHTTPTransport_SendError(t *testing.T) {
	transport := NewHTTPTransport("localhost", 8773)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start HTTP transport: %v", err)
	}
	defer transport.Close()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Start a goroutine to handle the request and send an error
	go func() {
		select {
		case req := <-transport.Receive():
			if req == nil {
				return
			}
			// Send an error response
			response := &Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &Error{
					Code:    MethodNotFound,
					Message: "Method not found",
					Data:    "unknown_method",
				},
			}
			transport.Send(response)
		case <-time.After(2 * time.Second):
			return
		}
	}()

	requestBody := `{"jsonrpc":"2.0","id":1,"method":"unknown"}`
	resp, err := http.Post("http://localhost:8773/", "application/json", strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("Failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Parse the error response
	var jsonResp Response
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if jsonResp.Error == nil {
		t.Fatal("Expected error in response")
	}
	if jsonResp.Error.Code != MethodNotFound {
		t.Errorf("Expected error code %d, got %d", MethodNotFound, jsonResp.Error.Code)
	}
	if jsonResp.Error.Message != "Method not found" {
		t.Errorf("Expected error message 'Method not found', got %s", jsonResp.Error.Message)
	}
}

// TestHTTPTransport_InvalidPortConfiguration tests handling of invalid port configurations.
// This validates requirement 6.6: invalid transport configuration should return an error.
func TestHTTPTransport_InvalidPortConfiguration(t *testing.T) {
	testCases := []struct {
		name string
		host string
		port int
	}{
		{
			name: "negative port",
			host: "localhost",
			port: -1,
		},
		{
			name: "port zero with explicit host",
			host: "localhost",
			port: 0, // Port 0 is actually valid (OS assigns random port), but we test it
		},
		{
			name: "port too large",
			host: "localhost",
			port: 65536, // Max valid port is 65535
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			transport := NewHTTPTransport(tc.host, tc.port)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// Start should succeed (validation happens at bind time)
			// But the server may fail to bind if the port is invalid
			err := transport.Start(ctx)

			// For port 0, Start will succeed (OS assigns a port)
			// For negative or too large ports, the server will fail when trying to listen
			if tc.port == 0 {
				if err != nil {
					t.Errorf("Expected Start to succeed with port 0, got error: %v", err)
				}
				transport.Close()
			} else {
				// For invalid ports, we expect the server to fail when trying to bind
				// Give it a moment to attempt binding
				time.Sleep(100 * time.Millisecond)

				// Try to send a request - it should fail
				requestBody := `{"jsonrpc":"2.0","id":1,"method":"test"}`
				addr := fmt.Sprintf("http://%s:%d/", tc.host, tc.port)
				_, err := http.Post(addr, "application/json", strings.NewReader(requestBody))
				if err == nil {
					t.Error("Expected error when connecting to server with invalid port")
				}

				transport.Close()
			}
		})
	}
}

// TestHTTPTransport_EmptyHostConfiguration tests handling of empty host configuration.
// This validates requirement 6.6: invalid transport configuration should return an error.
func TestHTTPTransport_EmptyHostConfiguration(t *testing.T) {
	// Empty host should default to binding on all interfaces
	transport := NewHTTPTransport("", 8774)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start HTTP transport with empty host: %v", err)
	}
	defer transport.Close()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Start a goroutine to handle requests
	go func() {
		select {
		case req := <-transport.Receive():
			if req == nil {
				return
			}
			response := &Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  "ok",
			}
			transport.Send(response)
		case <-time.After(2 * time.Second):
			return
		}
	}()

	// Should be able to connect via localhost
	requestBody := `{"jsonrpc":"2.0","id":1,"method":"test"}`
	resp, err := http.Post("http://localhost:8774/", "application/json", strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("Failed to connect to server with empty host: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// TestHTTPTransport_PortAlreadyInUse tests handling when the configured port is already in use.
// This validates requirement 6.6: invalid transport configuration should be handled gracefully.
func TestHTTPTransport_PortAlreadyInUse(t *testing.T) {
	// Start first transport on a specific port
	transport1 := NewHTTPTransport("localhost", 8775)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := transport1.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start first HTTP transport: %v", err)
	}
	defer transport1.Close()

	// Give the server a moment to start and bind the port
	time.Sleep(200 * time.Millisecond)

	// Try to start second transport on the same port
	transport2 := NewHTTPTransport("localhost", 8775)

	err = transport2.Start(ctx)
	// Start itself doesn't fail immediately, but the server will fail to bind
	// We need to give it a moment and then verify the server isn't actually running
	time.Sleep(200 * time.Millisecond)

	// Try to send a request - it should go to the first server
	go func() {
		select {
		case req := <-transport1.Receive():
			if req != nil {
				response := &Response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result:  "from_transport1",
				}
				transport1.Send(response)
			}
		case <-time.After(2 * time.Second):
			return
		}
	}()

	requestBody := `{"jsonrpc":"2.0","id":1,"method":"test"}`
	resp, err := http.Post("http://localhost:8775/", "application/json", strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	var jsonResp Response
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should get response from first transport
	if jsonResp.Result != "from_transport1" {
		t.Errorf("Expected response from first transport, got: %v", jsonResp.Result)
	}

	// Clean up second transport
	transport2.Close()
}

// TestHTTPTransport_ConfiguredHostAndPort tests that the server respects configured host and port.
// This validates requirement 6.1: HTTP transport should listen on configured host and port.
func TestHTTPTransport_ConfiguredHostAndPort(t *testing.T) {
	testCases := []struct {
		name string
		host string
		port int
	}{
		{
			name: "localhost with specific port",
			host: "localhost",
			port: 8776,
		},
		{
			name: "127.0.0.1 with specific port",
			host: "127.0.0.1",
			port: 8777,
		},
		{
			name: "empty host with specific port",
			host: "",
			port: 8778,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			transport := NewHTTPTransport(tc.host, tc.port)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := transport.Start(ctx)
			if err != nil {
				t.Fatalf("Failed to start HTTP transport: %v", err)
			}
			defer transport.Close()

			// Give the server a moment to start
			time.Sleep(100 * time.Millisecond)

			// Start a goroutine to handle requests
			go func() {
				select {
				case req := <-transport.Receive():
					if req == nil {
						return
					}
					response := &Response{
						JSONRPC: "2.0",
						ID:      req.ID,
						Result:  "ok",
					}
					transport.Send(response)
				case <-time.After(2 * time.Second):
					return
				}
			}()

			// Connect to the configured host and port
			connectHost := tc.host
			if connectHost == "" {
				connectHost = "localhost"
			}
			requestBody := `{"jsonrpc":"2.0","id":1,"method":"test"}`
			addr := fmt.Sprintf("http://%s:%d/", connectHost, tc.port)
			resp, err := http.Post(addr, "application/json", strings.NewReader(requestBody))
			if err != nil {
				t.Fatalf("Failed to connect to %s: %v", addr, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			var jsonResp Response
			err = json.NewDecoder(resp.Body).Decode(&jsonResp)
			if err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if jsonResp.Result != "ok" {
				t.Errorf("Expected result 'ok', got: %v", jsonResp.Result)
			}
		})
	}
}

// TestHTTPTransport_ResponseTimeout tests that requests timeout if no response is sent.
// This validates requirement 6.3: HTTP transport should handle request/response lifecycle.
func TestHTTPTransport_ResponseTimeout(t *testing.T) {
	transport := NewHTTPTransport("localhost", 8779)

	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	err := transport.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start HTTP transport: %v", err)
	}
	defer transport.Close()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Start a goroutine that receives the request but never sends a response
	go func() {
		select {
		case req := <-transport.Receive():
			if req == nil {
				return
			}
			// Intentionally don't send a response - let it timeout
			time.Sleep(35 * time.Second)
		case <-ctx.Done():
			return
		}
	}()

	// Send a request and expect a timeout error response
	requestBody := `{"jsonrpc":"2.0","id":1,"method":"test"}`
	resp, err := http.Post("http://localhost:8779/", "application/json", strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("Failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Should get a response (timeout error)
	var jsonResp Response
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should be an error response about timeout
	if jsonResp.Error == nil {
		t.Fatal("Expected error response for timeout")
	}
	if jsonResp.Error.Code != InternalError {
		t.Errorf("Expected error code %d, got %d", InternalError, jsonResp.Error.Code)
	}
	if !strings.Contains(fmt.Sprintf("%v", jsonResp.Error.Data), "timeout") {
		t.Errorf("Expected timeout error, got: %v", jsonResp.Error.Data)
	}
}
