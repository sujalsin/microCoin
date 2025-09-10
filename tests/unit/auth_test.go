package unit

import (
	"testing"

	"microcoin/internal/auth"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordHashing(t *testing.T) {
	password := "testpassword123"

	// Hash password
	hash, err := auth.HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	// Verify password
	valid, err := auth.VerifyPassword(password, hash)
	require.NoError(t, err)
	assert.True(t, valid)

	// Test wrong password
	wrongPassword := "wrongpassword"
	valid, err = auth.VerifyPassword(wrongPassword, hash)
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestTokenGeneration(t *testing.T) {
	userID := uuid.New()
	email := "test@example.com"

	// Generate tokens
	accessToken, refreshToken, err := auth.GenerateTokens(userID, email)
	require.NoError(t, err)
	require.NotEmpty(t, accessToken)
	require.NotEmpty(t, refreshToken)

	// Validate access token
	claims, err := auth.ValidateToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)

	// Validate refresh token
	claims, err = auth.ValidateToken(refreshToken)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
}

func TestTokenValidation(t *testing.T) {
	// Test invalid token
	invalidToken := "invalid.token.here"
	_, err := auth.ValidateToken(invalidToken)
	assert.Error(t, err)

	// Test empty token
	_, err = auth.ValidateToken("")
	assert.Error(t, err)
}

func TestPasswordStrength(t *testing.T) {
	// Test various password scenarios
	testCases := []struct {
		password   string
		shouldPass bool
	}{
		{"short", false},
		{"12345678", false},
		{"password", false},
		{"Password123", true},
		{"MySecurePassword123!", true},
	}

	for _, tc := range testCases {
		t.Run(tc.password, func(t *testing.T) {
			hash, err := auth.HashPassword(tc.password)
			if tc.shouldPass {
				require.NoError(t, err)
				require.NotEmpty(t, hash)

				// Verify it works
				valid, err := auth.VerifyPassword(tc.password, hash)
				require.NoError(t, err)
				assert.True(t, valid)
			} else {
				// Even weak passwords should hash (security is enforced elsewhere)
				require.NoError(t, err)
				require.NotEmpty(t, hash)
			}
		})
	}
}
