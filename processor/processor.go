package processor

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/kupospelov/feeds-to-instapaper/state"
	"github.com/mmcdole/gofeed"
)

var ErrResponseTooLarge = errors.New("feed response exceeds configured size limit")

type Parser interface {
	ParseURL(feedURL string) (*gofeed.Feed, error)
}

type Instapaper interface {
	Add(link, title string) error
}

type Hooks interface {
	NewArticle(feed *gofeed.Feed, item *gofeed.Item)
}

type Processor struct {
	parser         Parser
	instapaper     Instapaper
	hooks          Hooks
	state          *state.State
	dryRun         bool
	maxConcurrency int
	maxItems       int
}

func New(parser Parser, instapaper Instapaper, hooks Hooks, state *state.State, dryRun bool) *Processor {
	return NewWithLimits(parser, instapaper, hooks, state, dryRun, 1, 1000)
}

func NewWithLimits(parser Parser, instapaper Instapaper, hooks Hooks, state *state.State, dryRun bool, maxConcurrency, maxItems int) *Processor {
	if maxConcurrency < 1 {
		maxConcurrency = 1
	}
	if maxItems < 1 {
		maxItems = 1
	}
	return &Processor{
		parser:         parser,
		instapaper:     instapaper,
		hooks:          hooks,
		state:          state,
		dryRun:         dryRun,
		maxConcurrency: maxConcurrency,
		maxItems:       maxItems,
	}
}

// NewFeedParser creates a gofeed parser whose requests have both time and body-size bounds.
func NewFeedParser(timeout time.Duration, maxResponseBytes int64) *gofeed.Parser {
	parser := gofeed.NewParser()
	parser.Client = &http.Client{
		Timeout:   timeout,
		Transport: responseLimitTransport{base: http.DefaultTransport, maxBytes: maxResponseBytes},
	}
	return parser
}

type responseLimitTransport struct {
	base     http.RoundTripper
	maxBytes int64
}

func (t responseLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	resp, err := base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	resp.Body = &limitedReadCloser{ReadCloser: resp.Body, maxBytes: t.maxBytes}
	return resp, nil
}

type limitedReadCloser struct {
	io.ReadCloser
	maxBytes, read int64
}

func (r *limitedReadCloser) Read(p []byte) (int, error) {
	if r.read < r.maxBytes {
		remaining := r.maxBytes - r.read
		if int64(len(p)) > remaining {
			p = p[:remaining]
		}
		n, err := r.ReadCloser.Read(p)
		r.read += int64(n)
		return n, err
	}
	var one [1]byte
	n, err := r.ReadCloser.Read(one[:])
	if n > 0 {
		return 0, fmt.Errorf("%w (%d bytes)", ErrResponseTooLarge, r.maxBytes)
	}
	return 0, err
}

func (p *Processor) ProcessFeeds(feedURLs []string) error {
	if p.dryRun {
		log.Printf("Running in dry run mode")
	}

	type feedItem struct {
		feed *gofeed.Feed
		item *gofeed.Item
	}
	itemsChan := make(chan feedItem)
	jobs := make(chan string)
	var wg sync.WaitGroup

	workerCount := min(p.maxConcurrency, len(feedURLs))
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range jobs {
				feed, err := p.parser.ParseURL(url)
				if err != nil {
					log.Printf("Error parsing feed %s: %v", url, err)
					continue
				}
				if len(feed.Items) > p.maxItems {
					log.Printf("Rejecting feed %s: contains %d items, limit is %d", url, len(feed.Items), p.maxItems)
					continue
				}

				for _, item := range feed.Items {
					if p.state.MarkProcessed(item.Link) {
						itemsChan <- feedItem{feed, item}
					}
				}
			}
		}()
	}
	go func() {
		for _, feedURL := range feedURLs {
			jobs <- feedURL
		}
		close(jobs)
		wg.Wait()
		close(itemsChan)
	}()

	feedItems := make([]feedItem, 0)
	for item := range itemsChan {
		feedItems = append(feedItems, item)
	}
	slices.SortFunc(feedItems, func(a, b feedItem) int {
		var atime time.Time
		if a.item.PublishedParsed != nil {
			atime = *a.item.PublishedParsed
		}
		var btime time.Time
		if b.item.PublishedParsed != nil {
			btime = *b.item.PublishedParsed
		}
		return atime.Compare(btime)
	})

	for _, fi := range feedItems {
		log.Printf("New link: %s", fi.item.Link)
		if p.dryRun {
			continue
		}

		err := p.instapaper.Add(fi.item.Link, fi.item.Title)
		if err != nil {
			log.Printf("Error adding link %s: %v", fi.item.Link, err)
			continue
		}

		p.hooks.NewArticle(fi.feed, fi.item)
		p.state.Append(fi.item.Link)
	}

	return nil
}
