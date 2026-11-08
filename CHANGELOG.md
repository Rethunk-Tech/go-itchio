# Changelog

Notable changes to `go-itchio`. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `Client` over the itch.io server-side API with the user's own key: `Me`, `SearchGames`, `Uploads` and
  `DownloadURL`, with answers capped at 16 MiB and cross-origin redirects refused.
- `ErrNeedsKey` and `ErrBadKey`, so a caller tells a missing key from one itch.io rejected, including its HTTP 200
  answers carrying an `invalid key` error.
- Page helpers `ValidPage`, `PageURL` and `ParsePageURL` for `<user>.itch.io/<game>` page ids.

[Unreleased]: https://github.com/Rethunk-Tech/go-itchio/commits/main
