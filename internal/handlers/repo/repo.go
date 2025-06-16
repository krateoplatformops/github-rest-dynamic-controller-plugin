package repo

import (
	"fmt"
	"io"
	"net/http"

	"github.com/krateoplatformops/github-rest-dynamic-controller-plugin/internal/handlers"
)

func GetRepo(opts handlers.HandlerOptions) handlers.Handler {
	return &handler{
		HandlerOptions: opts,
	}
}

var _ handlers.Handler = &handler{}

type handler struct {
	handlers.HandlerOptions
}

// @Summary Get the permission of a user in a repository
// @Description Get the permission of a user in a repository
// @ID get-repo-permission
// @Param owner path string true "Owner of the repository"
// @Param repo path string true "Name of the repository"
// @Param username path string true "Username of the collaborator"
// @Produce json
// @Success 200 {object} repo.RepoPermissions
// @Router /repository/{owner}/{repo}/collaborators/{username}/permission [get]
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	username := r.PathValue("username")

	h.Log.Println("Calling GitHub Repository API")

	auth_header := r.Header.Get("Authorization")

	// 1. Check if user is a collaborator
	req, err := http.NewRequest("GET", "https://api.github.com/repos/"+owner+"/"+repo+"/collaborators/"+username, nil)
	if err != nil {
		h.Log.Println(err)
		w.Write([]byte(fmt.Sprint("Error: ", err)))
		return
	}

	if len(auth_header) > 0 {
		req.Header.Set("Authorization", auth_header)
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		h.Log.Print(err)
		w.Write([]byte(fmt.Sprint("Error: ", err)))
		return
	}

	// 2. If user is a collaborator of the repo (GitHub returns StatusNoContent (204))
	// Then we can get the permission
	if resp.StatusCode == http.StatusNoContent {
		h.Log.Print("User is a collaborator")
		req, err := http.NewRequest("GET", "https://api.github.com/repos/"+owner+"/"+repo+"/collaborators/"+username+"/permission", nil)
		if err != nil {
			h.Log.Print(err)
			w.Write([]byte(fmt.Sprint("Error: ", err)))
			return
		}

		if len(auth_header) > 0 {
			req.Header.Set("Authorization", auth_header)
		}

		resp, err = h.Client.Do(req)
		if err != nil {
			h.Log.Print(err)
			w.Write([]byte(fmt.Sprint("Error: ", err)))
			return
		}

		// Return 200 OK with permission data in the body

		// Read response body from GitHub API
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			h.Log.Print(err)
			w.Write([]byte(fmt.Sprint("Error: ", err)))
			return
		}

		// Flatten the response
		flattenedBody, err := FlattenGitHubUserPermissionBytes(body)
		if err != nil {
			h.Log.Print("Failed to flatten response:", err)
			h.Log.Print("Returning original response body")
			h.Log.Print("Original body:", string(body))
			w.Header().Set("Content-Type", "application/json")
			w.Write(body)
			return // Early return
		}

		// Correct the permission field due to GitHub API inconsistency
		correctedBody, err := CorrectGitHubUserPermissionField(flattenedBody)
		if err != nil {
			h.Log.Print("Failed to correct permission field:", err)
			h.Log.Print("Returning flattened response body")
			h.Log.Print("Flattened body:", string(flattenedBody))
			w.Header().Set("Content-Type", "application/json")
			w.Write(flattenedBody)
			return // Early return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(correctedBody)

		h.Log.Printf("Corrected body: %s", string(correctedBody))
		h.Log.Print("Successfully called", req.URL)
		return
	}

	// 3. If user is NOT a collaborator of the repo but an invitation to collaborate could exists
	// Check repository invitations with pagination
	invitation, found, err := h.findUserInvitation(owner, repo, username, auth_header)
	h.Log.Printf("Checking invitations for user %s in repository %s/%s", username, owner, repo)
	if err != nil {
		h.Log.Print("Error checking invitations:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprint("Error: ", err)))
		return
	}

	if found {
		h.Log.Printf("User %s has pending invitation with %s permission", username, invitation.Permissions)

		// Build response similar to collaborator case but with additional invitation details
		invitationResponse, err := BuildInvitationResponse(invitation, username)
		if err != nil {
			h.Log.Print("Failed to build invitation response:", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprint("Error: ", err)))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(invitationResponse)
		return
	}

	// Otherwise, if user is NOT a collaborator of the repo,
	// or user does not have received invitation,
	// or user does not exist
	// return the error status from GitHub (404 Not Found)
	h.Log.Println("User is not a collaborator of the repo OR does not have received invitation OR does not exist", resp.StatusCode, req.URL)
	w.WriteHeader(resp.StatusCode)
	w.Write([]byte(fmt.Sprint("Error: ", resp.Status)))
}

// findUserInvitation searches for a user invitation across all pages
func (h *handler) findUserInvitation(owner, repo, username, authHeader string) (*GitHubInvitation, bool, error) {

	h.Log.Printf("Checking invitations for user %s in repository %s/%s", username, owner, repo)

	page := 1
	perPage := 30

	for {
		req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/invitations?per_page=%d&page=%d", owner, repo, perPage, page), nil)
		if err != nil {
			return nil, false, err
		}

		if len(authHeader) > 0 {
			req.Header.Set("Authorization", authHeader)
		}

		inviteResp, err := h.Client.Do(req)
		if err != nil {
			return nil, false, err
		}
		defer inviteResp.Body.Close()

		// If we can't get invitations (not 200 OK), return not found
		if inviteResp.StatusCode != http.StatusOK {
			h.Log.Printf("Failed to get invitations, status: %d", inviteResp.StatusCode)
			return nil, false, nil
		}

		inviteBody, err := io.ReadAll(inviteResp.Body)
		if err != nil {
			return nil, false, err
		}

		// Check if username exists in current page of invitations
		if invitation, found := getUserInvitationFromPage(inviteBody, username); found {
			return invitation, true, nil
		}

		// Check if we have more pages
		// If we got less than perPage results, we've reached the last page
		invitations, err := parseInvitations(inviteBody)
		if err != nil {
			return nil, false, err
		}

		if len(invitations) < perPage {
			// Last page reached, user not found
			break
		}

		page++
	}

	return nil, false, nil
}
