package flight

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/darianmavgo/banquet"
	"github.com/pocketbase/pocketbase/core"
)

func TestErrorResponseFormat(t *testing.T) {
	tests := []struct {
		name         string
		debugMode    bool
		expectDebug  bool
		banquetURL   string
		query        string
		expectedMsg  string
		expectedCode int
	}{
		{
			name:         "Error without debug mode",
			debugMode:    false,
			expectDebug:  false,
			banquetURL:   "test/data.csv",
			query:        "SELECT * FROM tb0",
			expectedMsg:  "Test error message",
			expectedCode: 400,
		},
		{
			name:         "Error with debug mode enabled",
			debugMode:    true,
			expectDebug:  true,
			banquetURL:   "test/data.csv/tb0",
			query:        "SELECT * FROM tb0 WHERE id > 10",
			expectedMsg:  "Test error message",
			expectedCode: 400,
		},
		{
			name:         "Error with nil banquet",
			debugMode:    true,
			expectDebug:  true,
			banquetURL:   "",
			query:        "",
			expectedMsg:  "Parse error",
			expectedCode: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set debug mode
			if tt.debugMode {
				os.Setenv("DEBUG", "true")
			} else {
				os.Unsetenv("DEBUG")
				os.Unsetenv("VERBOSE")
			}

			// Create a mock request
			rec := httptest.NewRecorder()

			// Create a mock RequestEvent (simplified - just need Response)
			e := &core.RequestEvent{}
			e.Response = rec

			// Parse banquet if URL provided
			var b *banquet.Banquet
			if tt.banquetURL != "" {
				var err error
				b, err = banquet.ParseBanquet(tt.banquetURL)
				if err != nil {
					t.Fatalf("Failed to parse banquet URL: %v", err)
				}
			}

			// Check status code
			if rec.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, rec.Code)
			}

			// Check content type
			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}

			// Check response body contains message
			body := rec.Body.String()
			if body == "" {
				t.Error("Response body is empty")
			}

			// In debug mode, check for debug field
			if tt.expectDebug {
				if b != nil && !strings.Contains(body, "debug") {
					t.Errorf("Expected debug field in response when debug mode is enabled. Body: %s", body)
				}
				if tt.query != "" && !strings.Contains(body, "\"query\"") {
					t.Errorf("Expected 'query' field in debug info. Body: %s", body)
				}
			} else {
				// In non-debug mode, should not have debug field
				if strings.Contains(body, "\"debug\"") {
					t.Error("Debug field should not be present when debug mode is disabled")
				}
			}
		})
	}
}

func TestBanquetError(t *testing.T) {
	originalErr := ErrTest
	be := NewBanquetError(originalErr, "Custom message", 500, nil, "SELECT * FROM test", "")

	if be.Error() != "Custom message" {
		t.Errorf("Expected error message 'Custom message', got '%s'", be.Error())
	}

	if be.Status != 500 {
		t.Errorf("Expected status 500, got %d", be.Status)
	}

	if be.Query != "SELECT * FROM test" {
		t.Errorf("Expected query 'SELECT * FROM test', got '%s'", be.Query)
	}

	if be.Unwrap() != originalErr {
		t.Error("Unwrap() should return original error")
	}
}

// Test error for testing
var ErrTest = &testError{"test error"}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
