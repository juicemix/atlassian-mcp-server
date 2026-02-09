package domain

import (
	"encoding/json"
	"testing"
)

func TestConfluencePageJSONSerialization(t *testing.T) {
	page := ConfluencePage{
		ID:    "12345",
		Type:  "page",
		Title: "Test Page",
		Space: Space{
			ID:   "1",
			Key:  "TEST",
			Name: "Test Space",
		},
		Body: Body{
			Storage: Storage{
				Value:          "<p>This is test content</p>",
				Representation: "storage",
			},
		},
		Version: Version{
			Number: 1,
			When:   "2024-01-01T10:00:00.000Z",
			By: User{
				Name:         "jsmith",
				DisplayName:  "John Smith",
				EmailAddress: "jsmith@example.com",
			},
		},
	}

	// Test serialization
	data, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("Failed to marshal ConfluencePage: %v", err)
	}

	// Test deserialization
	var decoded ConfluencePage
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal ConfluencePage: %v", err)
	}

	// Verify key fields
	if decoded.ID != page.ID {
		t.Errorf("Expected ID %s, got %s", page.ID, decoded.ID)
	}
	if decoded.Type != page.Type {
		t.Errorf("Expected Type %s, got %s", page.Type, decoded.Type)
	}
	if decoded.Title != page.Title {
		t.Errorf("Expected Title %s, got %s", page.Title, decoded.Title)
	}
	if decoded.Space.Key != page.Space.Key {
		t.Errorf("Expected Space.Key %s, got %s", page.Space.Key, decoded.Space.Key)
	}
	if decoded.Body.Storage.Value != page.Body.Storage.Value {
		t.Errorf("Expected Body.Storage.Value %s, got %s", page.Body.Storage.Value, decoded.Body.Storage.Value)
	}
	if decoded.Version.Number != page.Version.Number {
		t.Errorf("Expected Version.Number %d, got %d", page.Version.Number, decoded.Version.Number)
	}
	if decoded.Version.By.Name != page.Version.By.Name {
		t.Errorf("Expected Version.By.Name %s, got %s", page.Version.By.Name, decoded.Version.By.Name)
	}
}

func TestSpaceJSONSerialization(t *testing.T) {
	space := Space{
		ID:   "1",
		Key:  "TEST",
		Name: "Test Space",
	}

	// Test serialization
	data, err := json.Marshal(space)
	if err != nil {
		t.Fatalf("Failed to marshal Space: %v", err)
	}

	// Test deserialization
	var decoded Space
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal Space: %v", err)
	}

	// Verify fields
	if decoded.ID != space.ID {
		t.Errorf("Expected ID %s, got %s", space.ID, decoded.ID)
	}
	if decoded.Key != space.Key {
		t.Errorf("Expected Key %s, got %s", space.Key, decoded.Key)
	}
	if decoded.Name != space.Name {
		t.Errorf("Expected Name %s, got %s", space.Name, decoded.Name)
	}
}

func TestBodyJSONSerialization(t *testing.T) {
	body := Body{
		Storage: Storage{
			Value:          "<p>Test content</p>",
			Representation: "storage",
		},
	}

	// Test serialization
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal Body: %v", err)
	}

	// Test deserialization
	var decoded Body
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal Body: %v", err)
	}

	// Verify fields
	if decoded.Storage.Value != body.Storage.Value {
		t.Errorf("Expected Storage.Value %s, got %s", body.Storage.Value, decoded.Storage.Value)
	}
	if decoded.Storage.Representation != body.Storage.Representation {
		t.Errorf("Expected Storage.Representation %s, got %s", body.Storage.Representation, decoded.Storage.Representation)
	}
}

func TestVersionJSONSerialization(t *testing.T) {
	version := Version{
		Number: 5,
		When:   "2024-01-15T14:30:00.000Z",
		By: User{
			Name:         "jdoe",
			DisplayName:  "Jane Doe",
			EmailAddress: "jdoe@example.com",
		},
	}

	// Test serialization
	data, err := json.Marshal(version)
	if err != nil {
		t.Fatalf("Failed to marshal Version: %v", err)
	}

	// Test deserialization
	var decoded Version
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal Version: %v", err)
	}

	// Verify fields
	if decoded.Number != version.Number {
		t.Errorf("Expected Number %d, got %d", version.Number, decoded.Number)
	}
	if decoded.When != version.When {
		t.Errorf("Expected When %s, got %s", version.When, decoded.When)
	}
	if decoded.By.Name != version.By.Name {
		t.Errorf("Expected By.Name %s, got %s", version.By.Name, decoded.By.Name)
	}
}

func TestPageHistoryJSONSerialization(t *testing.T) {
	history := PageHistory{
		Latest: true,
		CreatedBy: User{
			Name:         "jsmith",
			DisplayName:  "John Smith",
			EmailAddress: "jsmith@example.com",
		},
		CreatedDate: "2024-01-01T10:00:00.000Z",
		LastUpdated: LastUpdated{
			By: User{
				Name:         "jdoe",
				DisplayName:  "Jane Doe",
				EmailAddress: "jdoe@example.com",
			},
			When: "2024-01-15T14:30:00.000Z",
		},
	}

	// Test serialization
	data, err := json.Marshal(history)
	if err != nil {
		t.Fatalf("Failed to marshal PageHistory: %v", err)
	}

	// Test deserialization
	var decoded PageHistory
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal PageHistory: %v", err)
	}

	// Verify fields
	if decoded.Latest != history.Latest {
		t.Errorf("Expected Latest %v, got %v", history.Latest, decoded.Latest)
	}
	if decoded.CreatedBy.Name != history.CreatedBy.Name {
		t.Errorf("Expected CreatedBy.Name %s, got %s", history.CreatedBy.Name, decoded.CreatedBy.Name)
	}
	if decoded.CreatedDate != history.CreatedDate {
		t.Errorf("Expected CreatedDate %s, got %s", history.CreatedDate, decoded.CreatedDate)
	}
	if decoded.LastUpdated.By.Name != history.LastUpdated.By.Name {
		t.Errorf("Expected LastUpdated.By.Name %s, got %s", history.LastUpdated.By.Name, decoded.LastUpdated.By.Name)
	}
	if decoded.LastUpdated.When != history.LastUpdated.When {
		t.Errorf("Expected LastUpdated.When %s, got %s", history.LastUpdated.When, decoded.LastUpdated.When)
	}
}

func TestPageCreateJSONSerialization(t *testing.T) {
	pageCreate := PageCreate{
		Type:  "page",
		Title: "New Test Page",
		Space: SpaceRef{
			Key: "TEST",
		},
		Body: BodyCreate{
			Storage: StorageCreate{
				Value:          "<p>New page content</p>",
				Representation: "storage",
			},
		},
	}

	// Test serialization
	data, err := json.Marshal(pageCreate)
	if err != nil {
		t.Fatalf("Failed to marshal PageCreate: %v", err)
	}

	// Test deserialization
	var decoded PageCreate
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal PageCreate: %v", err)
	}

	// Verify fields
	if decoded.Type != pageCreate.Type {
		t.Errorf("Expected Type %s, got %s", pageCreate.Type, decoded.Type)
	}
	if decoded.Title != pageCreate.Title {
		t.Errorf("Expected Title %s, got %s", pageCreate.Title, decoded.Title)
	}
	if decoded.Space.Key != pageCreate.Space.Key {
		t.Errorf("Expected Space.Key %s, got %s", pageCreate.Space.Key, decoded.Space.Key)
	}
	if decoded.Body.Storage.Value != pageCreate.Body.Storage.Value {
		t.Errorf("Expected Body.Storage.Value %s, got %s", pageCreate.Body.Storage.Value, decoded.Body.Storage.Value)
	}
	if decoded.Body.Storage.Representation != pageCreate.Body.Storage.Representation {
		t.Errorf("Expected Body.Storage.Representation %s, got %s", pageCreate.Body.Storage.Representation, decoded.Body.Storage.Representation)
	}
}

func TestPageUpdateJSONSerialization(t *testing.T) {
	pageUpdate := PageUpdate{
		Version: VersionUpdate{
			Number: 2,
		},
		Title: "Updated Page Title",
		Type:  "page",
		Body: &BodyCreate{
			Storage: StorageCreate{
				Value:          "<p>Updated content</p>",
				Representation: "storage",
			},
		},
	}

	// Test serialization
	data, err := json.Marshal(pageUpdate)
	if err != nil {
		t.Fatalf("Failed to marshal PageUpdate: %v", err)
	}

	// Test deserialization
	var decoded PageUpdate
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal PageUpdate: %v", err)
	}

	// Verify fields
	if decoded.Version.Number != pageUpdate.Version.Number {
		t.Errorf("Expected Version.Number %d, got %d", pageUpdate.Version.Number, decoded.Version.Number)
	}
	if decoded.Title != pageUpdate.Title {
		t.Errorf("Expected Title %s, got %s", pageUpdate.Title, decoded.Title)
	}
	if decoded.Type != pageUpdate.Type {
		t.Errorf("Expected Type %s, got %s", pageUpdate.Type, decoded.Type)
	}
	if decoded.Body == nil {
		t.Fatal("Expected Body to be non-nil")
	}
	if decoded.Body.Storage.Value != pageUpdate.Body.Storage.Value {
		t.Errorf("Expected Body.Storage.Value %s, got %s", pageUpdate.Body.Storage.Value, decoded.Body.Storage.Value)
	}
}

func TestPageUpdateWithNilBody(t *testing.T) {
	// Test that Body can be nil in PageUpdate
	pageUpdate := PageUpdate{
		Version: VersionUpdate{
			Number: 3,
		},
		Title: "Only Title Update",
		Type:  "page",
		Body:  nil, // Optional field
	}

	// Test serialization
	data, err := json.Marshal(pageUpdate)
	if err != nil {
		t.Fatalf("Failed to marshal PageUpdate with nil Body: %v", err)
	}

	// Test deserialization
	var decoded PageUpdate
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal PageUpdate with nil Body: %v", err)
	}

	// Verify fields
	if decoded.Version.Number != pageUpdate.Version.Number {
		t.Errorf("Expected Version.Number %d, got %d", pageUpdate.Version.Number, decoded.Version.Number)
	}
	if decoded.Title != pageUpdate.Title {
		t.Errorf("Expected Title %s, got %s", pageUpdate.Title, decoded.Title)
	}
	if decoded.Body != nil {
		t.Errorf("Expected Body to be nil, got %v", decoded.Body)
	}
}

func TestStorageWithSpecialCharacters(t *testing.T) {
	// Test that Storage can handle special characters and HTML
	storage := Storage{
		Value:          "<p>Test with <strong>bold</strong> and &amp; special chars</p>",
		Representation: "storage",
	}

	// Test serialization
	data, err := json.Marshal(storage)
	if err != nil {
		t.Fatalf("Failed to marshal Storage with special characters: %v", err)
	}

	// Test deserialization
	var decoded Storage
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal Storage with special characters: %v", err)
	}

	// Verify fields
	if decoded.Value != storage.Value {
		t.Errorf("Expected Value %s, got %s", storage.Value, decoded.Value)
	}
	if decoded.Representation != storage.Representation {
		t.Errorf("Expected Representation %s, got %s", storage.Representation, decoded.Representation)
	}
}

func TestConfluencePageWithComplexContent(t *testing.T) {
	// Test a more complex page with nested structures
	page := ConfluencePage{
		ID:    "67890",
		Type:  "page",
		Title: "Complex Page with Special Characters: <>&\"'",
		Space: Space{
			ID:   "2",
			Key:  "DOCS",
			Name: "Documentation Space",
		},
		Body: Body{
			Storage: Storage{
				Value: `<h1>Heading</h1>
<p>Paragraph with <strong>bold</strong> and <em>italic</em></p>
<ul>
  <li>Item 1</li>
  <li>Item 2</li>
</ul>
<ac:structured-macro ac:name="code">
  <ac:plain-text-body><![CDATA[code block]]></ac:plain-text-body>
</ac:structured-macro>`,
				Representation: "storage",
			},
		},
		Version: Version{
			Number: 10,
			When:   "2024-02-20T16:45:30.123Z",
			By: User{
				Name:         "admin",
				DisplayName:  "Administrator",
				EmailAddress: "admin@example.com",
			},
		},
	}

	// Test serialization
	data, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("Failed to marshal complex ConfluencePage: %v", err)
	}

	// Test deserialization
	var decoded ConfluencePage
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal complex ConfluencePage: %v", err)
	}

	// Verify key fields
	if decoded.ID != page.ID {
		t.Errorf("Expected ID %s, got %s", page.ID, decoded.ID)
	}
	if decoded.Title != page.Title {
		t.Errorf("Expected Title %s, got %s", page.Title, decoded.Title)
	}
	if decoded.Body.Storage.Value != page.Body.Storage.Value {
		t.Errorf("Expected Body.Storage.Value to match, got different content")
	}
	if decoded.Version.Number != page.Version.Number {
		t.Errorf("Expected Version.Number %d, got %d", page.Version.Number, decoded.Version.Number)
	}
}
