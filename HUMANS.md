# HUMANS.md

Run and use `go-itchio`. Internals: [AGENTS.md](AGENTS.md).

## Quick start

```bash
go get github.com/Rethunk-Tech/go-itchio
```

```go
import (
 "context"
 "fmt"

 "github.com/Rethunk-Tech/go-itchio"
)

func find(ctx context.Context, key, text string) error {
 c := itchio.New(itchio.Config{Key: key, UserAgent: "my-mod-manager/1.0"})
 me, err := c.Me(ctx)
 if err != nil {
  return err
 }
 games, err := c.SearchGames(ctx, text, 1)
 if err != nil {
  return err
 }
 for _, g := range games {
  fmt.Println(me.Username, g.ID, g.Title)
 }
 uploads, err := c.Uploads(ctx, "134565")
 if err != nil {
  return err
 }
 if len(uploads) == 0 {
  return fmt.Errorf("game has no uploads")
 }
 link, err := c.DownloadURL(ctx, fmt.Sprint(uploads[0].ID))
 if err != nil {
  return err
 }
 fmt.Println(uploads[0].Filename, uploads[0].Size, link)
 return nil
}
```

A pasted address becomes a page id without touching the network:

```go
if id, ok := itchio.ParsePageURL("https://roburky.itch.io/emotional-inertia"); ok {
 fmt.Println(id, itchio.PageURL(id))
}
```

## Contracts

| Entry point | Contract |
| --- | --- |
| `New` | Never fails; an empty `Key` makes every call that needs one return `ErrNeedsKey` without sending. |
| `Me`, `SearchGames`, `Uploads`, `DownloadURL` | A 401 or 403, and itch.io's HTTP 200 `{"errors":["invalid key"]}`, are `ErrBadKey`; other statuses are `*httpkit.StatusError`, other error lists a plain error naming them. |
| `SearchGames` | Page 1 is every hit; page 2 and later are empty and send nothing, since the API ignores the number. |
| `DownloadURL` | Returns the redirect's `Location` and follows nothing, so the key never reaches the file host. The address expires; ask again before each download. |
| `ParsePageURL` | https only, `<user>.itch.io` only, one slug; always rebuild the address with `PageURL` rather than opening the pasted one. |
| Client calls | Every answer body is capped at 16 MiB, and a redirect to another origin is refused. |

## Requirements

Go 1.27.3 or newer, plus `lefthook` and `gitleaks` on PATH if you want the git hooks.

```bash
lefthook install
```

That is per clone; git does not commit hooks. Pre-commit scans staged changes for secrets, and
pre-push runs `gate`.

## Verify

```bash
gate
```
