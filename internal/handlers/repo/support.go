package repo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	teamrepo "github.com/krateoplatformops/github-rest-dynamic-controller-plugin/internal/handlers/teamRepo"
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

// Invitations handling
// This section deals with GitHub repository invitations, allowing us to check if a user has been invited to collaborate on a repository.
// It includes parsing the invitation response, checking if a user exists in the invitations, and building a response similar to the collaborator permission response.

// GitHubInvitation represents a GitHub repository invitation
type GitHubInvitation struct {
	ID         int64               `json:"id"`
	NodeID     string              `json:"node_id"`
	Repository teamrepo.Repository `json:"repository"`
	Invitee    struct {
		Login     string `json:"login"`
		ID        int64  `json:"id"`
		NodeID    string `json:"node_id"`
		AvatarURL string `json:"avatar_url"`
		HTMLURL   string `json:"html_url"`
		Type      string `json:"type"`
	} `json:"invitee"`
	Inviter struct {
		Login     string `json:"login"`
		ID        int64  `json:"id"`
		NodeID    string `json:"node_id"`
		AvatarURL string `json:"avatar_url"`
		HTMLURL   string `json:"html_url"`
		Type      string `json:"type"`
	} `json:"inviter"`
	Permissions string `json:"permissions"`
	CreatedAt   string `json:"created_at"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	Expired     bool   `json:"expired"`
}

// parseInvitations parses invitation response body into slice of GitHubInvitation
func parseInvitations(inviteBody []byte) ([]GitHubInvitation, error) {
	var invitations []GitHubInvitation
	if err := json.Unmarshal(inviteBody, &invitations); err != nil {
		return nil, err
	}
	return invitations, nil
}

// getUserInvitationFromPage checks if a username exists in a page of invitations
// returns the invitation (if found) along with a boolean indicating if it was found
// Here we could potentially check if the invitation is expired or not, if needed
func getUserInvitationFromPage(inviteBody []byte, username string) (*GitHubInvitation, bool) {
	invitations, err := parseInvitations(inviteBody)
	if err != nil {
		return nil, false
	}

	for _, invitation := range invitations {
		if strings.EqualFold(invitation.Invitee.Login, username) {
			return &invitation, true
		}
	}
	return nil, false
}

func createPermissionsObject(permission string) map[string]bool {
	permissions := map[string]bool{
		"admin":    false,
		"maintain": false,
		"push":     false,
		"triage":   false,
		"pull":     false,
	}

	switch permission {
	case "pull":
		permissions["pull"] = true
	case "push":
		permissions["pull"] = true
		permissions["push"] = true
	case "triage":
		permissions["pull"] = true
		permissions["triage"] = true
	case "maintain":
		permissions["pull"] = true
		permissions["push"] = true
		permissions["triage"] = true
		permissions["maintain"] = true
	case "admin":
		permissions["pull"] = true
		permissions["push"] = true
		permissions["triage"] = true
		permissions["maintain"] = true
		permissions["admin"] = true
	}

	return permissions
}

// BuildInvitationResponse builds a response similar to the collaborator permission response
// but with additional invitation status information
func BuildInvitationResponse(invitation *GitHubInvitation, username string) ([]byte, error) {
	// Map GitHub API permission to our expected format (same as CorrectGitHubUserPermissionField)
	// Note that in the case of invitations the `invitation.Permissions` field is the same as `role_name``: a single string with `read` and `write` permissions to be corrected to `pull` and `push` respectively, while `admin`, `maintain`, and `triage` remain unchanged.

	var correctedPermission string
	switch invitation.Permissions {
	case "read":
		correctedPermission = "pull"
	case "write":
		correctedPermission = "push"
	case "admin":
		correctedPermission = "admin"
	case "maintain":
		correctedPermission = "maintain"
	case "triage":
		correctedPermission = "triage"
	default:
		correctedPermission = invitation.Permissions
	}

	// Create permissions object
	permissionsObj := createPermissionsObject(correctedPermission)

	// Build user object similar to the collaborator response
	userObj := map[string]interface{}{
		"avatar_url":          invitation.Invitee.AvatarURL,
		"events_url":          fmt.Sprintf("https://api.github.com/users/%s/events{/privacy}", invitation.Invitee.Login),
		"followers_url":       fmt.Sprintf("https://api.github.com/users/%s/followers", invitation.Invitee.Login),
		"following_url":       fmt.Sprintf("https://api.github.com/users/%s/following{/other_user}", invitation.Invitee.Login),
		"gists_url":           fmt.Sprintf("https://api.github.com/users/%s/gists{/gist_id}", invitation.Invitee.Login),
		"gravatar_id":         "",
		"html_url":            invitation.Invitee.HTMLURL,
		"id":                  invitation.Invitee.ID,
		"login":               invitation.Invitee.Login,
		"node_id":             invitation.Invitee.NodeID,
		"organizations_url":   fmt.Sprintf("https://api.github.com/users/%s/orgs", invitation.Invitee.Login),
		"permissions":         permissionsObj,
		"received_events_url": fmt.Sprintf("https://api.github.com/users/%s/received_events", invitation.Invitee.Login),
		"repos_url":           fmt.Sprintf("https://api.github.com/users/%s/repos", invitation.Invitee.Login),
		"role_name":           invitation.Permissions, // Original permission from GitHub
		"site_admin":          false,
		"starred_url":         fmt.Sprintf("https://api.github.com/users/%s/starred{/owner}{/repo}", invitation.Invitee.Login),
		"subscriptions_url":   fmt.Sprintf("https://api.github.com/users/%s/subscriptions", invitation.Invitee.Login),
		"type":                invitation.Invitee.Type,
		"url":                 fmt.Sprintf("https://api.github.com/users/%s", invitation.Invitee.Login),
		"user_view_type":      "public",
	}

	// InviteeInfo object is an additional object in the response with partial information about the invitee
	inviteeInfoObj := map[string]interface{}{
		// Additional invitation-specific fields, useful for debug when checking collaborator-controller logs
		"invitation_status":   "pending",
		"invitation_id":       invitation.ID,
		"invitation_url":      invitation.URL,
		"invitation_html_url": invitation.HTMLURL,
		"invited_at":          invitation.CreatedAt,
		"invited_by":          invitation.Inviter.Login,
		"invitation_expired":  invitation.Expired,
	}

	// Build response structure similar to collaborator permission response
	response := map[string]interface{}{
		"html_url":     invitation.Invitee.HTMLURL,
		"id":           invitation.Invitee.ID,
		"permission":   correctedPermission,
		"permissions":  permissionsObj,
		"role_name":    invitation.Permissions, // Original permission from GitHub (single string)
		"user":         userObj,
		"invitee_info": inviteeInfoObj,
	}

	return json.Marshal(response)
}
