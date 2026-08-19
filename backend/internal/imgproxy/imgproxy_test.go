package imgproxy

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func newTestProxy(t *testing.T) *Proxy {
	t.Helper()
	p, err := NewAllowLocal(t.TempDir(), slog.Default())
	if err != nil {
		t.Fatalf("NewAllowLocal: %v", err)
	}
	return p
}

func TestSignRoundtrip(t *testing.T) {
	p := newTestProxy(t)
	u := "http://img.youtube.com/vi/abc/0.jpg"
	sig := p.Sign(u)
	if !p.valid(u, sig) {
		t.Fatal("firma válida rechazada")
	}
	if p.valid(u+"x", sig) {
		t.Fatal("firma de otra URL aceptada")
	}
	if p.valid(u, sig+"0") {
		t.Fatal("firma manipulada aceptada")
	}
}

func TestProxiedURLRoundtrip(t *testing.T) {
	base := "/index.php/apps/notes/api/v1/"
	u := "https://example.com/a%20b.jpg?x=1&y=2,3"
	pu := ProxiedURL(base, u, "sig")
	q, err := url.Parse(pu)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if q.Path != base+"img" {
		t.Fatalf("path inesperado: %s", q.Path)
	}
	if got := q.Query().Get("u"); got != u {
		t.Fatalf("u no recupera la original: %q != %q", got, u)
	}
}

func TestServeRejectsBadSignature(t *testing.T) {
	p := newTestProxy(t)
	srv := httptest.NewServer(http.HandlerFunc(p.Serve))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "?u=http://example.com/x.jpg&t=badbadbad")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("esperado 403, got %d", resp.StatusCode)
	}
}

func TestServeRejectsNonImage(t *testing.T) {
	p := newTestProxy(t)
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html>not an image</html>"))
	}))
	defer origin.Close()
	srv := httptest.NewServer(http.HandlerFunc(p.Serve))
	defer srv.Close()
	sig := p.Sign(origin.URL + "/x")
	resp, err := http.Get(srv.URL + "?u=" + url.QueryEscape(origin.URL+"/x") + "&t=" + sig)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("esperado 415, got %d", resp.StatusCode)
	}
}
