// Package imgproxy: proxy de imágenes firmado. El host OpenCloud sirve CSP
// img-src 'self' (+allowlist corta), así que el preview no puede cargar
// imágenes externas (p.ej. miniaturas de YouTube). Este proxy descarga la
// imagen y la sirve desde el propio dominio. El tag <img> del navegador NO
// puede llevar cabeceras de auth → la ruta pública va firmada con
// HMAC-SHA256 (secret persistente por instancia): solo se proxifican URLs
// firmadas por el propio servidor a petición de un usuario autenticado.
// Mitiga SSRF (transporte netguard) y abuso del proxy.
package imgproxy

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.opencloud.example.com/gnacho/ocnotes/backend/internal/netguard"
)

const (
	maxImageBytes = 8 << 20 // 8 MB
	fetchTimeout  = 15 * time.Second
	userAgent     = "ocnotes/0.1 (+https://github.com/gnacho/ocnotes)"
)

type Proxy struct {
	secret []byte
	dir    string
	client *http.Client
	log    *slog.Logger
}

// New carga (o genera) el secret en <dataDir>/imgsecret y prepara la caché.
func New(dataDir string, log *slog.Logger) (*Proxy, error) {
	secretPath := filepath.Join(dataDir, "imgsecret")
	secret, err := os.ReadFile(secretPath)
	if err != nil || len(secret) < 32 {
		secret = make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, secret); err != nil {
			return nil, fmt.Errorf("generar imgsecret: %w", err)
		}
		if err := os.WriteFile(secretPath, secret, 0o600); err != nil {
			return nil, fmt.Errorf("persistir imgsecret: %w", err)
		}
	}
	dir := filepath.Join(dataDir, "imgcache")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Proxy{
		secret: secret,
		dir:    dir,
		client: netguard.Client(fetchTimeout),
		log:    log,
	}, nil
}

// NewAllowLocal es como New pero el transporte permite loopback. SOLO tests.
func NewAllowLocal(dataDir string, log *slog.Logger) (*Proxy, error) {
	p, err := New(dataDir, log)
	if err != nil {
		return nil, err
	}
	p.client = netguard.ClientAllowLocal(fetchTimeout)
	return p, nil
}

// Sign devuelve la firma HMAC (hex, 32 chars) de una URL de imagen.
func (p *Proxy) Sign(u string) string {
	mac := hmac.New(sha256.New, p.secret)
	mac.Write([]byte(u))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

// ProxiedURL construye la URL pública del proxy para una URL firmada.
// u viaja con url.QueryEscape completo (incluye %→%25) para que q.Get()
// recupere EXACTAMENTE la URL que se firmó.
func ProxiedURL(base, u, sig string) string {
	return base + "img?u=" + url.QueryEscape(u) + "&t=" + sig
}

func (p *Proxy) valid(u, sig string) bool {
	return hmac.Equal([]byte(p.Sign(u)), []byte(sig))
}

func cachePath(u string) string {
	sum := sha256.Sum256([]byte(u))
	return hex.EncodeToString(sum[:])
}

// Serve responde con la imagen (caché en disco) o error. Firma inválida → 403.
func (p *Proxy) Serve(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	u, sig := q.Get("u"), q.Get("t")
	if u == "" || !p.valid(u, sig) {
		http.Error(w, `{"error":"invalid signature"}`, http.StatusForbidden)
		return
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		http.Error(w, `{"error":"scheme not allowed"}`, http.StatusForbidden)
		return
	}
	p.serveImage(w, r, u)
}

func (p *Proxy) serveImage(w http.ResponseWriter, r *http.Request, u string) {
	base := filepath.Join(p.dir, cachePath(u))
	if cachedCT, err := os.ReadFile(base + ".ct"); err == nil {
		if img, err := os.ReadFile(base); err == nil {
			respond(w, string(cachedCT), img)
			return
		}
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, u, nil)
	if err != nil {
		http.Error(w, `{"error":"invalid url"}`, http.StatusForbidden)
		return
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Debug("imgproxy fetch failed", "url", u, "err", err)
		http.Error(w, `{"error":"unavailable"}`, http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		http.Error(w, `{"error":"unavailable"}`, http.StatusBadGateway)
		return
	}
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if !strings.HasPrefix(ct, "image/") {
		http.Error(w, `{"error":"not an image"}`, http.StatusUnsupportedMediaType)
		return
	}
	img, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes))
	if err != nil || len(img) == 0 {
		http.Error(w, `{"error":"unavailable"}`, http.StatusBadGateway)
		return
	}
	_ = os.WriteFile(base, img, 0o600)
	_ = os.WriteFile(base+".ct", []byte(ct), 0o600)
	respond(w, ct, img)
}

func respond(w http.ResponseWriter, ct string, img []byte) {
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(img)
}
