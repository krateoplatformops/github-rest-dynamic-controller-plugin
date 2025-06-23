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

var _ handlers.Handler = &getHandler{}
var _ handlers.Handler = &postHandler{}

type getHandler struct {
	handlers.HandlerOptions
}

type postHandler struct {
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

		// add field "message" to the response
		finalBody, err := AddFieldToResponse(correctedBody, "message", "User is a collaborator of the repository")
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
		finalBody, err := AddFieldToResponse([]byte("{}"), "message", "Invitation sent to user "+username+" for repository "+owner+"/"+repo)
		if err != nil {
			h.Log.Printf("Failed to add message field to response: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error adding message field: %v", err)))
			return
		}

		// print log final body
		h.Log.Printf("[Final body]: %s", string(finalBody))

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
