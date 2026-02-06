// Copyright (c) 2026 Toowoxx IT GmbH
// Licensed under the AGPL-3.0 license. See LICENSE for details.

package openid

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// OIDCDiscovery represents the OpenID Connect Discovery document.
// See: https://openid.net/specs/openid-connect-discovery-1_0.html
type OIDCDiscovery struct {
	// Issuer is the identifier of the OIDC provider (required).
	Issuer string `json:"issuer"`

	// AuthorizationEndpoint is the URL for user authorization (required).
	AuthorizationEndpoint string `json:"authorization_endpoint"`

	// TokenEndpoint is the URL for token exchange (required).
	TokenEndpoint string `json:"token_endpoint"`

	// UserInfoEndpoint is the URL for fetching user claims (recommended).
	UserInfoEndpoint string `json:"userinfo_endpoint"`

	// JwksURI is the URL for the JSON Web Key Set (required for ID token validation).
	JwksURI string `json:"jwks_uri"`

	// ScopesSupported lists the supported OAuth scopes.
	ScopesSupported []string `json:"scopes_supported"`

	// ResponseTypesSupported lists the supported response types.
	ResponseTypesSupported []string `json:"response_types_supported"`

	// GrantTypesSupported lists the supported grant types.
	GrantTypesSupported []string `json:"grant_types_supported"`

	// SubjectTypesSupported lists the supported subject identifier types.
	SubjectTypesSupported []string `json:"subject_types_supported"`

	// IDTokenSigningAlgValuesSupported lists the supported JWS algorithms for ID tokens.
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`

	// ClaimsSupported lists the claims that can be requested.
	ClaimsSupported []string `json:"claims_supported"`

	// CodeChallengeMethodsSupported lists supported PKCE methods (e.g., "S256", "plain").
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`

	// EndSessionEndpoint is the URL for logout (optional).
	EndSessionEndpoint string `json:"end_session_endpoint"`
}

// DiscoveryCache provides thread-safe caching for OIDC discovery documents.
type DiscoveryCache struct {
	mu       sync.RWMutex
	cache    map[string]*cachedDiscovery
	ttl      time.Duration
	client   *http.Client
}

type cachedDiscovery struct {
	document  *OIDCDiscovery
	expiresAt time.Time
}

// DefaultDiscoveryTTL is the default cache duration for discovery documents.
const DefaultDiscoveryTTL = 15 * time.Minute

// NewDiscoveryCache creates a new discovery cache with the given TTL.
func NewDiscoveryCache(ttl time.Duration) *DiscoveryCache {
	if ttl == 0 {
		ttl = DefaultDiscoveryTTL
	}
	return &DiscoveryCache{
		cache:  make(map[string]*cachedDiscovery),
		ttl:    ttl,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// DefaultDiscoveryCache is the global discovery cache instance.
var DefaultDiscoveryCache = NewDiscoveryCache(DefaultDiscoveryTTL)

// GetDiscovery fetches the OIDC discovery document from the given endpoint.
// Results are cached for the configured TTL.
func (c *DiscoveryCache) GetDiscovery(discoveryEndpoint string) (*OIDCDiscovery, error) {
	// Check cache first
	c.mu.RLock()
	if cached, ok := c.cache[discoveryEndpoint]; ok && time.Now().Before(cached.expiresAt) {
		c.mu.RUnlock()
		return cached.document, nil
	}
	c.mu.RUnlock()

	// Fetch fresh discovery document
	doc, err := c.fetchDiscovery(discoveryEndpoint)
	if err != nil {
		return nil, err
	}

	// Update cache
	c.mu.Lock()
	c.cache[discoveryEndpoint] = &cachedDiscovery{
		document:  doc,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return doc, nil
}

// fetchDiscovery retrieves the discovery document from the OIDC provider.
func (c *DiscoveryCache) fetchDiscovery(endpoint string) (*OIDCDiscovery, error) {
	resp, err := c.client.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OIDC discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OIDC discovery endpoint returned status %d", resp.StatusCode)
	}

	var doc OIDCDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("failed to parse OIDC discovery document: %w", err)
	}

	// Validate required fields
	if doc.Issuer == "" {
		return nil, fmt.Errorf("OIDC discovery document missing required 'issuer' field")
	}
	if doc.AuthorizationEndpoint == "" {
		return nil, fmt.Errorf("OIDC discovery document missing required 'authorization_endpoint' field")
	}
	if doc.TokenEndpoint == "" {
		return nil, fmt.Errorf("OIDC discovery document missing required 'token_endpoint' field")
	}

	return &doc, nil
}

// Invalidate removes a cached discovery document.
func (c *DiscoveryCache) Invalidate(discoveryEndpoint string) {
	c.mu.Lock()
	delete(c.cache, discoveryEndpoint)
	c.mu.Unlock()
}

// Clear removes all cached discovery documents.
func (c *DiscoveryCache) Clear() {
	c.mu.Lock()
	c.cache = make(map[string]*cachedDiscovery)
	c.mu.Unlock()
}

// DiscoveryEndpointFromIssuer constructs the discovery endpoint URL from an issuer URL.
// Per OIDC spec, the discovery document is at {issuer}/.well-known/openid-configuration
func DiscoveryEndpointFromIssuer(issuer string) string {
	issuer = strings.TrimSuffix(issuer, "/")
	return issuer + "/.well-known/openid-configuration"
}

// GetDiscovery is a convenience function using the default cache.
func GetDiscovery(discoveryEndpoint string) (*OIDCDiscovery, error) {
	return DefaultDiscoveryCache.GetDiscovery(discoveryEndpoint)
}

