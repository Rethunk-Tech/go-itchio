package itchio

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

type answer struct {
	code     int
	location string
	body     []byte
}

// FuzzHostileAnswer holds that no status, Location or body makes a call panic, and that DownloadURL never returns
// an address that is not absolute. One server serves the whole run: a server per input runs the machine out of
// ports.
func FuzzHostileAnswer(f *testing.F) {
	f.Add(302, "https://cdn.example/f?sig=1", []byte(nil))
	f.Add(302, "//other.example/x", []byte(`{"errors":["invalid key"]}`))
	f.Add(301, "%zz", []byte(nil))
	f.Add(200, "", []byte(`{"errors":["a","b"],"user":{"id":1}}`))
	f.Add(200, "", []byte(strings.Repeat(`{"uploads":[`, 20000)))
	var a atomic.Pointer[answer]
	a.Store(&answer{code: 200})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		cur := a.Load()
		w.Header()["Location"] = []string{cur.location}
		w.WriteHeader(cur.code)
		_, _ = w.Write(cur.body)
	}))
	f.Cleanup(srv.Close)
	c := New(Config{Key: "k", BaseURL: srv.URL})
	f.Fuzz(func(t *testing.T, code int, location string, body []byte) {
		if code < 200 || code > 599 || code == http.StatusNotModified {
			code = 200
		}
		a.Store(&answer{code: code, location: location, body: body})
		ctx := t.Context()
		_, _ = c.Me(ctx)
		_, _ = c.SearchGames(ctx, "x", 1)
		_, _ = c.Uploads(ctx, "1")
		link, err := c.DownloadURL(ctx, "1")
		if err != nil {
			return
		}
		if u, perr := url.Parse(link); perr != nil || !u.IsAbs() {
			t.Fatalf("DownloadURL gave %q for Location %q", link, location)
		}
	})
}

func TestALyingContentLengthFails(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		conn, rw, err := http.NewResponseController(w).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = rw.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 1099511627776\r\n\r\n{}")
		_ = rw.Flush()
	}))
	t.Cleanup(srv.Close)
	if _, err := New(Config{Key: "k", BaseURL: srv.URL}).Me(t.Context()); err == nil {
		t.Fatal("a body cut short of its Content-Length was accepted")
	}
}
