package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// FieldMapping defines how to extract and rename fields
type FieldMapping struct {
	SourcePath string // e.g., "user.permissions", "user.html_url", "user.id"
	TargetKey  string // e.g., "permissions", "html_url", "id"
}

// ResponseFlattener handles flattening of HTTP response bodies
type ResponseFlattener struct {
	mappings []FieldMapping
}

// NewResponseFlattener creates a new flattener with field mappings
func NewResponseFlattener(mappings []FieldMapping) *ResponseFlattener {
	return &ResponseFlattener{mappings: mappings}
}

// FlattenResponse reads and flattens an HTTP response body
func (rf *ResponseFlattener) FlattenResponse(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return rf.FlattenBytes(body)
}

// FlattenBytes flattens a JSON byte array
func (rf *ResponseFlattener) FlattenBytes(body []byte) ([]byte, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Start with the complete original data to preserve everything
	flattened := make(map[string]interface{})

	// Copy the entire original response
	for key, value := range data {
		flattened[key] = value
	}

	// Then, add the flattened nested fields to root level
	for _, mapping := range rf.mappings {
		value, err := rf.extractValue(data, mapping.SourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to extract %s: %w", mapping.SourcePath, err)
		}
		// Add/override at root level
		flattened[mapping.TargetKey] = value
	}

	return json.Marshal(flattened)
}

// extractValue extracts a value from nested map using dot notation path
func (rf *ResponseFlattener) extractValue(data map[string]interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - return the value
			value, exists := current[part]
			if !exists {
				return nil, fmt.Errorf("field %s not found", part)
			}
			return value, nil
		}

		// Intermediate part - navigate deeper
		next, exists := current[part]
		if !exists {
			return nil, fmt.Errorf("field %s not found in path %s", part, path)
		}

		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("field %s is not an object in path %s", part, path)
		}
		current = nextMap
	}

	return nil, fmt.Errorf("invalid path %s", path)
}

var (
	// GitHubUserPermissionFlattener for GitHub user permission responses
	// Preserves all root fields and brings nested user fields to root
	GitHubUserPermissionFlattener = NewResponseFlattener([]FieldMapping{
		{SourcePath: "user.permissions", TargetKey: "permissions"},
		{SourcePath: "user.html_url", TargetKey: "html_url"},
		{SourcePath: "user.id", TargetKey: "id"},
	})
)

// Convenience function for GitHub user permission responses
func FlattenGitHubUserPermission(resp *http.Response) ([]byte, error) {
	return GitHubUserPermissionFlattener.FlattenResponse(resp)
}

// FlattenGitHubUserPermissionBytes flattens GitHub permission response from bytes
func FlattenGitHubUserPermissionBytes(body []byte) ([]byte, error) {
	return GitHubUserPermissionFlattener.FlattenBytes(body)
}

// utility function to deal with GitHub permissions discrepancies in the API responses

// GitHubPermissions represents the permissions object from GitHub API
type GitHubPermissions struct {
	Admin    bool `json:"admin"`
	Maintain bool `json:"maintain"`
	Pull     bool `json:"pull"`
	Push     bool `json:"push"`
	Triage   bool `json:"triage"`
}

// DetermineCorrectPermission determines the correct permission level based on GitHub permissions
func DetermineCorrectPermission(perms GitHubPermissions) string {
	// Check in order of precedence: admin > maintain > push > triage > pull
	if perms.Admin {
		return "admin"
	}
	if perms.Maintain {
		return "maintain"
	}
	if perms.Push {
		return "push"
	}
	if perms.Triage {
		return "triage"
	}
	if perms.Pull {
		return "pull"
	}
	return "none" // fallback case
}

// CorrectGitHubPermissionField corrects the permission field in GitHub API responses
func CorrectGitHubUserPermissionField(body []byte) ([]byte, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Extract permissions object
	permissionsInterface, exists := data["permissions"]
	if !exists {
		return body, nil // No permissions field to correct
	}

	// Convert permissions to our struct
	permissionsBytes, err := json.Marshal(permissionsInterface)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal permissions: %w", err)
	}

	var permissions GitHubPermissions
	if err := json.Unmarshal(permissionsBytes, &permissions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal permissions: %w", err)
	}

	// Determine correct permission and update
	correctPermission := DetermineCorrectPermission(permissions)
	data["permission"] = correctPermission

	return json.Marshal(data)
}
