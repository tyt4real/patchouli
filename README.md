
# Patchy

Patchy is a small, self-hosted imageboard / crawler web application written in Go. It provides a lightweight server, a web UI, and a crawler component for fetching and organizing images into local storage.

## Features

- Minimal, dependency-light Go codebase
- Built-in web UI and API handlers
- Crawler for automated image collection
- Local storage under `data/` and a SQLite database in `database/`

## What imageboards does it work on?

For now it should work on all vichan imageboards which have not changed the actual html rendering on their sites, and which provide the API. The posts are fetched via the HTML because some imageboards like the party do not provide posts as JSON.

## Quick Start

Prerequisites:

- Go 1.18+ installed
- Whatever dependencies there are

Build:

```bash
go build -o bin/patchy .
```

Run (binary):

```bash
./bin/patchy
```

Or run directly with Go:

```bash
go run main.go
```

The web UI is served by the server; see the console output for the listening address (defaults may be configured in `patchy.json`).

## Configuration

Default runtime options are stored in `patchy.json`. Edit that file to change crawler settings.

## Development

Run the web UI locally during development:

```bash
go run cmd/webui/main.go
```

Run the crawler (if separate executable is present):

```bash
go run crawler/crawler.go
```

Logs are printed to stdout; use your shell or a process manager for background running.