package domain

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestToolDefinitionJSONSerialization tests that ToolDefinition marshals correctly to JSON.
func TestToolDefinitionJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		toolDef  ToolDefinition
		wantJSON string
	}{
		{
			name: "complete tool definition",
			toolDef: ToolDefinition{
				Name:        "jira_get_issue",
				Description: "Get a Jira issue by key",
				InputSchema: JSONSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"issueKey": map[string]interface{}{
							"type":        "string",
							"description": "The issue key (e.g., PROJ-123)",
						},
					},
					Required: []string{"issueKey"},
				},
			},
			wantJSON: `{"name":"jira_get_issue","description":"Get a Jira issue by key","inputSchema":{"type":"object","properties":{"issueKey":{"description":"The issue key (e.g., PROJ-123)","type":"string"}},"required":["issueKey"]}}`,
		},
		{
			name: "tool definition without required fields",
			toolDef: ToolDefinition{
				Name:        "simple_tool",
				Description: "A simple tool",
				InputSchema: JSONSchema{
					Type: "object",
				},
			},
			wantJSON: `{"name":"simple_tool","description":"A simple tool","inputSchema":{"type":"object"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.toolDef)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			// Compare JSON strings (order-independent comparison would be better, but this works for our tests)
			var gotMap, wantMap map[string]interface{}
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Fatalf("json.Unmarshal(got) error = %v", err)
			}
			if err := json.Unmarshal([]byte(tt.wantJSON), &wantMap); err != nil {
				t.Fatalf("json.Unmarshal(want) error = %v", err)
			}

			if !reflect.DeepEqual(gotMap, wantMap) {
				t.Errorf("json.Marshal() = %s, want %s", string(got), tt.wantJSON)
			}
		})
	}
}

// TestToolRequestJSONSerialization tests that ToolRequest marshals correctly to JSON.
func TestToolRequestJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		request  ToolRequest
		wantJSON string
	}{
		{
			name: "request with arguments",
			request: ToolRequest{
				Name: "jira_get_issue",
				Arguments: map[string]interface{}{
					"issueKey": "PROJ-123",
				},
			},
			wantJSON: `{"name":"jira_get_issue","arguments":{"issueKey":"PROJ-123"}}`,
		},
		{
			name: "request with empty arguments",
			request: ToolRequest{
				Name:      "jira_list_projects",
				Arguments: map[string]interface{}{},
			},
			wantJSON: `{"name":"jira_list_projects","arguments":{}}`,
		},
		{
			name: "request with complex arguments",
			request: ToolRequest{
				Name: "jira_create_issue",
				Arguments: map[string]interface{}{
					"project": "PROJ",
					"summary": "Test issue",
					"fields": map[string]interface{}{
						"priority": "High",
						"labels":   []string{"bug", "urgent"},
					},
				},
			},
			wantJSON: `{"name":"jira_create_issue","arguments":{"project":"PROJ","summary":"Test issue","fields":{"priority":"High","labels":["bug","urgent"]}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			var gotMap, wantMap map[string]interface{}
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Fatalf("json.Unmarshal(got) error = %v", err)
			}
			if err := json.Unmarshal([]byte(tt.wantJSON), &wantMap); err != nil {
				t.Fatalf("json.Unmarshal(want) error = %v", err)
			}

			if !reflect.DeepEqual(gotMap, wantMap) {
				t.Errorf("json.Marshal() = %s, want %s", string(got), tt.wantJSON)
			}
		})
	}
}

// TestToolResponseJSONSerialization tests that ToolResponse marshals correctly to JSON.
func TestToolResponseJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		response ToolResponse
		wantJSON string
	}{
		{
			name: "response with text content",
			response: ToolResponse{
				Content: []ContentBlock{
					{
						Type: "text",
						Text: "Issue PROJ-123 retrieved successfully",
					},
				},
			},
			wantJSON: `{"content":[{"type":"text","text":"Issue PROJ-123 retrieved successfully"}]}`,
		},
		{
			name: "response with resource content",
			response: ToolResponse{
				Content: []ContentBlock{
					{
						Type: "resource",
						Resource: &Resource{
							URI:      "jira://issue/PROJ-123",
							MimeType: "application/json",
							Text:     `{"key":"PROJ-123","summary":"Test issue"}`,
						},
					},
				},
			},
			wantJSON: `{"content":[{"type":"resource","resource":{"uri":"jira://issue/PROJ-123","mimeType":"application/json","text":"{\"key\":\"PROJ-123\",\"summary\":\"Test issue\"}"}}]}`,
		},
		{
			name: "response with multiple content blocks",
			response: ToolResponse{
				Content: []ContentBlock{
					{
						Type: "text",
						Text: "Found 2 issues:",
					},
					{
						Type: "text",
						Text: "PROJ-123: First issue",
					},
					{
						Type: "text",
						Text: "PROJ-124: Second issue",
					},
				},
			},
			wantJSON: `{"content":[{"type":"text","text":"Found 2 issues:"},{"type":"text","text":"PROJ-123: First issue"},{"type":"text","text":"PROJ-124: Second issue"}]}`,
		},
		{
			name: "error response",
			response: ToolResponse{
				Content: []ContentBlock{
					{
						Type: "text",
						Text: "Error: Issue not found",
					},
				},
				IsError: true,
			},
			wantJSON: `{"content":[{"type":"text","text":"Error: Issue not found"}],"isError":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.response)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			var gotMap, wantMap map[string]interface{}
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Fatalf("json.Unmarshal(got) error = %v", err)
			}
			if err := json.Unmarshal([]byte(tt.wantJSON), &wantMap); err != nil {
				t.Fatalf("json.Unmarshal(want) error = %v", err)
			}

			if !reflect.DeepEqual(gotMap, wantMap) {
				t.Errorf("json.Marshal() = %s, want %s", string(got), tt.wantJSON)
			}
		})
	}
}

// TestContentBlockJSONSerialization tests that ContentBlock marshals correctly to JSON.
func TestContentBlockJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		block    ContentBlock
		wantJSON string
	}{
		{
			name: "text content block",
			block: ContentBlock{
				Type: "text",
				Text: "Hello, world!",
			},
			wantJSON: `{"type":"text","text":"Hello, world!"}`,
		},
		{
			name: "resource content block",
			block: ContentBlock{
				Type: "resource",
				Resource: &Resource{
					URI:      "https://example.com/resource",
					MimeType: "application/json",
					Text:     `{"data":"value"}`,
				},
			},
			wantJSON: `{"type":"resource","resource":{"uri":"https://example.com/resource","mimeType":"application/json","text":"{\"data\":\"value\"}"}}`,
		},
		{
			name: "resource without mime type",
			block: ContentBlock{
				Type: "resource",
				Resource: &Resource{
					URI:  "https://example.com/resource",
					Text: "plain text",
				},
			},
			wantJSON: `{"type":"resource","resource":{"uri":"https://example.com/resource","text":"plain text"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.block)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			var gotMap, wantMap map[string]interface{}
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Fatalf("json.Unmarshal(got) error = %v", err)
			}
			if err := json.Unmarshal([]byte(tt.wantJSON), &wantMap); err != nil {
				t.Fatalf("json.Unmarshal(want) error = %v", err)
			}

			if !reflect.DeepEqual(gotMap, wantMap) {
				t.Errorf("json.Marshal() = %s, want %s", string(got), tt.wantJSON)
			}
		})
	}
}

// TestJSONSchemaJSONSerialization tests that JSONSchema marshals correctly to JSON.
func TestJSONSchemaJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		schema   JSONSchema
		wantJSON string
	}{
		{
			name: "simple object schema",
			schema: JSONSchema{
				Type: "object",
			},
			wantJSON: `{"type":"object"}`,
		},
		{
			name: "schema with properties",
			schema: JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "The name field",
					},
					"age": map[string]interface{}{
						"type":    "integer",
						"minimum": 0,
					},
				},
			},
			wantJSON: `{"type":"object","properties":{"name":{"type":"string","description":"The name field"},"age":{"type":"integer","minimum":0}}}`,
		},
		{
			name: "schema with required fields",
			schema: JSONSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"id": map[string]interface{}{
						"type": "string",
					},
				},
				Required: []string{"id"},
			},
			wantJSON: `{"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.schema)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			var gotMap, wantMap map[string]interface{}
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Fatalf("json.Unmarshal(got) error = %v", err)
			}
			if err := json.Unmarshal([]byte(tt.wantJSON), &wantMap); err != nil {
				t.Fatalf("json.Unmarshal(want) error = %v", err)
			}

			if !reflect.DeepEqual(gotMap, wantMap) {
				t.Errorf("json.Marshal() = %s, want %s", string(got), tt.wantJSON)
			}
		})
	}
}

// TestToolDefinitionDeserialization tests that ToolDefinition unmarshals correctly from JSON.
func TestToolDefinitionDeserialization(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    ToolDefinition
		wantErr bool
	}{
		{
			name: "valid tool definition",
			json: `{"name":"test_tool","description":"A test tool","inputSchema":{"type":"object","required":["param1"]}}`,
			want: ToolDefinition{
				Name:        "test_tool",
				Description: "A test tool",
				InputSchema: JSONSchema{
					Type:     "object",
					Required: []string{"param1"},
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			json:    `{"name":"test_tool","description":}`,
			want:    ToolDefinition{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got ToolDefinition
			err := json.Unmarshal([]byte(tt.json), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("json.Unmarshal() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestToolRequestDeserialization tests that ToolRequest unmarshals correctly from JSON.
func TestToolRequestDeserialization(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    ToolRequest
		wantErr bool
	}{
		{
			name: "valid tool request",
			json: `{"name":"jira_get_issue","arguments":{"issueKey":"PROJ-123"}}`,
			want: ToolRequest{
				Name: "jira_get_issue",
				Arguments: map[string]interface{}{
					"issueKey": "PROJ-123",
				},
			},
			wantErr: false,
		},
		{
			name: "request with empty arguments",
			json: `{"name":"list_tools","arguments":{}}`,
			want: ToolRequest{
				Name:      "list_tools",
				Arguments: map[string]interface{}{},
			},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			json:    `{"name":"test","arguments":}`,
			want:    ToolRequest{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got ToolRequest
			err := json.Unmarshal([]byte(tt.json), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("json.Unmarshal() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestToolResponseDeserialization tests that ToolResponse unmarshals correctly from JSON.
func TestToolResponseDeserialization(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    ToolResponse
		wantErr bool
	}{
		{
			name: "valid response with text content",
			json: `{"content":[{"type":"text","text":"Success"}]}`,
			want: ToolResponse{
				Content: []ContentBlock{
					{
						Type: "text",
						Text: "Success",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "error response",
			json: `{"content":[{"type":"text","text":"Error occurred"}],"isError":true}`,
			want: ToolResponse{
				Content: []ContentBlock{
					{
						Type: "text",
						Text: "Error occurred",
					},
				},
				IsError: true,
			},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			json:    `{"content":}`,
			want:    ToolResponse{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got ToolResponse
			err := json.Unmarshal([]byte(tt.json), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("json.Unmarshal() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestMCPStructTags verifies that MCP structs have correct JSON tags.
func TestMCPStructTags(t *testing.T) {
	// Test ToolDefinition
	toolDef := ToolDefinition{
		Name:        "test",
		Description: "desc",
		InputSchema: JSONSchema{Type: "object"},
	}
	data, _ := json.Marshal(toolDef)
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if _, ok := result["name"]; !ok {
		t.Error("ToolDefinition missing 'name' field in JSON")
	}
	if _, ok := result["description"]; !ok {
		t.Error("ToolDefinition missing 'description' field in JSON")
	}
	if _, ok := result["inputSchema"]; !ok {
		t.Error("ToolDefinition missing 'inputSchema' field in JSON")
	}

	// Test ToolRequest
	toolReq := ToolRequest{
		Name:      "test",
		Arguments: map[string]interface{}{"key": "value"},
	}
	data, _ = json.Marshal(toolReq)
	json.Unmarshal(data, &result)

	if _, ok := result["name"]; !ok {
		t.Error("ToolRequest missing 'name' field in JSON")
	}
	if _, ok := result["arguments"]; !ok {
		t.Error("ToolRequest missing 'arguments' field in JSON")
	}

	// Test ToolResponse
	toolResp := ToolResponse{
		Content: []ContentBlock{{Type: "text", Text: "test"}},
	}
	data, _ = json.Marshal(toolResp)
	json.Unmarshal(data, &result)

	if _, ok := result["content"]; !ok {
		t.Error("ToolResponse missing 'content' field in JSON")
	}

	// Test ContentBlock
	block := ContentBlock{
		Type: "text",
		Text: "test",
	}
	data, _ = json.Marshal(block)
	json.Unmarshal(data, &result)

	if _, ok := result["type"]; !ok {
		t.Error("ContentBlock missing 'type' field in JSON")
	}
	if _, ok := result["text"]; !ok {
		t.Error("ContentBlock missing 'text' field in JSON")
	}

	// Test JSONSchema
	schema := JSONSchema{
		Type:     "object",
		Required: []string{"field1"},
	}
	data, _ = json.Marshal(schema)
	json.Unmarshal(data, &result)

	if _, ok := result["type"]; !ok {
		t.Error("JSONSchema missing 'type' field in JSON")
	}
	if _, ok := result["required"]; !ok {
		t.Error("JSONSchema missing 'required' field in JSON")
	}
}

// TestOmitEmptyBehaviorMCP verifies that omitempty works correctly for MCP structs.
func TestOmitEmptyBehaviorMCP(t *testing.T) {
	// Test ToolResponse without IsError
	resp := ToolResponse{
		Content: []ContentBlock{{Type: "text", Text: "test"}},
	}
	data, _ := json.Marshal(resp)
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if _, ok := result["isError"]; ok {
		t.Error("ToolResponse should omit 'isError' when false")
	}

	// Test ToolResponse with IsError
	resp.IsError = true
	data, _ = json.Marshal(resp)
	json.Unmarshal(data, &result)

	if _, ok := result["isError"]; !ok {
		t.Error("ToolResponse should include 'isError' when true")
	}

	// Test ContentBlock with only text
	block := ContentBlock{
		Type: "text",
		Text: "test",
	}
	data, _ = json.Marshal(block)
	json.Unmarshal(data, &result)

	if _, ok := result["resource"]; ok {
		t.Error("ContentBlock should omit 'resource' when nil")
	}

	// Test ContentBlock with resource
	block = ContentBlock{
		Type:     "resource",
		Resource: &Resource{URI: "test://uri"},
	}
	data, _ = json.Marshal(block)
	json.Unmarshal(data, &result)

	if _, ok := result["resource"]; !ok {
		t.Error("ContentBlock should include 'resource' when set")
	}

	// Test JSONSchema without properties
	schema := JSONSchema{
		Type: "object",
	}
	data, _ = json.Marshal(schema)
	json.Unmarshal(data, &result)

	if _, ok := result["properties"]; ok {
		t.Error("JSONSchema should omit 'properties' when nil")
	}
	if _, ok := result["required"]; ok {
		t.Error("JSONSchema should omit 'required' when nil")
	}
}
