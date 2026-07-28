# feeds-to-instapaper

An application that checks RSS, Atom, or JSON feeds and adds new articles to Instapaper.

## Installation

```bash
go install github.com/kupospelov/feeds-to-instapaper@latest
```

## Configuration

Create a configuration file at `~/.config/feeds-to-instapaper/config.toml` with
owner-only permissions (the application rejects files readable by other users):

```bash
install -d -m 700 ~/.config/feeds-to-instapaper
install -m 600 /dev/null ~/.config/feeds-to-instapaper/config.toml
```

Then edit it:

```toml
[instapaper]
username = "your-instapaper-email"
password = "your-instapaper-password"

[feeds]
urls = [
    "https://example.com/feed.xml",
    "https://another-site.com/atom",
]
max_concurrency = 4
request_timeout_seconds = 30
max_response_bytes = 10485760
max_items = 1000
```

All feed limits are optional and shown with their defaults. Each must be at
least 1. `max_concurrency` limits simultaneous feed downloads,
`request_timeout_seconds` bounds each download, `max_response_bytes` limits a
single feed response, and `max_items` rejects an entire feed with too many
items. Limits cannot be disabled.

## Building

Dependencies:
* Go
* make
* scdoc (optional, for man pages)

Run `make`.

## Usage

Run `feeds-to-instapaper`.

You can use cron or systemd timers to schedule the runs. Check out the [examples](https://github.com/kupospelov/feeds-to-instapaper/tree/main/doc/systemd).

The application writes a state file at `~/.local/state/feeds-to-instapaper/added` to keep track of previously processed articles. Its directory and file are enforced as owner-only (`0700` and `0600`).
