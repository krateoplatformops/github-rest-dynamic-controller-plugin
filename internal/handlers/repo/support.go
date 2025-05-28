package repo

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/krateoplatformops/github-rest-dynamic-controller-plugin/internal/utils"
)

// NewResponseFlattener creates a new flattener with field mappings
func NewResponseFlattener(mappings []utils.FieldMapping) *utils.ResponseFlattener {
	return &utils.ResponseFlattener{Mappings: mappings}
}

// GitHubUserPermissionFlattener for GitHub user permission responses
// Preserves all root fields and brings nested user fields to root
var GitHubUserPermissionFlattener = NewResponseFlattener([]utils.FieldMapping{
	{SourcePath: "user.permissions", TargetKey: "permissions"},
	{SourcePath: "user.html_url", TargetKey: "html_url"},
	{SourcePath: "user.id", TargetKey: "id"},
})

// Convenience function for GitHub user permission responses
func FlattenGitHubUserPermission(resp *http.Response) ([]byte, error) {
	return GitHubUserPermissionFlattener.FlattenResponse(resp)
}

// FlattenGitHubUserPermissionBytes flattens GitHub permission response from bytes
func FlattenGitHubUserPermissionBytes(body []byte) ([]byte, error) {
	return GitHubUserPermissionFlattener.FlattenBytes(body)
}

// The following are utility functions to deal with GitHub permissions discrepancies in the API responses

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
