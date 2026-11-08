# AGENTS.md

`itchio` is a client for the itch.io server-side API and its `<user>.itch.io/<game>` page addresses, signed with the
user's own API key. It imports only the standard library and go-httpkit, and never a package that consumes it.

## Gate

`gate` is the command. Each override in `.gate.toml` is there because a runner
cannot resolve that check from PATH.

## Shape

- `Client.get` is the single JSON request path: it refuses an empty key before sending (`ErrNeedsKey`), caps the
  body at 16 MiB, and maps 401 and 403 to `ErrBadKey` through `mapStatus`. A new call goes through it, so no call
  can skip those bounds.
- itch.io reports a bad key two ways, a status and a 200 whose body is `{"errors":["invalid key"]}`; `errorsFrom`
  reads the second for every call, since a call that only checked statuses would decode that body into an empty
  result.
- `DownloadURL` builds its own client with `httpkit.NoFollow` and reads `Location`: the answer is a redirect to a
  CDN whose origin differs from the API's, so the client's `SameOrigin` policy would refuse it, and following it
  would carry the key off-origin.
- `ParsePageURL` returns an id, never an address, and `PageURL` rebuilds from that id: an address taken from pasted
  text is never opened as given.
- `maxBody` is policy (16 MiB, far above the largest answer itch.io sends); `requestTimeout` is a bound on a hung
  connection, not a caller-tunable.

## Files

| File | Holds |
| --- | --- |
| `client.go` | Config, Client, the request path, Me, SearchGames, Uploads, DownloadURL, the error mapping |
| `types.go` | User, Game, Upload |
| `page.go` | ValidPage, PageURL, ParsePageURL |
| `doc.go` | The package comment |
