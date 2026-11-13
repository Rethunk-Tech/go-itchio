package itchio

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-httpkit"
)

const userAgent = "test/1.0"

// fake answers the four calls the way api.itch.io does, refusing a key it was not built with.
func fake(t *testing.T) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		switch auth := r.Header.Get("Authorization"); auth {
		case "Bearer k1":
			if r.Header.Get("User-Agent") != userAgent {
				t.Errorf("user agent %q", r.Header.Get("User-Agent"))
			}
		case "":
			w.WriteHeader(http.StatusUnauthorized)
			return
		default:
			_, _ = w.Write([]byte(`{"errors":["invalid key"]}`))
			return
		}
		if strings.Contains(r.URL.String(), "k1") {
			t.Errorf("key in address %s", r.URL)
		}
		q := r.URL.Query()
		switch r.URL.Path {
		case "/profile":
			_, _ = w.Write([]byte(`{"user":{"username":"farmer","id":3}}`))
		case "/search/games":
			if q.Get("query") != "stardew cool" || q.Has("page") {
				t.Errorf("params %v", q)
			}
			_, _ = w.Write([]byte(`{"games":[{"id":7,"title":"Cool Mod","short_text":"hi","url":"https://a.itch.io/cool",` +
				`"cover_url":"c.png","published_at":"2026-01-01T00:00:00Z","user":{"username":"a"}}]}`))
		case "/games/7/uploads":
			_, _ = w.Write([]byte(`{"uploads":[{"id":1,"filename":"demo.zip","demo":true,"size":1,` +
				`"updated_at":"2026-01-01T00:00:00Z"},{"id":2,"filename":"cool.zip","size":9,"type":"default"}]}`))
		case "/uploads/2/download":
			http.Redirect(w, r, "https://cdn.itch.io/cool.zip", http.StatusFound)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return New(Config{Key: " k1 ", BaseURL: srv.URL + "/", UserAgent: userAgent})
}

func TestCalls(t *testing.T) {
	t.Parallel()
	c := fake(t)
	ctx := t.Context()
	me, err := c.Me(ctx)
	if err != nil || me.Username != "farmer" {
		t.Fatalf("me = %+v, %v", me, err)
	}
	games, err := c.SearchGames(ctx, " stardew cool ", 1)
	if err != nil || len(games) != 1 {
		t.Fatalf("search = %+v, %v", games, err)
	}
	if g := games[0]; g.ID != 7 || g.Title != "Cool Mod" || g.ShortText != "hi" || g.CoverURL != "c.png" ||
		g.URL != "https://a.itch.io/cool" || g.PublishedAt != "2026-01-01T00:00:00Z" || g.User.Username != "a" {
		t.Errorf("game = %+v", g)
	}
	// itch.io answers every hit on the first page, so a later page is empty and sends nothing.
	if later, err := c.SearchGames(ctx, "stardew cool", 2); err != nil || len(later) != 0 {
		t.Fatalf("page 2 = %+v, %v", later, err)
	}
	uploads, err := c.Uploads(ctx, "7")
	if err != nil || len(uploads) != 2 {
		t.Fatalf("uploads = %+v, %v", uploads, err)
	}
	if u := uploads[0]; u.ID != 1 || u.Filename != "demo.zip" || !u.Demo || u.Size != 1 || u.UpdatedAt == "" {
		t.Errorf("upload = %+v", u)
	}
	if u := uploads[1]; u.ID != 2 || u.Type != "default" || u.Demo || u.Size != 9 {
		t.Errorf("upload = %+v", u)
	}
	link, err := c.DownloadURL(ctx, "2")
	if err != nil || link != "https://cdn.itch.io/cool.zip" {
		t.Fatalf("download url = %q, %v", link, err)
	}
}

func TestKeyHandling(t *testing.T) {
	t.Parallel()
	c := fake(t)
	none := New(Config{BaseURL: c.base})
	ctx := t.Context()
	if _, err := none.Me(ctx); !errors.Is(err, ErrNeedsKey) {
		t.Errorf("me without key = %v", err)
	}
	if _, err := none.SearchGames(ctx, "x", 1); !errors.Is(err, ErrNeedsKey) {
		t.Errorf("search without key = %v", err)
	}
	if _, err := none.Uploads(ctx, "7"); !errors.Is(err, ErrNeedsKey) {
		t.Errorf("uploads without key = %v", err)
	}
	if _, err := none.DownloadURL(ctx, "2"); !errors.Is(err, ErrNeedsKey) {
		t.Errorf("download without key = %v", err)
	}
	bad := New(Config{Key: "nope", BaseURL: c.base})
	if _, err := bad.Me(ctx); !errors.Is(err, ErrBadKey) {
		t.Errorf("rejected key = %v", err)
	}
	if _, err := bad.Uploads(ctx, "7"); !errors.Is(err, ErrBadKey) {
		t.Errorf("rejected key on a 200 with errors = %v", err)
	}
}

func TestForbiddenIsABadKey(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)
	if _, err := New(Config{Key: "k", BaseURL: srv.URL}).Me(t.Context()); !errors.Is(err, ErrBadKey) {
		t.Fatalf("403 = %v", err)
	}
}

func TestOtherStatusIsReportedNotParsed(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"games":[{"id":1,"title":"T"}]}`))
	}))
	t.Cleanup(srv.Close)
	_, err := New(Config{Key: "k", BaseURL: srv.URL}).SearchGames(t.Context(), "x", 1)
	if se, ok := errors.AsType[*httpkit.StatusError](err); !ok || se.Code != http.StatusBadGateway {
		t.Fatalf("err = %v, want the 502 as a status error", err)
	}
}

func TestAnswersOtherErrors(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"errors":["game not found","try again"]}`))
	}))
	t.Cleanup(srv.Close)
	_, err := New(Config{Key: "k", BaseURL: srv.URL}).Uploads(t.Context(), "7")
	if err == nil || err.Error() != "itch.io: game not found; try again" {
		t.Fatalf("err = %v", err)
	}
}

func TestRedirectOffOriginIsRefused(t *testing.T) {
	t.Parallel()
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"user":{"username":"thief"}}`))
	}))
	t.Cleanup(elsewhere.Close)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set by hand rather than http.Redirect, which a linter treats as an open redirect wherever the
		// destination is not a constant.
		w.Header().Set("Location", elsewhere.URL+r.URL.Path)
		w.WriteHeader(http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	if me, err := New(Config{Key: "k", BaseURL: srv.URL}).Me(t.Context()); err == nil {
		t.Fatalf("followed a redirect off origin: %+v", me)
	}
}

func TestPageHelpers(t *testing.T) {
	t.Parallel()
	for _, id := range []string{"a/b", "rob-urky/emotional-inertia", "user_1/Game-2"} {
		if !ValidPage(id) || PageURL(id) == "" {
			t.Errorf("%q is a page", id)
		}
	}
	if PageURL("rob/emotional-inertia") != "https://rob.itch.io/emotional-inertia" {
		t.Errorf("url = %q", PageURL("rob/emotional-inertia"))
	}
	for _, id := range []string{"", "rob", "Rob/game", "rob/", "/game", "rob/game/two", "rob/ga me"} {
		if ValidPage(id) || PageURL(id) != "" {
			t.Errorf("%q is not a page", id)
		}
	}
	for text, want := range map[string]string{
		"https://rob.itch.io/emotional-inertia":    "rob/emotional-inertia",
		" https://rob.itch.io/emotional-inertia/ ": "rob/emotional-inertia",
		"https://ROB.itch.io/game?x=1#y":           "rob/game",
		"https://rob.itch.io/game/":                "rob/game",
		"https://sub.rob.itch.io/game":             "", // a user name has no dot
		"http://rob.itch.io/game":                  "",
		"https://rob.itch.io:8443/game":            "",
		"https://itch.io/game":                     "",
		"https://www.itch.io/game":                 "",
		"https://rob.itch.io":                      "",
		"https://rob.itch.io/game/two":             "",
		"https://user@rob.itch.io/game":            "",
		"https://rob.itch.io.evil.example/game":    "",
		"https://evil.example/rob.itch.io/game":    "",
		"not an address":                           "",
		"https://rob.itch.io/":                     "",
		"https://rob.itch.io/-game":                "",
	} {
		got, ok := ParsePageURL(text)
		if got != want || ok != (want != "") {
			t.Errorf("ParsePageURL(%q) = %q, %v; want %q", text, got, ok, want)
		}
	}
}

func redirectServer(t *testing.T, tls bool, loc string) *httptest.Server {
	t.Helper()
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", loc)
		w.WriteHeader(http.StatusFound)
	})
	srv := httptest.NewServer(h)
	if tls {
		srv = httptest.NewTLSServer(h)
	}
	t.Cleanup(srv.Close)
	return srv
}

func TestDownloadURLIgnoresTheCallersRedirectPolicy(t *testing.T) {
	t.Parallel()
	srv := redirectServer(t, false, "https://cdn.itch.io/cool.zip")
	hc := &http.Client{CheckRedirect: httpkit.SameOrigin}
	link, err := New(Config{Key: "k", BaseURL: srv.URL, HTTP: hc}).DownloadURL(t.Context(), "2")
	if err != nil || link != "https://cdn.itch.io/cool.zip" {
		t.Fatalf("link = %q, %v", link, err)
	}
}

func TestDownloadURLResolvesARelativeLocation(t *testing.T) {
	t.Parallel()
	srv := redirectServer(t, false, "/files/cool.zip?sig=x")
	link, err := New(Config{Key: "k", BaseURL: srv.URL}).DownloadURL(t.Context(), "2")
	if err != nil || link != srv.URL+"/files/cool.zip?sig=x" {
		t.Fatalf("link = %q, %v", link, err)
	}
}

func TestDownloadURLRefusesAnInsecureLocation(t *testing.T) {
	t.Parallel()
	srv := redirectServer(t, true, "http://cdn.itch.io/cool.zip")
	c := New(Config{Key: "k", BaseURL: srv.URL, HTTP: srv.Client()})
	if link, err := c.DownloadURL(t.Context(), "2"); err == nil {
		t.Fatalf("link = %q, want an error", link)
	}
}
