package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validConfig = `[instapaper]
username = "user"
password = "password"

[feeds]
urls = ["https://example.com/feed.xml"]
`

func writeConfig(t *testing.T, mode os.FileMode, contents string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "feeds-to-instapaper")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(contents), mode); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(dir))
	return path
}

func TestLoadDefaultsFeedLimits(t *testing.T) {
	writeConfig(t, 0600, validConfig)
	config, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if config.Feeds.MaxConcurrency != DefaultMaxConcurrency || config.Feeds.RequestTimeoutSeconds != DefaultRequestTimeoutSeconds || config.Feeds.MaxResponseBytes != DefaultMaxResponseBytes || config.Feeds.MaxItems != DefaultMaxItems {
		t.Fatalf("feed defaults were not applied: %#v", config.Feeds)
	}
}

func TestLoadRejectsInsecureOrNonRegularConfig(t *testing.T) {
	path := writeConfig(t, 0644, validConfig)
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "chmod 600") {
		t.Fatalf("expected chmod 600 error, got %v", err)
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	_, err = Load()
	if err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("expected regular file error, got %v", err)
	}
}

func TestLoadRejectsInvalidFeedLimits(t *testing.T) {
	for _, key := range []string{"max_concurrency", "request_timeout_seconds", "max_response_bytes", "max_items"} {
		t.Run(key, func(t *testing.T) {
			contents := strings.Replace(validConfig, "urls = [\"https://example.com/feed.xml\"]", "urls = [\"https://example.com/feed.xml\"]\n"+key+" = 0", 1)
			writeConfig(t, 0600, contents)
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), "must be at least 1") {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
}
