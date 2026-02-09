package domain

import "encoding/json"

// ConfluencePage represents a Confluence page with all its fields.
// This is the main entity returned by Confluence API operations.
type ConfluencePage struct {
	ID      json.Number `json:"id"`
	Type    string      `json:"type"`
	Title   string      `json:"title"`
	Space   Space       `json:"space"`
	Body    Body        `json:"body"`
	Version Version     `json:"version"`
}

// Space represents a Confluence space.
type Space struct {
	ID   json.Number `json:"id"`
	Key  string      `json:"key"`
	Name string      `json:"name"`
}

// Body represents the body content of a Confluence page.
type Body struct {
	Storage Storage `json:"storage"`
}

// Storage represents the storage format of page content.
type Storage struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

// Version represents the version information of a Confluence page.
type Version struct {
	Number int    `json:"number"`
	When   string `json:"when"`
	By     User   `json:"by"`
}

// PageHistory represents the history information of a Confluence page.
type PageHistory struct {
	Latest      bool        `json:"latest"`
	CreatedBy   User        `json:"createdBy"`
	CreatedDate string      `json:"createdDate"`
	LastUpdated LastUpdated `json:"lastUpdated"`
}

// LastUpdated represents the last update information of a page.
type LastUpdated struct {
	By   User   `json:"by"`
	When string `json:"when"`
}

// PageCreate represents the request body for creating a new Confluence page.
type PageCreate struct {
	Type  string     `json:"type"`
	Title string     `json:"title"`
	Space SpaceRef   `json:"space"`
	Body  BodyCreate `json:"body"`
}

// SpaceRef is a reference to a space (used in create/update operations).
type SpaceRef struct {
	Key string `json:"key"`
}

// BodyCreate represents the body content for creating a page.
type BodyCreate struct {
	Storage StorageCreate `json:"storage"`
}

// StorageCreate represents the storage format for creating page content.
type StorageCreate struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

// PageUpdate represents the request body for updating a Confluence page.
type PageUpdate struct {
	Version VersionUpdate `json:"version"`
	Title   string        `json:"title,omitempty"`
	Type    string        `json:"type,omitempty"`
	Body    *BodyCreate   `json:"body,omitempty"`
}

// VersionUpdate represents the version information for updating a page.
type VersionUpdate struct {
	Number int `json:"number"`
}
