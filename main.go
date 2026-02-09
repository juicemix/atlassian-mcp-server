package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"atlassian-mcp-server/internal/application"
	"atlassian-mcp-server/internal/domain"
	"atlassian-mcp-server/internal/infrastructure"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	log.Printf("Loading configuration from: %s", *configPath)
	config, err := domain.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Println("Configuration loaded successfully")

	// Create authentication manager
	authManager := domain.NewAuthenticationManagerFromConfig(config)
	log.Println("Authentication manager initialized")

	// Create response mapper
	mapper := domain.NewResponseMapper()

	// Create API clients and handlers for each configured tool
	var handlers []domain.ToolHandler

	// Jira
	if config.Tools.Jira != nil {
		log.Println("Initializing Jira client and handler")

		var jiraClient *infrastructure.JiraClient

		// Only create default client if auth is configured
		if config.Tools.Jira.Auth != nil {
			httpClient, err := authManager.GetAuthenticatedClient("jira")
			if err != nil {
				log.Fatalf("Failed to create authenticated client for Jira: %v", err)
			}
			jiraClient = infrastructure.NewJiraClient(config.Tools.Jira.BaseURL, httpClient)
		} else {
			// No default credentials - client must provide credentials
			log.Println("Jira configured without default credentials - clients must provide auth")
			jiraClient = nil
		}

		jiraHandler := application.NewJiraHandler(jiraClient, mapper, authManager, config.Tools.Jira.BaseURL)
		handlers = append(handlers, jiraHandler)
		log.Println("Jira handler registered")
	}

	// Confluence
	if config.Tools.Confluence != nil {
		log.Println("Initializing Confluence client and handler")
		httpClient, err := authManager.GetAuthenticatedClient("confluence")
		if err != nil {
			log.Fatalf("Failed to create authenticated client for Confluence: %v", err)
		}
		confluenceClient := infrastructure.NewConfluenceClient(config.Tools.Confluence.BaseURL, httpClient)
		confluenceHandler := application.NewConfluenceHandler(confluenceClient, mapper)
		handlers = append(handlers, confluenceHandler)
		log.Println("Confluence handler registered")
	}

	// Bitbucket
	if config.Tools.Bitbucket != nil {
		log.Println("Initializing Bitbucket client and handler")
		httpClient, err := authManager.GetAuthenticatedClient("bitbucket")
		if err != nil {
			log.Fatalf("Failed to create authenticated client for Bitbucket: %v", err)
		}
		bitbucketClient := infrastructure.NewBitbucketClient(config.Tools.Bitbucket.BaseURL, httpClient)
		bitbucketHandler := application.NewBitbucketHandler(bitbucketClient, mapper)
		handlers = append(handlers, bitbucketHandler)
		log.Println("Bitbucket handler registered")
	}

	// Bamboo
	if config.Tools.Bamboo != nil {
		log.Println("Initializing Bamboo client and handler")
		httpClient, err := authManager.GetAuthenticatedClient("bamboo")
		if err != nil {
			log.Fatalf("Failed to create authenticated client for Bamboo: %v", err)
		}
		bambooClient := infrastructure.NewBambooClient(config.Tools.Bamboo.BaseURL, httpClient)
		bambooHandler := application.NewBambooHandler(bambooClient, mapper)
		handlers = append(handlers, bambooHandler)
		log.Println("Bamboo handler registered")
	}

	// Verify at least one handler is registered
	if len(handlers) == 0 {
		log.Fatal("No tools configured - at least one Atlassian tool must be configured")
	}

	// Create request router with all handlers
	router := application.NewRequestRouter(handlers...)
	log.Printf("Request router initialized with %d handler(s)", len(handlers))

	// Create transport based on configuration
	var transport domain.Transport
	switch config.Transport.Type {
	case "stdio":
		log.Println("Initializing stdio transport")
		transport = domain.NewStdioTransport()
	case "http":
		log.Printf("Initializing HTTP transport on %s:%d", config.Transport.HTTP.Host, config.Transport.HTTP.Port)
		transport = domain.NewHTTPTransport(config.Transport.HTTP.Host, config.Transport.HTTP.Port)
	default:
		log.Fatalf("Invalid transport type: %s", config.Transport.Type)
	}

	// Create server with all dependencies
	server := application.NewServer(transport, router, authManager, config)
	log.Println("MCP server created")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Println("Starting MCP server...")
		if err := server.Start(ctx); err != nil {
			errChan <- fmt.Errorf("server failed to start: %w", err)
		}
	}()

	// Log successful startup
	if config.Transport.Type == "stdio" {
		log.Println("MCP server started successfully (stdio transport)")
	} else {
		log.Printf("MCP server started successfully (HTTP transport on %s:%d)",
			config.Transport.HTTP.Host, config.Transport.HTTP.Port)
	}

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
		log.Println("Initiating graceful shutdown...")
		cancel()
	case err := <-errChan:
		log.Printf("Server error: %v", err)
		cancel()
		if err := server.Close(); err != nil {
			log.Printf("Error closing server: %v", err)
		}
		os.Exit(1)
	}

	// Close the server
	log.Println("Closing server...")
	if err := server.Close(); err != nil {
		log.Printf("Error during server shutdown: %v", err)
		os.Exit(1)
	}

	log.Println("Server shutdown complete")
}
