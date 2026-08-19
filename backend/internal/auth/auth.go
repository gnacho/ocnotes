package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type ShadowUser struct {
	ID           string `json:"oc_id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	Email        string `json:"email"`
	TokenHash    string
	CreatedAt    int64
}

type Validator struct {
	graphURL  string
	mu        sync.Mutex
	cache     map[string]ShadowUser
	lastEvict time.Time
}

func NewOpenCloudValidator(graphURL string) *Validator {
	return &Validator{
		graphURL: graphURL,
		cache:    make(map[string]ShadowUser),
	}
}

func (v *Validator) ValidateBasic(username, token string) (*ShadowUser, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.shouldEvict() {
		v.cache = make(map[string]ShadowUser)
	}

	key := fmt.Sprintf("basic:%s", username)
	if su, ok := v.cache[key]; ok {
		return &su, nil
	}

	resp, err := http.Get(v.graphURL)
	if err != nil {
		return nil, fmt.Errorf("graph request: %w", err)
	}
	defer resp.Body.Close()

	var body io.Reader = resp.Body
	data, _ := io.ReadAll(body)
	resp.Body = io.NopCloser(bytes.NewReader(data))

	var graphUser struct {
		ID                string `json:"id"`
		UserPrincipalName string `json:"userPrincipalName"`
		DisplayName       string `json:"displayName"`
		Mail              string `json:"mail"`
	}
	if err := json.Unmarshal(data, &graphUser); err != nil {
		return nil, fmt.Errorf("decode graph: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graph %d", resp.StatusCode)
	}

	su := ShadowUser{
		ID:          graphUser.ID,
		Username:    graphUser.UserPrincipalName,
		DisplayName: graphUser.DisplayName,
		Email:       graphUser.Mail,
		TokenHash:   base64.StdEncoding.EncodeToString([]byte(token)),
		CreatedAt:   time.Now().Unix(),
	}

	v.cache[key] = su
	v.lastEvict = time.Now()

	return &su, nil
}

func (v *Validator) ValidateBearer(token string) (*ShadowUser, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.shouldEvict() {
		v.cache = make(map[string]ShadowUser)
	}

	key := fmt.Sprintf("bearer:%x", token[:32])
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

	data, _ := io.ReadAll(resp.Body)

	var graphUser struct {
		ID                string `json:"id"`
		UserPrincipalName string `json:"userPrincipalName"`
		DisplayName       string `json:"displayName"`
		Mail              string `json:"mail"`
	}
	if err := json.Unmarshal(data, &graphUser); err != nil {
		return nil, fmt.Errorf("decode graph: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graph %d", resp.StatusCode)
	}

	su := ShadowUser{
		ID:          graphUser.ID,
		Username:    graphUser.UserPrincipalName,
		DisplayName: graphUser.DisplayName,
		Email:       graphUser.Mail,
		TokenHash:   "",
		CreatedAt:   time.Now().Unix(),
	}

	v.cache[key] = su
	v.lastEvict = time.Now()

	return &su, nil
}

func (v *Validator) shouldEvict() bool {
	return time.Since(v.lastEvict) > 5*time.Minute || len(v.cache) == 0
}
