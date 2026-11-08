# Security

## Supported versions

The latest tagged release. There are no long-lived release branches, so a fix ships in the next tag
rather than being backported.

## Reporting a vulnerability

Use GitHub's **private vulnerability reporting** on this repository (Security → Report a
vulnerability). It stays private until a fix ships.

Please do not open a public issue for a suspected vulnerability first.

## What is in scope

- API answers (`Me`, `SearchGames`, `Uploads`, `DownloadURL`) are decoded from JSON read through a 16 MiB cap, and a
  body that is malformed or an error list never panics a call.
- A redirect answer is refused unless it stays on the API's origin and scheme, so the `Authorization` header never
  travels to another host; `DownloadURL` follows no redirect at all and only returns its `Location`.
- The API key travels in a header only; it never enters a URL, a wrapped error or an error's text.
- Page addresses (`ParsePageURL`) are parsed, never opened: only `https://<user>.itch.io/<slug>` with no user name,
  port, extra path segment or look-alike host yields an id, and `PageURL` rebuilds the address from that id alone, so
  a query, fragment or hostile scheme in pasted text is dropped.

## What is out of scope

The caller keeps the API key out of logs and hands it only to `Config.Key`; `Uploads`, `SearchGames` and `Me` echo
whatever itch.io returns, so a caller renders those strings as text rather than markup. Whether a signed download
address may be fetched, and whether an upload is the file a game needs (a demo or a soundtrack is an upload too), is
the caller's decision: the client reports `Upload.Type` and `Upload.Demo` and downloads nothing.
