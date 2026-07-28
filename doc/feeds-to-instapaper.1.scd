feeds-to-instapaper(1)

# NAME

feeds-to-instapaper - upload feeds to Instapaper

# SYNOPSIS

*feeds-to-instapaper* [options...]

# DESCRIPTION

feeds-to-instapaper is a command-line tool that reads a list of RSS/Atom
feeds, finds new articles, and uploads them to Instapaper.

It keeps track of processed articles to avoid duplicates. The state is stored
in "$XDG_STATE_HOME/feeds-to-instapaper/added" (or
"~/.local/state/feeds-to-instapaper/added" if unset).

# OPTIONS

*-dry-run*
	Only fetch and print new feeds.

# CONFIGURATION

The configuration file is located at
"$XDG_CONFIG_HOME/feeds-to-instapaper/config.toml" (or
"~/.config/feeds-to-instapaper/config.toml" if unset).

Example:

```
[instapaper]
username = "your-email"
password = "your-password"

[feeds]
urls = [
  "https://example.com/feed.xml",
  "https://another.org/rss.xml"
]
max_concurrency = 4
request_timeout_seconds = 30
max_response_bytes = 10485760
max_items = 1000

[[hooks.new_article]]
spawn = ["notify-send", "New article on {{.Feed.Title}}", "{{.Title}}"]
```

The following sections are supported:

*[instapaper]*
	Instapaper credentials.

	*username* <string>
		Your Instapaper email. This is required.

	*password* <string>
		Your Instapaper password. This is required.

*[feeds]*
	Feeds to process.

	*urls* <array of strings>
		A list of RSS/Atom feed URLs to process.

	*max_concurrency* <integer>
		Maximum number of feeds fetched concurrently. Defaults to 4.

	*request_timeout_seconds* <integer>
		Maximum duration of an individual feed request. Defaults to 30.

	*max_response_bytes* <integer>
		Maximum response size allowed for an individual feed. Defaults to
		10485760 (10 MiB).

	*max_items* <integer>
		Maximum number of items allowed in an individual feed. Defaults to
		1000. Feeds exceeding this limit are rejected entirely.

*[hooks]*
	Hooks to run on events.

	*new_article* <array of tables>
		A list of hooks to run when a new article is found. Each hook is a
		table with a _spawn_ key.

		*spawn* <array of strings>
			A command and its arguments to execute. The arguments are
			interpreted as Go templates. See *TEMPLATES* for details.

# TEMPLATES

Hooks can use Go templates in their arguments. For full documentation on the
template syntax, see <https://pkg.go.dev/text/template>.

The following data is available in _new_article_ hooks:

*.Title*
	The article title.

*.Feed.Title*
	The feed title of the article.

*.Feed.Link*
	The feed link of the article.

# AUTHORS

Konstantin Pospelov <kupospelov@gmail.com>
