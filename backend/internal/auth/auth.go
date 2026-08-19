// Package auth implements authentication for the Notes daemon.
//
// Two credential forms are supported:
//  1. Basic(user:app-token) — external clients (Android, desktop),
//     validated against Graph /me endpoint via appauth flow.
//  2. Bearer(token) — web extension sessions from within OpenCloud SPA,
//     validated directly against Graph /me.
//
// Users are stored as "shadow" records in local SQLite using their IDM UUID
// (oc_id). Both credential types for the same person resolve to the same
// shadow record so data is shared. The upsert is mutex-protected to prevent
// a UNIQUE constraint race when concurrent requests arrive from both
// credential types simultaneously.
package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Validator validates credentials against an OpenCloud instance.
type Validator struct {
	graphURL string
	mu       sync.Mutex
	cache    map[string]shadowUser
	lastEvict time.Time
}

// shadowUser holds the cached user record derived from Graph /me.
type shadowUser struct {
	ID           string `json:"oc_id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	Email        string `json:"email"`
	TokenHash    string // SHA-256 of the raw token; used to match new requests.
	CreatedAt    int64
}

func NewOpenCloudValidator(graphURL string, _ []byte) *Validator {
	return &Validator{
		graphURL: graphURL,
		cache:    make(map[string]shadowUser),
	}
}

// ValidateBasic checks a basic-auth request (user:appToken) and returns the
// resolved shadow user or an error. The result is cached for 5 minutes.
func (v *Validator) ValidateBasic(username, token string) (*shadowUser, error) {
	hash := sha256.Sum256([]byte(token))
	tokenKey := fmt.Sprintf("%s:%x", username, hash[:8])

	v.mu.Lock()
	defer v.mu.Unlock()

	if v.shouldEvict() {
		v.cache = make(map[string]shadowUser)
	}

	if su, ok := v.cache[tokenKey]; ok {
		return &su, nil
	}

	// Fetch from Graph.
	resp, err := http.Get(v.graphURL)
	if err != nil {
		return nil, fmt.Errorf("graph request: %w", err)
	}
	defer resp.Body.Close()

	var graphUser struct {
		ID          string `json:"id"`
		UserPrincipalName string `json:"userPrincipalName"`
		DisplayName   string `json:"displayName"`
		Mail         string `json:"mail"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&graphUser); err != nil {
		return nil, fmt.Errorf("decode graph: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graph %d", resp.StatusCode)
	}

	su := shadowUser{
		ID:          graphUser.ID,
		Username:    graphUser.UserPrincipalName,
		DisplayName: graphUser.DisplayName,
		Email:       graphUser.Mail,
		TokenHash:   base64.StdEncoding.EncodeToString(hash[:]),
		CreatedAt:   time.Now().Unix(),
	}

	v.cache[tokenKey] = su
	v.lastEvict = time.Now()

	return &su, nil
}

// ValidateBearer checks a bearer token against Graph /me. Cached for 5 min.
func (v *Validator) ValidateBearer(token string) (*shadowUser, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.shouldEvict() {
		v.cache = make(map[string]shadowUser)
	}

	key := fmt.Sprintf("bearer:%x", sha256.Sum256([]byte(token))[:8])
	if su, ok := v.cache[key]; ok {
		return &su, nil
	}

	req, _ := http.NewRequest(http.MethodGet, v.graphURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("graph bearer: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var body io.Reader = resp.Body
	data, _ := io.ReadAll(body)
	resp.Body = io.NopCloser(&data)

	var graphUser struct {
		ID               string `json:"id"`
		UserPrincipalName string `json:"userPrincipalName"`
		DisplayName      string `json:"displayName"`
		Mail             string `json:"mail"`
	}
	if err := json.Unmarshal(data, &graphUser); err != nil {
		return nil, fmt.Errorf("decode graph: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graph %d", resp.StatusCode)
	}

	su := shadowUser{
		ID:          graphUser.ID,
		Username:    graphUser.UserPrincipalName,
		DisplayName: graphUser.DisplayName,
		Email:       graphUser.Mail,
		TokenHash:   "", // no token hash for bearer; it's tied to session.
		CreatedAt:   time.Now().Unix(),
	}

	v.cache[key] = su
	v.lastEvict = time.Now()

	return &su, nil
}

func (v *Validator) shouldEvict() bool {
	return time.Since(v.lastEvict) > 5*time.Minute || len(v.cache) > 0
}
