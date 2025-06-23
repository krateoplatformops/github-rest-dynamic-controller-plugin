package collaborator

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/krateoplatformops/github-rest-dynamic-controller-plugin/internal/handlers"
	"github.com/rs/zerolog"
)

// mockHTTPClient implements http.Client's Do method for testing
// this is needed for external API calls in the handler
// it allows us to simulate responses and errors without making real HTTP requests (e.g., for GitHub API calls).
type mockHTTPClient struct {
	responses map[string]*http.Response
	errors    map[string]error
	requests  []*http.Request
}

// newMockHTTPClient creates a new instance of mockHTTPClient
// with empty maps for responses and errors
// and an empty slice for requests.
func newMockHTTPClient() *mockHTTPClient {
	return &mockHTTPClient{
		responses: make(map[string]*http.Response),
		errors:    make(map[string]error),
		requests:  make([]*http.Request, 0),
	}
}

// Do implements the http.Client Do method for mockHTTPClient.
// It simulates sending an HTTP request and returns a response or an error
func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	// Store the request for verification
	m.requests = append(m.requests, req)

	key := req.URL.String()

	// Check if there's an error configured for this URL
	if err, exists := m.errors[key]; exists {
		return nil, err
	}

	// Return configured response or default 404
	if resp, exists := m.responses[key]; exists {
		return resp, nil
	}

	// Default response
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader(`{"message": "Not Found"}`)),
		Header:     make(http.Header),
	}, nil
}

// setResponse allows setting a predefined response for a specific URL
func (m *mockHTTPClient) setResponse(url string, statusCode int, body string) {
	m.responses[url] = &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func (m *mockHTTPClient) setError(url string, err error) {
	m.errors[url] = err
}

func (m *mockHTTPClient) getLastRequest() *http.Request {
	if len(m.requests) == 0 {
		return nil
	}
	return m.requests[len(m.requests)-1]
}

func (m *mockHTTPClient) getRequestCount() int {
	return len(m.requests)
}

func (m *mockHTTPClient) reset() {
	m.responses = make(map[string]*http.Response)
	m.errors = make(map[string]error)
	m.requests = make([]*http.Request, 0)
}

// createTestHandler creates a handler instance for testing with a mock client
func createTestHandler(mockClient *mockHTTPClient) *getHandler {
	logger := zerolog.New(io.Discard).With().Timestamp().Logger()
	h := &getHandler{
		HandlerOptions: handlers.HandlerOptions{
			Client: mockClient, // Use the mock client directly
			Log:    &logger,
		},
	}
	return h
}

// createTestHandlerWithSilentLog creates a handler with discarded logs
func createTestHandlerWithSilentLog(mockClient *mockHTTPClient) *getHandler {
	return createTestHandler(mockClient)
}

// Test data constants
const (
	testOwner    = "testowner"
	testRepo     = "testrepo"
	testUsername = "testuser"
	testToken    = "token test-token-123"
)

var (
	collaboratorExternalURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators/%s", testOwner, testRepo, testUsername)
	permissionExternalURL   = fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators/%s/permission", testOwner, testRepo, testUsername)
	validPermissionResp     = `{
		"permission": "admin",
		"user": {
			"login": "testuser",
			"id": 12345,
			"html_url": "https://github.com/testuser",
			"permissions": {
				"admin": true,
				"maintain": true,
				"push": true,
				"triage": true,
				"pull": true
			}
		},
		"role_name": "admin"
	}`
	validInvitationResp = `[{
		"id": 1,
		"node_id": "MDEwOlJlcG9zaXRvcnkxMjk2MjY5",
		"repository": {
			"id": 1296269,
			"name": "Hello-World",
			"full_name": "octocat/Hello-World"
		},
		"invitee": {
			"login": "testuser",
			"id": 12345,
			"node_id": "MDQ6VXNlcjU4MzIzMQ==",
			"avatar_url": "https://github.com/images/error/testuser_happy.gif",
			"html_url": "https://github.com/testuser",
			"type": "User"
		},
		"inviter": {
			"login": "testowner",
			"id": 67890,
			"node_id": "MDQ6VXNlcjU4MzIzMQ==",
			"avatar_url": "https://github.com/images/error/testowner_happy.gif",
			"html_url": "https://github.com/testowner",
			"type": "User"
		},
		"permissions": "read",
		"created_at": "2016-06-13T14:52:50-05:00",
		"url": "https://api.github.com/user/repository_invitations/1",
		"html_url": "https://github.com/testowner/testrepo/invitations",
		"expired": false
	}]`
	validInvitationWriteResp = `[{
		"id": 2,
		"node_id": "MDEwOlJlcG9zaXRvcnkxMjk2MjY5",
		"repository": {
			"id": 1296269,
			"name": "Hello-World",
			"full_name": "octocat/Hello-World"
		},
		"invitee": {
			"login": "testuser",
			"id": 12345,
			"node_id": "MDQ6VXNlcjU4MzIzMQ==",
			"avatar_url": "https://github.com/images/error/testuser_happy.gif",
			"html_url": "https://github.com/testuser",
			"type": "User"
		},
		"inviter": {
			"login": "testowner",
			"id": 67890,
			"node_id": "MDQ6VXNlcjU4MzIzMQ==",
			"avatar_url": "https://github.com/images/error/testowner_happy.gif",
			"html_url": "https://github.com/testowner",
			"type": "User"
		},
		"permissions": "write",
		"created_at": "2016-06-13T14:52:50-05:00",
		"url": "https://api.github.com/user/repository_invitations/2",
		"html_url": "https://github.com/testowner/testrepo/invitations",
		"expired": false
	}]`
	emptyInvitationResp = `[]`
)

// GetRepo returns a new handler for repository permissions
func TestGetRepo(t *testing.T) {
	t.Run("returns valid handler", func(t *testing.T) {
		client := &http.Client{}
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		opts := handlers.HandlerOptions{
			Client: client, // Use the real http.Client for this test
			Log:    &logger,
		}

		handlerInterface := GetRepo(opts)

		if handlerInterface == nil {
			t.Fatal("GetRepo should return a non-nil handler")
		}

		// Verify it implements the Handler interface
		var _ handlers.Handler = handlerInterface

		// Verify the handler has the correct type and options
		h, ok := handlerInterface.(*getHandler)
		if !ok {
			t.Fatal("GetRepo should return a *handler")
		}

		if h.Client != client {
			t.Error("Handler should have the provided client")
		}

		if h.Log != &logger {
			t.Error("Handler should have the provided logger")
		}
	})
}

func TestHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name                 string
		owner                string
		repo                 string
		username             string
		authHeader           string
		setupMock            func(*mockHTTPClient)
		expectedStatus       int
		expectedContentType  string
		expectedBodyContains string
		expectedRequestCount int
		verifyRequests       func(t *testing.T, mockClient *mockHTTPClient)
	}{
		{
			name:       "successful permission check with admin role",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				// First call: check if user is collaborator (GitHub returns 204)
				mockClient.setResponse(collaboratorExternalURL, http.StatusNoContent, "")
				// Second call: get permission (returns permission data)
				mockClient.setResponse(permissionExternalURL, http.StatusOK, validPermissionResp)
			},
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: `"permission":"admin"`,
			expectedRequestCount: 2,
			verifyRequests: func(t *testing.T, mockClient *mockHTTPClient) {
				if mockClient.getRequestCount() != 2 {
					t.Errorf("Expected 2 requests, got %d", mockClient.getRequestCount())
				}

				// Verify first request (collaborator check)
				req1 := mockClient.requests[0]
				if req1.URL.String() != collaboratorExternalURL {
					t.Errorf("First request URL = %s, want %s", req1.URL.String(), collaboratorExternalURL)
				}
				if req1.Header.Get("Authorization") != testToken {
					t.Errorf("First request Authorization header = %s, want %s", req1.Header.Get("Authorization"), testToken)
				}

				// Verify second request (permission check)
				req2 := mockClient.requests[1]
				if req2.URL.String() != permissionExternalURL {
					t.Errorf("Second request URL = %s, want %s", req2.URL.String(), permissionExternalURL)
				}
				if req2.Header.Get("Authorization") != testToken {
					t.Errorf("Second request Authorization header = %s, want %s", req2.Header.Get("Authorization"), testToken)
				}
			},
		},
		{
			name:       "successful permission check with `read` permission from GitHub (corrected to `pull`)",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				mockClient.setResponse(collaboratorExternalURL, http.StatusNoContent, "")
				readPermissionResp := `{
						"permission": "read",
						"user": {
							"login": "testuser",
							"id": 12345,
							"html_url": "https://github.com/testuser",
							"permissions": {
								"admin": false,
								"maintain": false,
								"push": false,
								"triage": true,
								"pull": true
							}
						},
						"role_name": "read"
					}`
				mockClient.setResponse(permissionExternalURL, http.StatusOK, readPermissionResp)
			},
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: `"permission":"pull"`, // we expect the permission to be corrected to `pull`
			expectedRequestCount: 2,
		},
		{
			name:       "successful permission check with `write` permission from GitHub (corrected to `push`)",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				mockClient.setResponse(collaboratorExternalURL, http.StatusNoContent, "")
				writePermissionResp := `{
						"permission": "write",
						"user": {
							"login": "testuser",
							"id": 12345,
							"html_url": "https://github.com/testuser",
							"permissions": {
								"admin": false,
								"maintain": false,
								"push": true,
								"triage": true,
								"pull": true
							}
						},
						"role_name": "write"
					}`
				mockClient.setResponse(permissionExternalURL, http.StatusOK, writePermissionResp)
			},
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: `"permission":"push"`, // we expect the permission to be corrected to `push`
			expectedRequestCount: 2,
		},
		{
			name:       "successful permission check with maintain role",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				mockClient.setResponse(collaboratorExternalURL, http.StatusNoContent, "")
				maintainPermissionResp := `{
					"permission": "write",
					"user": {
						"login": "testuser",
						"id": 12345,
						"html_url": "https://github.com/testuser",
						"permissions": {
							"admin": false,
							"maintain": true,
							"push": true,
							"triage": true,
							"pull": true
						}
					},
					"role_name": "maintain"
				}`
				mockClient.setResponse(permissionExternalURL, http.StatusOK, maintainPermissionResp)
			},
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: `"permission":"maintain"`,
			expectedRequestCount: 2,
		},
		{
			name:       "successful permission check with triage role",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				mockClient.setResponse(collaboratorExternalURL, http.StatusNoContent, "")
				triagePermissionResp := `{
					"permission": "read",
					"user": {
						"login": "testuser",
						"id": 12345,
						"html_url": "https://github.com/testuser",
						"permissions": {
							"admin": false,
							"maintain": false,
							"push": false,
							"triage": true,
							"pull": true
						}
					},
					"role_name": "triage"
				}`
				mockClient.setResponse(permissionExternalURL, http.StatusOK, triagePermissionResp)
			},
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: `"permission":"triage"`,
			expectedRequestCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockClient := newMockHTTPClient()
			tt.setupMock(mockClient)

			handler := createTestHandlerWithSilentLog(mockClient)

			// Create a proper ServeMux with the pattern matching
			mux := http.NewServeMux()
			mux.Handle("GET /repository/{owner}/{repo}/collaborators/{username}/permission", handler)

			// Create request with the proper path
			path := fmt.Sprintf("/repository/%s/%s/collaborators/%s/permission", tt.owner, tt.repo, tt.username)
			req := httptest.NewRequest("GET", path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute through the mux so path values are properly set
			mux.ServeHTTP(rr, req)

			// Verify status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}

			// Verify content type if expected
			if tt.expectedContentType != "" {
				contentType := rr.Header().Get("Content-Type")
				if contentType != tt.expectedContentType {
					t.Errorf("handler returned wrong content type: got %v want %v", contentType, tt.expectedContentType)
				}
			}

			// Verify response body contains expected content
			if tt.expectedBodyContains != "" {
				body := rr.Body.String()
				if !strings.Contains(body, tt.expectedBodyContains) {
					t.Errorf("handler response body does not contain expected content.\nGot: %s\nWant to contain: %s", body, tt.expectedBodyContains)
				}
			}

			// Verify request count
			if mockClient.getRequestCount() != tt.expectedRequestCount {
				t.Errorf("expected %d requests, got %d", tt.expectedRequestCount, mockClient.getRequestCount())
			}

			// Run custom request verification if provided
			if tt.verifyRequests != nil {
				tt.verifyRequests(t, mockClient)
			}
		})
	}
}
