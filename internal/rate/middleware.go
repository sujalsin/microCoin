package rate

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"microcoin/internal/auth"
	"microcoin/internal/models"

	"github.com/google/uuid"
)

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(limiter *Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting for certain endpoints
			if shouldSkipRateLimit(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Get user ID from context
			userID, ok := auth.GetUserIDFromContext(r.Context())
			if !ok {
				http.Error(w, "User not authenticated", http.StatusUnauthorized)
				return
			}

			// Check rate limit
			allowed, err := limiter.Allow(r.Context(), userID)
			if err != nil {
				http.Error(w, "Rate limit check failed", http.StatusInternalServerError)
				return
			}

			if !allowed {
				// Return rate limit error
				errorResp := models.ErrorResponse{
					Error: models.ErrorDetail{
						Code:      models.ErrorCodeRateLimit,
						Message:   "Rate limit exceeded",
						RequestID: generateRequestID(),
					},
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(errorResp)
				return
			}

			// Add rate limit headers
			remaining, err := limiter.GetRemainingTokens(r.Context(), userID)
			if err == nil {
				w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))
			}

			next.ServeHTTP(w, r)
		})
	}
}

// shouldSkipRateLimit determines if rate limiting should be skipped for a given path
func shouldSkipRateLimit(path string) bool {
	skipPaths := []string{
		"/health",
		"/metrics",
		"/auth/signup",
		"/auth/login",
	}

	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
	}

	return false
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return uuid.New().String()
}
