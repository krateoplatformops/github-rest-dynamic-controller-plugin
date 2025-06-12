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

// The following is a utility function to deal with GitHub permissions discrepancies in the API responses

/*
Discrepancies for a Collaborator:

'permission' in CR		'permission' in GitHub RESPONSE		'role_name' in GitHub RESPONSE
pull              		read                                read
push                  	write                               write
admin                 	admin                               admin
maintain              	write                               maintain
triage                	read                                triage
*/

// CorrectGitHubPermissionField corrects the permission field in GitHub API responses
func CorrectGitHubUserPermissionField(body []byte) ([]byte, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// change the permission filed based on the role_name
	// but if the role_name is read, then set the permission to pull
	// and if the role_name is write, then set the permission to push
	roleName, exists := data["role_name"].(string)
	if !exists {
		return body, nil // No roleName field to correct
	}
	var permission string
	switch roleName {
	case "read":
		permission = "pull"
	case "write":
		permission = "push"
	case "admin":
		permission = "admin"
	case "maintain":
		permission = "maintain"
	case "triage":
		permission = "triage"
	default:
		permission = roleName // Use the roleName as is if it doesn't match any known roles
	}
	if permission == "" {
		return body, nil // No permission to correct
	}

	// Update the permission field in the data map
	if data["permission"] == nil {
		data["permission"] = make(map[string]interface{})
	}
	data["permission"] = permission

	// Marshal the updated data back to JSON
	return json.Marshal(data)
}
