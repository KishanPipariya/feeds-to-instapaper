# feeds-to-instapaper

An application that checks RSS, Atom, or JSON feeds and adds new articles to Instapaper.

## Installation

Requires Go 1.25 or newer.

```bash
go install github.com/kupospelov/feeds-to-instapaper@latest
```

## Configuration

Create a configuration file at `$XDG_CONFIG_HOME/feeds-to-instapaper/config.toml`
(or `~/.config/feeds-to-instapaper/config.toml` when `XDG_CONFIG_HOME` is not
set) with owner-only permissions. The application rejects a non-regular file
or one readable by group or other users:

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
max_items = 20
```

All feed limits are optional and shown with their defaults. Each must be at
least 1. `max_concurrency` limits simultaneous feed downloads,
`request_timeout_seconds` bounds each download, `max_response_bytes` limits a
single feed response, and `max_items` limits each feed to its most recently
published items before any are sent to Instapaper or recorded in state.
Limits cannot be disabled.

If you already have a configuration file, secure it with:

```bash
chmod 600 ~/.config/feeds-to-instapaper/config.toml
```

## Building

Dependencies:
* Go
* make
* scdoc (optional, for man pages)

Run `make`.

## Usage

Run `feeds-to-instapaper`.

You can use cron or systemd timers to schedule the runs. Check out the [examples](https://github.com/kupospelov/feeds-to-instapaper/tree/main/doc/systemd).

The application writes a state file at
`$XDG_STATE_HOME/feeds-to-instapaper/added` (or
`~/.local/state/feeds-to-instapaper/added` when unset) to keep track of
previously processed articles. Its directory and file are enforced as
owner-only (`0700` and `0600`).
