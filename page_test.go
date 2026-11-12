package itchio

import "testing"

func FuzzParsePageURL(f *testing.F) {
	f.Add("https://leafo.itch.io/x-moon")
	f.Add("https://fasterthanlime.itch.io/dialog")
	f.Add("https://user.itch.io/game/")
	f.Add("https://user.itch.io/game?foo=bar")
	f.Add("https://user.itch.io/game#frag")
	f.Add("http://user.itch.io/game")
	f.Add("https://www.itch.io/game")
	f.Add("https://user.itch.io:443/game")
	f.Add("https://user:pass@host.itch.io/game")
	f.Add("https://USER.ITCH.IO/Game-Name")
	f.Add("")
	f.Add("not a url")
	f.Fuzz(func(t *testing.T, text string) {
		id, ok := ParsePageURL(text)
		if !ok {
			if id != "" {
				t.Fatalf("ParsePageURL(%q) failed with id %q", text, id)
			}
			return
		}
		if !ValidPage(id) {
			t.Fatalf("ParsePageURL(%q) = %q, not a ValidPage", text, id)
		}
		if len(id) > 140 {
			t.Fatalf("ParsePageURL(%q) = %q, longer than user/game bounds", text, id)
		}
		canon := PageURL(id)
		if canon == "" {
			t.Fatalf("ValidPage(%q) but PageURL is empty", id)
		}
		got, round := ParsePageURL(canon)
		if !round || got != id {
			t.Fatalf("round-trip %q -> %q -> %q (ok=%v)", canon, id, got, round)
		}
	})
}

func FuzzValidPage(f *testing.F) {
	f.Add("leafo/x-moon")
	f.Add("fasterthanlime/dialog")
	f.Add("user/game")
	f.Add("")
	f.Add("user")
	f.Add("user/game/extra")
	f.Add("/game")
	f.Add("user/")
	f.Add("www/0")
	f.Fuzz(func(t *testing.T, id string) {
		valid := ValidPage(id)
		u := PageURL(id)
		if valid != (u != "") {
			t.Fatalf("ValidPage(%q)=%v PageURL=%q", id, valid, u)
		}
		if !valid {
			return
		}
		if len(id) > 140 {
			t.Fatalf("ValidPage(%q) with id longer than user/game bounds", id)
		}
		got, ok := ParsePageURL(u)
		if !ok || got != id {
			t.Fatalf("PageURL(%q)=%q ParsePageURL -> %q ok=%v", id, u, got, ok)
		}
	})
}
