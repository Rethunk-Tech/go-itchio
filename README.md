<h1 align="center">go-itchio</h1>

<div align="center">

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.27.3+-00ADD8?logo=go&logoColor=white)](go.mod)

</div>

---

A client for the itch.io game pages and uploads, as mod managers read them. Package `itchio`, module `github.com/Rethunk-Tech/go-itchio`.

Reads the itch.io server-side API with a user's own API key, and answers the one question a pasted game page
address raises: is it a page, and which page is it. Errors a caller can tell apart: no key, rejected key, anything
else.

## Quick start

```bash
go get github.com/Rethunk-Tech/go-itchio
```

A worked example and the contracts: [HUMANS.md](HUMANS.md).

## Features

- Search games by text, and read the signed-in account.
- A game's uploads with filename, size, kind and demo flag, and the time-limited download address of one upload.
- Page helpers: validate a `user/game` id, build its address, and parse a pasted page address into that id.
- Answers capped at 16 MiB, a redirect off the API's origin refused, and the key in a header only.

## Documentation

| Doc | Audience |
| --- | --- |
| [HUMANS.md](HUMANS.md) | Running and using the package |
| [AGENTS.md](AGENTS.md) | Internals, invariants, and why |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Process — commits, tests, docs |
| [SECURITY.md](SECURITY.md) | Vulnerability disclosure |
| [CHANGELOG.md](CHANGELOG.md) | Release notes |

## License

MIT — see [LICENSE](LICENSE). © 2026 Rethunk.Tech, LLC.
