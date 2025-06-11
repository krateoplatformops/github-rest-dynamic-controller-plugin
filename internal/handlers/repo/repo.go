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

	// 2. If user is a collaborator (StatusNoContent = 204), get the permission
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

		// Return 200 OK with permission data

		// Read response body
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

	// Otherwise, if user is NOT a collaborator, return the error status from GitHub (401 Unauthorized)
	// Also if user does not exist, GitHub returns 401 Unauthorized
	h.Log.Println("User is not a collaborator", resp.StatusCode, req.URL)

	w.WriteHeader(resp.StatusCode)
	w.Write([]byte(fmt.Sprint("Error: ", resp.Status)))
}
