package repo

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/krateoplatformops/github-rest-dynamic-controller-plugin/internal/handlers"
)

func GetRepo(opts handlers.HandlerOptions) handlers.Handler {
	return &getHandler{
		HandlerOptions: opts,
	}
}

func PostRepo(opts handlers.HandlerOptions) handlers.Handler {
	return &postHandler{
		HandlerOptions: opts,
	}
}

func PatchRepo(opts handlers.HandlerOptions) handlers.Handler {
	return &patchHandler{
		HandlerOptions: opts,
	}
}

var _ handlers.Handler = &getHandler{}
var _ handlers.Handler = &postHandler{}
var _ handlers.Handler = &patchHandler{}

type getHandler struct {
	handlers.HandlerOptions
}

type postHandler struct {
	handlers.HandlerOptions
}

type patchHandler struct {
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
func (h *getHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

		permissionGranted, err := ReadFieldFromBody(correctedBody, "permission")
		if err != nil {
			h.Log.Print("Failed to read permission from corrected body:", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprint("Error: ", err)))
			return
		}

		// add field "message" to the response
		finalBody, err := AddFieldToResponse(correctedBody, "message", "User is a collaborator of the repository "+owner+"/"+repo+" with permission "+fmt.Sprintf("%s", permissionGranted))
		if err != nil {
			h.Log.Print("Failed to add message field:", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprint("Error: ", err)))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(finalBody)

		h.Log.Printf("Corrected body: %s", string(finalBody))
		h.Log.Print("Successfully called", req.URL)
		return
	}

	// Otherwise, if user is NOT a collaborator of the repo,
	// or user does not exist
	// return the error status from GitHub (404 Not Found)
	h.Log.Println("User is not a collaborator of the repo OR does not exist", resp.StatusCode, req.URL)

	w.WriteHeader(resp.StatusCode)
	w.Write([]byte(fmt.Sprint("Error: ", resp.Status)))
}

// @Summary Add a repository collaborator
// @Description Add a repository collaborator or invite a user to collaborate on a repository
// @ID post-repo-collaborator
// @Param owner path string true "Owner of the repository"
// @Param repo path string true "Name of the repository"
// @Param username path string true "Username of the collaborator to add"
// @Param permission body repo.Permission true "Permission to grant to the collaborator"
// @Accept json
// @Produce json
// @Success 202  {object} repo.Message "Invitation sent to user"
// @Success 204 "User already collaborator"
// @Router /repository/{owner}/{repo}/collaborators/{username} [post]
func (h *postHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	username := r.PathValue("username")

	h.Log.Printf("Adding collaborator %s to repository %s/%s", username, owner, repo)

	authHeader := r.Header.Get("Authorization")

	// Read the request body (permission data)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.Log.Printf("Failed to read request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Error reading request body: %v", err)))
		return
	}

	// get  the permission from the body
	permissionToBeGranted, err := ReadFieldFromBody(body, "permission")
	if err != nil {
		h.Log.Printf("Failed to read permission from request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Error reading permission from request body: %v", err)))
		return
	}
	defer r.Body.Close()

	// Create request to GitHub API
	req, err := http.NewRequest("PUT", "https://api.github.com/repos/"+owner+"/"+repo+"/collaborators/"+username, nil)
	if err != nil {
		h.Log.Printf("Failed to create request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Error creating request: %v", err)))
		return
	}

	// Set authorization header if present
	if len(authHeader) > 0 {
		req.Header.Set("Authorization", authHeader)
	}

	// Set content type and body if there's data to send
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
	}

	// Make the request to GitHub API
	resp, err := h.Client.Do(req)
	if err != nil {
		h.Log.Printf("Failed to call GitHub API: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Error calling GitHub API: %v", err)))
		return
	}
	defer resp.Body.Close()

	// Read GitHub API response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		h.Log.Printf("Failed to read GitHub API response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Error reading GitHub API response: %v", err)))
		return
	}

	// Handle different GitHub API responses
	switch resp.StatusCode {
	case http.StatusCreated: // 201 - Invitation sent
		h.Log.Printf("Invitation sent to user %s for repository %s/%s", username, owner, repo)

		// Create a empty body and add just the message
		finalBody, err := AddFieldToResponse([]byte("{}"), "message", "Invitation sent to user "+username+" for repository "+owner+"/"+repo+" with permission "+fmt.Sprintf("%s", permissionToBeGranted))
		if err != nil {
			h.Log.Printf("Failed to add message field to response: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error adding message field: %v", err)))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted) // Change 201 to 202 as required
		w.Write(finalBody)

	case http.StatusNoContent: // 204 - User already collaborator
		h.Log.Printf("User %s is already a collaborator of repository %s/%s", username, owner, repo)
		w.WriteHeader(http.StatusNoContent)
		// No body for 204 responses

	default:
		// Forward the error status and body from GitHub API
		h.Log.Printf("GitHub API returned status %d for user %s and repository %s/%s", resp.StatusCode, username, owner, repo)
		w.WriteHeader(resp.StatusCode)
		if len(respBody) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.Write(respBody)
		} else {
			w.Write([]byte(fmt.Sprintf("Error: %s", resp.Status)))
		}
	}

	h.Log.Printf("Successfully processed collaborator request for %s", req.URL)
}

// @Summary Update repository collaborator permission or invitation
// @Description Update the permission of an existing collaborator or pending invitation
// @ID patch-repo-collaborator
// @Param owner path string true "Owner of the repository"
// @Param repo path string true "Name of the repository"
// @Param username path string true "Username of the collaborator"
// @Param permissions body repo.Permission true "New permission to set (`pull`, `push`, `admin`, `maintain`, `triage`)"
// @Accept json
// @Produce json
// @Success 200 {object} repo.Message "Permission updated successfully"
// @Success 202 {object} repo.Message "Invitation permission updated"
// @Router /repository/{owner}/{repo}/collaborators/{username} [patch]
func (h *patchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	username := r.PathValue("username")

	h.Log.Printf("Updating permission for user %s in repository %s/%s", username, owner, repo)

	authHeader := r.Header.Get("Authorization")

	// Read the request body (permission data)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.Log.Printf("Failed to read request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Error reading request body: %v", err)))
		return
	}

	// read which permission was granted (in original request body)
	permissionToBeGranted, err := ReadFieldFromBody(body, "permission")
	if err != nil {
		h.Log.Printf("Failed to read permission from request body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Error reading permission from request body: %v", err)))
		return
	}
	h.Log.Printf("Requested permission for user %s: %s", username, permissionToBeGranted)

	defer r.Body.Close()

	// First, check if user is already a collaborator
	collaboratorReq, err := http.NewRequest("GET", "https://api.github.com/repos/"+owner+"/"+repo+"/collaborators/"+username, nil)
	if err != nil {
		h.Log.Printf("Failed to create collaborator check request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Error creating request: %v", err)))
		return
	}

	if len(authHeader) > 0 {
		collaboratorReq.Header.Set("Authorization", authHeader)
	}

	collaboratorResp, err := h.Client.Do(collaboratorReq)
	if err != nil {
		h.Log.Printf("Failed to check collaborator status: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Error checking collaborator status: %v", err)))
		return
	}
	defer collaboratorResp.Body.Close()

	if collaboratorResp.StatusCode == http.StatusNoContent {

		// User is already a collaborator,
		// update their permission
		h.Log.Printf("User %s is already a collaborator, updating permission", username)

		updateReq, err := http.NewRequest("PUT", "https://api.github.com/repos/"+owner+"/"+repo+"/collaborators/"+username, nil)
		if err != nil {
			h.Log.Printf("Failed to create update request: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error creating update request: %v", err)))
			return
		}

		if len(authHeader) > 0 {
			updateReq.Header.Set("Authorization", authHeader)
		}

		if len(body) > 0 {
			updateReq.Header.Set("Content-Type", "application/json")
			updateReq.Body = io.NopCloser(bytes.NewReader(body))
			updateReq.ContentLength = int64(len(body))
		}

		updateResp, err := h.Client.Do(updateReq)
		if err != nil {
			h.Log.Printf("Failed to update collaborator permission: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error updating permission: %v", err)))
			return
		}
		defer updateResp.Body.Close()

		updateRespBody, err := io.ReadAll(updateResp.Body)
		if err != nil {
			h.Log.Printf("Failed to read update response: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error reading response: %v", err)))
			return
		}

		if updateResp.StatusCode == http.StatusNoContent {

			// Permission updated successfully
			finalBody, err := AddFieldToResponse([]byte("{}"), "message", "Permission updated successfully for collaborator "+username+" with permission "+fmt.Sprintf("%s", permissionToBeGranted))
			if err != nil {
				h.Log.Printf("Failed to add message field: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("Error adding message field: %v", err)))
				return
			}

			h.Log.Printf("Successfully updated permission for collaborator %s", username)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(finalBody)
			return
		} else {
			// Forward error from GitHub API
			h.Log.Printf("GitHub API returned error %d when updating collaborator", updateResp.StatusCode)
			w.WriteHeader(updateResp.StatusCode)
			if len(updateRespBody) > 0 {
				w.Header().Set("Content-Type", "application/json")
				w.Write(updateRespBody)
			} else {
				w.Write([]byte(fmt.Sprintf("Error: %s", updateResp.Status)))
			}
			return
		}
	}

	// User is not a collaborator, check if they have a pending invitation
	h.Log.Printf("User %s is not a collaborator, checking for pending invitations", username)

	invitation, found, err := h.findUserInvitation(owner, repo, username, authHeader)
	if err != nil {
		h.Log.Printf("Error checking invitations: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Error checking invitations: %v", err)))
		return
	}

	if found {
		// User has a pending invitation, update it
		h.Log.Printf("Found pending invitation for user %s (Invitation ID: %d), updating permission", username, invitation.ID)

		updateInviteReq, err := http.NewRequest("PATCH", fmt.Sprintf("https://api.github.com/repos/%s/%s/invitations/%d", owner, repo, invitation.ID), nil)
		if err != nil {
			h.Log.Printf("Failed to create invitation update request: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error creating invitation update request: %v", err)))
			return
		}

		if len(authHeader) > 0 {
			updateInviteReq.Header.Set("Authorization", authHeader)
		}

		if len(body) > 0 {
			body, err = CorrectGitHubUserPermissionsFieldReqBody(body) // Change field name and mapping `pull` to `read`, `push` to `write`
			if err != nil {
				h.Log.Printf("Failed to change permissions field: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("Error creating invitation update request: %v", err)))
				return
			}

			updateInviteReq.Header.Set("Content-Type", "application/json")
			updateInviteReq.Body = io.NopCloser(bytes.NewReader(body))
			updateInviteReq.ContentLength = int64(len(body))
		}

		updateInviteResp, err := h.Client.Do(updateInviteReq)
		if err != nil {
			h.Log.Printf("Failed to update invitation: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error updating invitation: %v", err)))
			return
		}
		defer updateInviteResp.Body.Close()

		updateInviteRespBody, err := io.ReadAll(updateInviteResp.Body)
		if err != nil {
			h.Log.Printf("Failed to read invitation update response: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error reading invitation response: %v", err)))
			return
		}

		if updateInviteResp.StatusCode == http.StatusOK {

			// Invitation updated successfully
			finalBody, err := AddFieldToResponse([]byte("{}"), "message", "Invitation permission updated successfully for user "+username+" with permission "+fmt.Sprintf("%s", permissionToBeGranted))
			if err != nil {
				h.Log.Printf("Failed to add message field: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("Error adding message field: %v", err)))
				return
			}

			h.Log.Printf("Successfully updated invitation permission for user %s", username)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted) // 202 for invitation update
			w.Write(finalBody)
			return
		} else {
			// Forward error from GitHub API
			h.Log.Printf("GitHub API returned error %d when updating invitation", updateInviteResp.StatusCode)
			w.WriteHeader(updateInviteResp.StatusCode)
			if len(updateInviteRespBody) > 0 {
				w.Header().Set("Content-Type", "application/json")
				w.Write(updateInviteRespBody)
			} else {
				w.Write([]byte(fmt.Sprintf("Error: %s", updateInviteResp.Status)))
			}
			return
		}
	}

	// User is neither a collaborator nor has a pending invitation
	h.Log.Printf("User %s has no collaborator status or pending invitation", username)
	w.WriteHeader(http.StatusNotFound)
	finalBody, err := AddFieldToResponse([]byte("{}"), "message", "User "+username+" is not a collaborator and has no pending invitation")
	if err != nil {
		h.Log.Printf("Failed to add message field: %v", err)
		w.Write([]byte(fmt.Sprintf("User %s not found as collaborator or invitee", username)))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(finalBody)
}

// findUserInvitation searches for a user invitation across all pages
func (h *patchHandler) findUserInvitation(owner, repo, username, authHeader string) (*GitHubInvitation, bool, error) {
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
