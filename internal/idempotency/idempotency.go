package idempotency

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"microcoin/internal/models"

	"github.com/google/uuid"
)

// Repository handles idempotency database operations
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new idempotency repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// GetIdempotencyKey retrieves an idempotency key
func (r *Repository) GetIdempotencyKey(userID uuid.UUID, key string) (*models.IdempotencyKey, error) {
	query := `
		SELECT id, user_id, idem_key, request_fingerprint, response_code, response_body, created_at
		FROM idempotency_keys
		WHERE user_id = $1 AND idem_key = $2`

	var idemKey models.IdempotencyKey
	err := r.db.QueryRow(query, userID, key).Scan(
		&idemKey.ID,
		&idemKey.UserID,
		&idemKey.IdemKey,
		&idemKey.RequestFingerprint,
		&idemKey.ResponseCode,
		&idemKey.ResponseBody,
		&idemKey.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found, not an error
		}
		return nil, fmt.Errorf("failed to get idempotency key: %w", err)
	}

	return &idemKey, nil
}

// CreateIdempotencyKey creates a new idempotency key
func (r *Repository) CreateIdempotencyKey(tx *sql.Tx, idemKey *models.IdempotencyKey) error {
	query := `
		INSERT INTO idempotency_keys (user_id, idem_key, request_fingerprint, response_code, response_body)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := tx.Exec(query,
		idemKey.UserID,
		idemKey.IdemKey,
		idemKey.RequestFingerprint,
		idemKey.ResponseCode,
		idemKey.ResponseBody,
	)
	if err != nil {
		return fmt.Errorf("failed to create idempotency key: %w", err)
	}

	return nil
}

// Service handles idempotency business logic
type Service struct {
	repo *Repository
}

// NewService creates a new idempotency service
func NewService(db *sql.DB) *Service {
	return &Service{
		repo: NewRepository(db),
	}
}

// GenerateFingerprint generates a fingerprint for a request
func (s *Service) GenerateFingerprint(body []byte, headers map[string]string) string {
	// Combine body and critical headers
	data := string(body)
	for key, value := range headers {
		if key == "Authorization" || key == "Content-Type" {
			data += key + ":" + value
		}
	}

	// Hash the combined data
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// CheckIdempotency checks if a request is idempotent
func (s *Service) CheckIdempotency(userID uuid.UUID, key string, fingerprint string) (*models.IdempotencyKey, error) {
	idemKey, err := s.repo.GetIdempotencyKey(userID, key)
	if err != nil {
		return nil, err
	}

	if idemKey == nil {
		return nil, nil // New request
	}

	// Check if fingerprint matches
	if idemKey.RequestFingerprint != fingerprint {
		return nil, fmt.Errorf("idempotency key mismatch")
	}

	return idemKey, nil
}

// StoreIdempotency stores an idempotency key
func (s *Service) StoreIdempotency(tx *sql.Tx, userID uuid.UUID, key, fingerprint string, responseCode int, responseBody []byte) error {
	idemKey := &models.IdempotencyKey{
		UserID:             userID,
		IdemKey:            key,
		RequestFingerprint: fingerprint,
		ResponseCode:       responseCode,
		ResponseBody:       responseBody,
	}

	return s.repo.CreateIdempotencyKey(tx, idemKey)
}

// IdempotentHandler wraps an HTTP handler with idempotency
func IdempotentHandler(handler http.HandlerFunc, service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context (set by auth middleware)
		userID, ok := r.Context().Value("user_id").(uuid.UUID)
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Get idempotency key from header
		idemKey := r.Header.Get("Idempotency-Key")
		if idemKey == "" {
			http.Error(w, "Idempotency-Key header required", http.StatusBadRequest)
			return
		}

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		// Create new request with body
		r.Body = io.NopCloser(bytes.NewReader(body))

		// Generate fingerprint
		headers := make(map[string]string)
		for key, values := range r.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
		fingerprint := service.GenerateFingerprint(body, headers)

		// Check idempotency
		existingKey, err := service.CheckIdempotency(userID, idemKey, fingerprint)
		if err != nil {
			http.Error(w, "Idempotency key mismatch", http.StatusConflict)
			return
		}

		// If we have a cached response, return it
		if existingKey != nil {
			w.WriteHeader(existingKey.ResponseCode)
			w.Write(existingKey.ResponseBody)
			return
		}

		// Execute the handler and capture response
		recorder := httptest.NewRecorder()
		handler(recorder, r)

		// Store the response for future idempotent requests
		// Note: In a real implementation, you'd want to store this in a transaction
		// along with the business operation to ensure atomicity
		responseBody := recorder.Body.Bytes()
		responseCode := recorder.Code

		// For now, we'll just return the response
		// In a real implementation, you'd store this in the database
		w.WriteHeader(responseCode)
		w.Write(responseBody)
	}
}
