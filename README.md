# ssh.austinpoor.com

![Demo of the ssh app](./assets/demo.gif)

_created by Austin Poor_

A terminal companion to [austinpoor.com](https://austinpoor.com), built with Go,
Bubble Tea, Bubbles, Lip Gloss, and Glamour. It has Home, About, Blog, and This
Project sections, with searchable posts and a Markdown reader.

## Run locally

Requires Go 1.27.3 (or automatic toolchain downloads) and a local checkout of the
private website repository:

```sh
make run WEBSITE_DIR=../austinpoor-dot-com
```

`make run` and `make build` refresh the blog snapshot before compiling. `make
build` writes `bin/ssh-austinpoor-com`. To refresh content separately:

```sh
go run ./cmd/sync-content -source /path/to/austinpoor-dot-com
go run .
```

Direct `go run .` / `go build .` use the last imported snapshot. A checkout with
no snapshot still runs, but its blog is empty.

## Content source

The importer uses the website checkout's `content/seed.json` for titles,
summaries, dates, and tags, and reads the current article bodies from
`content/blog-posts/<slug>.md`. It includes only metadata-listed posts with a
publication date at or before import time, excluding drafts and underscore
files. Posts are sorted newest first. Invalid or missing published content
fails the import without replacing the previous snapshot.

The website currently reads its live blog from a database. This importer uses
the **local content snapshot**; it does not fetch that database. New posts and
metadata edits need to be reflected in `content/seed.json` as well as the
Markdown files. Existing body edits are picked up directly from disk.

The generated `content/posts.json` is Git-ignored and embedded in the Go binary.
The app needs no website checkout, GitHub token, database, or content network
requests at runtime. For a container build, run `make content` before sending
the build context; `.dockerignore` intentionally includes the generated bundle.

Home, About, and This Project are maintained as Markdown in `content/pages/`.
Home/About were adapted from the website's existing copy; they are not parsed
from its Svelte components. Links resolve against the website, and each post
links back to its web version for images and richer content.

If builds move to CI, check out the private website repository during the build
using a repository-scoped read-only credential, then run the same importer.
Keep that credential in CI; the runtime app only receives the published bundle.

## Controls

- `Tab` / `Shift+Tab`, or `1`–`4`: switch sections.
- Arrow keys or `j` / `k`: select posts or scroll.
- `/`: search post titles, summaries, and tags (case-insensitive; all search words must match).
- `Enter`: apply the search, then open the selected post.
- `Esc`: return to the list, cancel search, or clear the filter.
- `q` / `Ctrl+C`: quit (`q` remains usable as text while searching).

## Validation and deployment status

```sh
go test ./...
go vet ./...
make build WEBSITE_DIR=../austinpoor-dot-com
```

The current entry point is the local TUI. The Wish SSH server was removed during
the restart and still needs to be restored. The Dockerfile still targets the
old `./cmd/server` directory, so the current source is not ready for SSH
deployment. The demo above shows the earlier SSH version.
