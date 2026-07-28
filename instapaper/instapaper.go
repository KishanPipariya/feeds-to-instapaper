package instapaper

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxErrorResponseBytes = 64 * 1024

type Instapaper struct {
	username string
	password string
	client   *http.Client
}

func New(username, password string) *Instapaper {
	return &Instapaper{
		username: username,
		password: password,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (api *Instapaper) Add(link, title string) error {
	apiURL := "https://www.instapaper.com/api/add"
	formData := fmt.Sprintf("username=%s&password=%s&url=%s",
		url.QueryEscape(api.username),
		url.QueryEscape(api.password),
		url.QueryEscape(link))
	if title != "" {
		formData += "&title=" + url.QueryEscape(title)
	}

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(formData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := api.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, truncated, err := readErrorBody(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		if truncated {
			return fmt.Errorf("Instapaper API error (status %d): %s [truncated]", resp.StatusCode, string(body))
		}
		return fmt.Errorf("Instapaper API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func readErrorBody(body io.Reader) ([]byte, bool, error) {
	limited := io.LimitReader(body, maxErrorResponseBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, false, err
	}
	if len(data) > maxErrorResponseBytes {
		return data[:maxErrorResponseBytes], true, nil
	}
	return data, false, nil
}
