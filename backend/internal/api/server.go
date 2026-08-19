package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.opencloud.example.com/gnacho/ocnotes/backend/internal/auth"
	"git.opencloud.example.com/gnacho/ocnotes/backend/internal/store"
)

const (
	Base               = "/index.php/apps/notes/api/v1/"
	APIVersionsHeader  = "X-Notes-API-Versions"
	AllowedAPIVersions = "0.2, 1.4"
	Version            = "6.0.0"
)

type noteKeyType struct{}

type prunedNote struct {
	ID int64 `json:"id"`
}

type Server struct {
	base      string
	store     *store.Store
	validator *auth.Validator
}

func NewServer(base string, s *store.Store, v *auth.Validator) *Server {
	return &Server{base: base, store: s, validator: v}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc(s.base+"notes", s.handleNotes)
	mux.HandleFunc(s.base+"notes/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, s.base), "/")
		if len(parts) >= 2 && parts[0] == "notes" && parts[1] != "" {
			idStr := parts[1]
			s.handleNoteByID(w, r, idStr)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc(s.base+"settings", s.handleSettings)
	mux.HandleFunc(s.base+"attachment/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, s.base), "/")
		if len(parts) >= 2 && parts[0] == "attachment" && parts[1] != "" {
			w.Header().Set(APIVersionsHeader, AllowedAPIVersions)
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error":"attachments not implemented yet"}`)
		} else {
			http.NotFound(w, r)
		}
	})

	// OCS endpoints for clients (capabilities, user info)
	ocsMux := http.NewServeMux()
	ocsMux.HandleFunc("/ocs/v2.php/cloud/capabilities", s.handleCapabilities)
	ocsMux.HandleFunc("/ocs/v2.php/cloud/user", s.handleUserInfo)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/ocs/v2.php/") {
			ocsMux.ServeHTTP(w, r)
			return
		}
		s.middleware(mux).ServeHTTP(w, r)
	})
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(APIVersionsHeader, AllowedAPIVersions)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, If-Match, If-None-Match, X-Requested-With, Accept, OCS-APIRequest")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		user, err := s.authenticate(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), noteKeyType{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) authenticate(r *http.Request) (*auth.ShadowUser, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("no auth header")
	}

	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		return s.validator.ValidateBearer(token)
	}

	if strings.HasPrefix(authHeader, "Basic ") {
		decoded, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid basic auth")
		}
		return s.validator.ValidateBasic(parts[0], parts[1])
	}

	return nil, fmt.Errorf("unsupported auth")
}

func (s *Server) handleNotes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetNotes(w, r)
	case http.MethodPost:
		s.handleCreateNote(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleNoteByID(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetNote(w, r, id)
	case http.MethodPut:
		s.handleUpdateNote(w, r, id)
	case http.MethodDelete:
		s.handleDeleteNote(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetSettings(w, r)
	case http.MethodPut:
		s.handlePutSettings(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGetNotes(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	exclude := strings.Split(r.URL.Query().Get("exclude"), ",")
	pruneBeforeStr := r.URL.Query().Get("pruneBefore")
	chunkSizeStr := r.URL.Query().Get("chunkSize")

	var pruneBefore int64
	if pruneBeforeStr != "" {
		pruneBefore, _ = strconv.ParseInt(pruneBeforeStr, 10, 64)
	}

	var chunkSize int
	if chunkSizeStr != "" {
		chunkSize, _ = strconv.Atoi(chunkSizeStr)
	}

	limit := 0
	if chunkSize > 0 {
		limit = chunkSize + 1
	}

	notes, err := s.store.ListNotes(category, exclude, limit, pruneBefore)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	now := time.Now().Unix()
	tm := time.Unix(now, 0)
	w.Header().Set("Last-Modified", tm.Format(http.TimeFormat))
	if len(notes) > 0 {
		w.Header().Set("ETag", fmt.Sprintf("%x", sha256.Sum256([]byte(notes[0].Etag))))
	}

	if len(notes) == 0 {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	if pruneBefore > 0 {
		pruned := make([]interface{}, 0, len(notes))
		for _, n := range notes {
			if n.Modified >= pruneBefore {
				pruned = append(pruned, map[string]interface{}{
					"id":       n.ID,
					"etag":     n.Etag,
					"modified": n.Modified,
					"title":    n.Title,
					"category": n.Category,
					"content":  n.Content,
					"favorite": n.Favorite,
				})
			} else {
				pruned = append(pruned, &prunedNote{ID: n.ID})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pruned)
		return
	}

	result := make([]map[string]interface{}, 0, len(notes))
	for _, n := range notes {
		if len(exclude) > 1 && exclude[0] != "" {
			m := map[string]interface{}{
				"id":       n.ID,
				"etag":     n.Etag,
				"modified": n.Modified,
				"title":    n.Title,
				"category": n.Category,
				"favorite": n.Favorite,
			}
			for _, e := range exclude {
				e = strings.TrimSpace(e)
				if e == "content" {
					delete(m, "content")
				}
			}
			result = append(result, m)
		} else {
			result = append(result, map[string]interface{}{
				"id":       n.ID,
				"etag":     n.Etag,
				"modified": n.Modified,
				"title":    n.Title,
				"category": n.Category,
				"content":  n.Content,
				"favorite": n.Favorite,
			})
		}
	}

	if chunkSize > 0 && len(result) > chunkSize {
		w.Header().Set("X-Notes-Chunk-Cursor", fmt.Sprintf("%d", now))
		result = result[:chunkSize]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleCreateNote(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		Category string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if input.Title == "" {
		input.Title = "New note"
	}
	now := time.Now().Unix()
	note, err := s.store.CreateNote(input.Title, input.Content, input.Category, now)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(note)
}

func (s *Server) handleGetNote(w http.ResponseWriter, r *http.Request, id int64) {
	note, err := s.store.GetNote(id)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("ETag", note.Etag)
	json.NewEncoder(w).Encode(note)
}

func (s *Server) handleUpdateNote(w http.ResponseWriter, r *http.Request, id int64) {
	note, err := s.store.GetNote(id)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	expectedEtag := strings.Trim(r.Header.Get("If-Match"), "\"")
	if expectedEtag != "" && expectedEtag != note.Etag {
		w.WriteHeader(http.StatusPreconditionFailed)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(note)
		return
	}

	var input struct {
		Title    *string `json:"title"`
		Content  *string `json:"content"`
		Category *string `json:"category"`
		Favorite *bool   `json:"favorite"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	now := time.Now().Unix()
	updated, err := s.store.UpdateNote(id, input.Title, input.Content, input.Category, input.Favorite, now)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (s *Server) handleDeleteNote(w http.ResponseWriter, r *http.Request, id int64) {
	err := s.store.DeleteNote(id)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{}`)
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.store.GetSettings()
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (s *Server) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var input store.Settings
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	result, err := s.store.UpdateSettings(input)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	xml := fmt.Sprintf(`<?xml version="1.0"?>
<ocs>
 <meta>
  <status>ok</status>
  <statuscode>200</statuscode>
  <message>OK</message>
 </meta>
 <data>
  <capabilities>
   <notes>
    <api_version>["0.2","1.4"]</api_version>
    <version>%s</version>
   </notes>
  </capabilities>
 </data>
</ocs>`, Version)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	fmt.Fprint(w, xml)
}

func (s *Server) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, _ := s.authenticate(r)
	displayName := "User"
	if user != nil {
		displayName = user.DisplayName
	}

	xml := fmt.Sprintf(`<?xml version="1.0"?>
<ocs>
 <meta>
  <status>ok</status>
  <statuscode>200</statuscode>
  <message>OK</message>
 </meta>
 <data>
  <id>%s</id>
  <displayname>%s</displayname>
 </data>
</ocs>`, displayName, displayName)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	fmt.Fprint(w, xml)
}
