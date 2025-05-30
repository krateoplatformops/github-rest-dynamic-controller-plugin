package repo

import (
	//"errors"
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

// mockableHandler allows us to inject a mock client for testing
type mockableHandler struct {
	*handler
	mockClient *mockHTTPClient
}

// Do implements the http.Client Do method for mockableHandler
func (h *mockableHandler) Do(req *http.Request) (*http.Response, error) {
	return h.mockClient.Do(req)
}

// createTestHandler creates a handler instance for testing with a mock client
func createTestHandler(mockClient *mockHTTPClient) *handler {
	logger := zerolog.New(io.Discard).With().Timestamp().Logger()
	h := &handler{
		HandlerOptions: handlers.HandlerOptions{
			Client: mockClient, // Use the mock client directly
			Log:    &logger,
		},
	}
	return h
}

// createTestHandlerWithSilentLog creates a handler with discarded logs
func createTestHandlerWithSilentLog(mockClient *mockHTTPClient) *handler {
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
		h, ok := handlerInterface.(*handler)
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

func TestHandler_ServeHTTP_PrivateRepoUserIsCollaborator(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		permissionResp string
		expectedStatus int
	}{
		{
			name:           "with auth header",
			authHeader:     testToken,
			permissionResp: validPermissionResp,
			expectedStatus: http.StatusOK,
		},
		//{
		//	name:           "without auth header",
		//	authHeader:     "",
		//	permissionResp: validPermissionResp, // to be verified
		//	expectedStatus: http.StatusOK, // to be verified
		//},
		{
			name:       "pull permission response",
			authHeader: testToken,
			permissionResp: `{
				"permission": "read",
				"user": {
					"login": "testuser",
					"id": 12345,
					"permissions": {
						"pull": true
					}
				}
			}`,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := newMockHTTPClient()

			// Set up mock responses from GitHub API (2 calls expected)
			mockClient.setResponse(collaboratorExternalURL, http.StatusNoContent, "")
			mockClient.setResponse(permissionExternalURL, http.StatusOK, tt.permissionResp)

			handler := createTestHandlerWithSilentLog(mockClient)

			req := httptest.NewRequest("GET", "/repository/"+testOwner+"/"+testRepo+"/collaborators/"+testUsername+"/permission", nil)
			req.SetPathValue("owner", testOwner)
			req.SetPathValue("repo", testRepo)
			req.SetPathValue("username", testUsername)

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			// Verify response status
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}

			// Verify content type
			expectedContentType := "application/json"
			if ct := w.Header().Get("Content-Type"); ct != expectedContentType {
				t.Errorf("Expected Content-Type %s, got %s", expectedContentType, ct)
			}

			// Verify 2 External API calls were made (mockClient should have 2 requests)
			if mockClient.getRequestCount() != 2 {
				t.Errorf("Expected 2 API calls, got %d", mockClient.getRequestCount())
			}

			// Verify both requests have correct auth header
			for i, req := range mockClient.requests {
				expectedAuth := tt.authHeader
				if actualAuth := req.Header.Get("Authorization"); actualAuth != expectedAuth {
					t.Errorf("Request %d: Expected Authorization header '%s', got '%s'", i, expectedAuth, actualAuth)
				}
			}

			// Verify response contains expected fields (after flattening)
			responseBody := w.Body.String()
			if !strings.Contains(responseBody, `"permission"`) {
				t.Error("Expected 'permission' field in response")
			}
		})
	}
}

// To be added:
// PrivateRepoUserIsNotCollaborator
// PublicRepoUserIsCollaborator
// PublicRepoUserIsNotCollaborator
// HTTPClientErrors
// InvalidJSON
// PathParameterVariations
// AuthorizationHeaders
// ResponseFlattening
