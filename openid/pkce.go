// Copyright (c) 2026 Toowoxx IT GmbH
// Licensed under the AGPL-3.0 license. See LICENSE for details.

package openid

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// PKCE (Proof Key for Code Exchange) provides protection against
// authorization code interception attacks.
// See: https://datatracker.ietf.org/doc/html/rfc7636

const (
	// PKCECodeVerifierLength is the length of the code verifier (43-128 chars per RFC 7636).
	PKCECodeVerifierLength = 64

	// PKCEChallengeMethodS256 is the SHA256 challenge method.
	PKCEChallengeMethodS256 = "S256"
)

// PKCEParams holds the PKCE code verifier and challenge.
type PKCEParams struct {
	// CodeVerifier is the high-entropy cryptographic random string (43-128 chars).
	CodeVerifier string

	// CodeChallenge is the SHA256 hash of the verifier, base64url-encoded.
	CodeChallenge string

	// ChallengeMethod is always "S256" for security.
	ChallengeMethod string
}

// GeneratePKCE creates a new PKCE code verifier and challenge.
// The challenge is computed as: BASE64URL(SHA256(verifier))
//
// Usage:
//  1. Generate PKCE params
//  2. Include code_challenge and code_challenge_method in authorization URL
//  3. Store code_verifier in session/state
//  4. Include code_verifier in token exchange request
func GeneratePKCE() (*PKCEParams, error) {
	// Generate cryptographically random bytes for the verifier
	verifierBytes := make([]byte, PKCECodeVerifierLength)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, err
	}

	// URL-safe base64 encode without padding (per RFC 7636)
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Compute challenge: BASE64URL(SHA256(verifier))
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return &PKCEParams{
		CodeVerifier:    verifier,
		CodeChallenge:   challenge,
		ChallengeMethod: PKCEChallengeMethodS256,
	}, nil
}

// ComputeChallenge computes the code_challenge from a given code_verifier.
// This is useful for verification or when you need to recompute the challenge.
func ComputeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// ValidatePKCE verifies that a code_verifier matches a code_challenge.
// Returns true if SHA256(verifier) base64url-encoded equals the challenge.
func ValidatePKCE(verifier, challenge string) bool {
	computed := ComputeChallenge(verifier)
	return computed == challenge
}
