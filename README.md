# Atlassian MCP Server

A Model Context Protocol (MCP) server that provides a unified interface to interact with Atlassian tools: Jira Server 9.12, Confluence Server 8.15, Bitbucket 8.9, and Bamboo 9.2.7.

## Features

- **Multi-tool Support**: Integrate with Jira, Confluence, Bitbucket, and Bamboo through a single MCP server
- **Flexible Transport**: Supports both stdio and HTTP transport mechanisms
- **Clean Architecture**: Well-structured codebase following clean architecture principles
- **Type-Safe**: Leverages Go's type system for compile-time guarantees
- **Comprehensive Error Handling**: Detailed error mapping between Atlassian APIs and MCP protocol

## Prerequisites

- Go 1.24.1 or later
- Access to one or more Atlassian tools (Jira, Confluence, Bitbucket, Bamboo)
- Valid credentials (username/password or personal access token) for each tool

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd atlassian-mcp-server
```

2. Install dependencies:
```bash
go mod download
```

3. Build the server:
```bash
go build -o atlassian-mcp-server main.go
```

## Configuration

1. Copy the example configuration file:
```bash
cp config.yaml.example config.yaml
```

2. Edit `config.yaml` with your Atlassian tool details:
   - Set the `base_url` for each tool you want to use
   - Configure authentication (basic or token-based)
   - Choose transport type (stdio or http)

### Configuration Structure

```yaml
transport:
  type: stdio  # or "http"
  # http:      # Only required for HTTP transport
  #   host: localhost
  #   port: 8080

tools:
  jira:
    base_url: https://jira.example.com
    auth:
      type: basic  # or "token"
      username: your-username
      password: your-password
      # token: your-token  # For token auth
  
  # Configure other tools similarly...
```

### Transport Options

**Stdio Transport** (default):
- Reads JSON-RPC messages from stdin
- Writes responses to stdout
- Ideal for process-based integrations

**HTTP Transport**:
- Exposes an HTTP endpoint for JSON-RPC messages
- Suitable for remote integrations
- Requires host and port configuration

### Authentication Methods

**Basic Authentication**:
```yaml
auth:
  type: basic
  username: your-username
  password: your-password
```

**Token Authentication**:
```yaml
auth:
  type: token
  token: your-personal-access-token
```

## Usage

### Running with Default Configuration

```bash
./atlassian-mcp-server
```

This will load configuration from `config.yaml` in the current directory.

### Running with Custom Configuration

```bash
./atlassian-mcp-server -config /path/to/config.yaml
```

### Command-Line Flags

- `-config`: Path to configuration file (default: `config.yaml`)

## Available Tools

### Jira Operations

- `jira_get_issue`: Retrieve a Jira issue by key
- `jira_create_issue`: Create a new Jira issue
- `jira_update_issue`: Update an existing issue
- `jira_delete_issue`: Delete an issue
- `jira_search_jql`: Search issues using JQL
- `jira_transition_issue`: Transition an issue to a new status
- `jira_add_comment`: Add a comment to an issue
- `jira_list_projects`: List all accessible projects

### Confluence Operations

- `confluence_get_page`: Retrieve a page by ID
- `confluence_create_page`: Create a new page
- `confluence_update_page`: Update an existing page
- `confluence_delete_page`: Delete a page
- `confluence_search_cql`: Search content using CQL
- `confluence_get_spaces`: List all accessible spaces
- `confluence_get_page_history`: Get page version history

### Bitbucket Operations

- `bitbucket_get_repositories`: List repositories in a project
- `bitbucket_get_branches`: List branches in a repository
- `bitbucket_create_branch`: Create a new branch
- `bitbucket_get_pull_request`: Get pull request details
- `bitbucket_create_pull_request`: Create a new pull request
- `bitbucket_merge_pull_request`: Merge a pull request
- `bitbucket_get_commits`: Get commit history
- `bitbucket_get_file_content`: Retrieve file content

### Bamboo Operations

- `bamboo_get_plans`: List all build plans
- `bamboo_get_plan`: Get a specific build plan
- `bamboo_trigger_build`: Trigger a build
- `bamboo_get_build_result`: Get build result details
- `bamboo_get_build_log`: Retrieve build logs
- `bamboo_get_deployment_projects`: List deployment projects
- `bamboo_trigger_deployment`: Trigger a deployment

## MCP Protocol

The server implements the Model Context Protocol (MCP) using JSON-RPC 2.0 messaging. It supports the following MCP methods:

- `initialize`: Initial handshake between client and server
- `tools/list`: Discover available tools
- `tools/call`: Execute a tool operation

### Example Tool Call

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "jira_get_issue",
    "arguments": {
      "issueKey": "TEST-123"
    }
  }
}
```

## Graceful Shutdown

The server handles graceful shutdown on receiving:
- `SIGINT` (Ctrl+C)
- `SIGTERM`

All in-flight requests will be completed before shutdown.

## Error Handling

The server provides comprehensive error handling with appropriate JSON-RPC error codes:

- `-32700`: Parse error
- `-32600`: Invalid request
- `-32601`: Method not found
- `-32602`: Invalid params
- `-32603`: Internal error
- `-32001`: Configuration error
- `-32002`: Authentication error
- `-32003`: API error
- `-32004`: Network error
- `-32005`: Rate limit error

## Logging

The server logs important events to stdout:
- Configuration loading
- Component initialization
- Server startup/shutdown
- Request processing (via structured logger)
- Errors and warnings

## Development

### Project Structure

```
atlassian-mcp-server/
├── main.go                          # Entry point with dependency injection
├── internal/
│   ├── domain/                      # Domain layer (core business logic)
│   │   ├── config.go               # Configuration management
│   │   ├── auth.go                 # Authentication manager
│   │   ├── transport.go            # Transport interfaces and implementations
│   │   ├── response_mapper.go      # Response transformation
│   │   └── models.go               # Domain models
│   ├── application/                 # Application layer (use cases)
│   │   ├── server.go               # MCP server core
│   │   ├── router.go               # Request router
│   │   ├── jira_handler.go         # Jira operations handler
│   │   ├── confluence_handler.go   # Confluence operations handler
│   │   ├── bitbucket_handler.go    # Bitbucket operations handler
│   │   └── bamboo_handler.go       # Bamboo operations handler
│   └── infrastructure/              # Infrastructure layer (external dependencies)
│       ├── jira_client.go          # Jira API client
│       ├── confluence_client.go    # Confluence API client
│       ├── bitbucket_client.go     # Bitbucket API client
│       └── bamboo_client.go        # Bamboo API client
└── config.yaml                      # Configuration file
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run property-based tests
go test -v ./internal/...
```

## License

[Add your license information here]

## Contributing

[Add contribution guidelines here]

## Support

For issues and questions, please [add support information here].
