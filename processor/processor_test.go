package processor

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kupospelov/feeds-to-instapaper/state"
	"github.com/mmcdole/gofeed"
)

type parser struct {
	feeds map[string]*gofeed.Feed
}

type trackingParser struct {
	active    int32
	maxActive int32
	feed      *gofeed.Feed
}

func (p *trackingParser) ParseURL(string) (*gofeed.Feed, error) {
	active := atomic.AddInt32(&p.active, 1)
	for {
		max := atomic.LoadInt32(&p.maxActive)
		if active <= max || atomic.CompareAndSwapInt32(&p.maxActive, max, active) {
			break
		}
	}
	time.Sleep(20 * time.Millisecond)
	atomic.AddInt32(&p.active, -1)
	return p.feed, nil
}

func (p *parser) ParseURL(feedURL string) (*gofeed.Feed, error) {
	feed, ok := p.feeds[feedURL]
	if ok {
		return feed, nil
	}

	return nil, fmt.Errorf("not found")
}

type instapaper struct {
	addedItems []addedItem
	err        map[string]error
}

type addedItem struct {
	link  string
	title string
}

func (i *instapaper) Add(link, title string) error {
	err := i.err[link]
	if err != nil {
		return err
	}

	i.addedItems = append(i.addedItems, addedItem{link: link, title: title})
	return nil
}

func (i *instapaper) assertAddedItems(t *testing.T, expected []addedItem) {
	if got, want := len(i.addedItems), len(expected); got != want {
		t.Errorf("len(addedItems)=%d, want=%d", got, want)
	}
	for j := range len(expected) {
		if got, want := i.addedItems[j].link, expected[j].link; got != want {
			t.Errorf("addedItems[%d].link=%s, want=%s", j, got, want)
		}
		if got, want := i.addedItems[j].title, expected[j].title; got != want {
			t.Errorf("addedItems[%d].title=%s, want=%s", j, got, want)
		}
	}
}

type hooks struct {
	newArticles []newArticle
}

type newArticle struct {
	feedTitle string
	itemTitle string
}

func (h *hooks) NewArticle(feed *gofeed.Feed, item *gofeed.Item) {
	h.newArticles = append(h.newArticles, newArticle{feedTitle: feed.Title, itemTitle: item.Title})
}

func (h *hooks) assertNewArticles(t *testing.T, expected []newArticle) {
	if got, want := len(h.newArticles), len(expected); got != want {
		t.Errorf("len(newArticles)=%d, want=%d", got, want)
	}
	for i := range len(expected) {
		if got, want := h.newArticles[i].feedTitle, expected[i].feedTitle; got != want {
			t.Errorf("newArticles[%d].feedTitle=%s, want=%s", i, got, want)
		}
		if got, want := h.newArticles[i].itemTitle, expected[i].itemTitle; got != want {
			t.Errorf("newArticles[%d].itemTitle=%s, want=%s", i, got, want)
		}
	}
}

func parseTime(t string) *time.Time {
	timestamp, _ := time.Parse(time.Kitchen, t)
	return &timestamp
}

func createParser(feeds []*gofeed.Feed) Parser {
	p := &parser{
		feeds: make(map[string]*gofeed.Feed),
	}
	for _, feed := range feeds {
		p.feeds[feed.Link] = feed
	}
	return p
}

func assertNewStateItems(t *testing.T, s *state.State, expected []string) {
	if got, want := len(s.NewItems), len(expected); got != want {
		t.Errorf("len(newItems)=%d, want=%d", got, want)
	}
	for i := range len(expected) {
		if got, want := s.NewItems[i], expected[i]; got != want {
			t.Errorf("newItems[%d]=%s, want=%s", i, got, want)
		}
	}
}

func TestSuccess(t *testing.T) {
	feeds := []*gofeed.Feed{
		{
			Title: "Feed 1 Title",
			Link:  "http://example.com",
			Items: []*gofeed.Item{
				{Link: "http://example.com/1", Title: "Article 1", PublishedParsed: parseTime("3:00PM")},
				{Link: "http://example.com/3", Title: "Article 3"},
				{Link: "http://example.com/2", Title: "Article 2", PublishedParsed: parseTime("3:02PM")},
			},
		},
	}
	testParser := createParser(feeds)
	testInstapaper := &instapaper{}
	testState := state.EmptyWithPath("test")
	testHooks := &hooks{}
	processor := New(testParser, testInstapaper, testHooks, testState, false)

	err := processor.ProcessFeeds([]string{"http://example.com"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	testInstapaper.assertAddedItems(t, []addedItem{
		{"http://example.com/1", "Article 1"},
		{"http://example.com/2", "Article 2"},
		{"http://example.com/3", "Article 3"},
	})
	testHooks.assertNewArticles(t, []newArticle{
		{"Feed 1 Title", "Article 1"},
		{"Feed 1 Title", "Article 2"},
		{"Feed 1 Title", "Article 3"},
	})
	assertNewStateItems(t, testState, []string{
		"http://example.com/1",
		"http://example.com/2",
		"http://example.com/3",
	})
}

func TestSkipProcessed(t *testing.T) {
	feeds := []*gofeed.Feed{
		{
			Title: "Feed 1 Title",
			Link:  "http://example.com",
			Items: []*gofeed.Item{
				{Link: "http://example.com/1", Title: "Already processed"},
				{Link: "http://example.com/2", Title: "New article"},
			},
		},
	}
	testParser := createParser(feeds)
	testInstapaper := &instapaper{}
	testState := state.EmptyWithPath("test")
	testState.MarkProcessed("http://example.com/1")
	testHooks := &hooks{}
	processor := New(testParser, testInstapaper, testHooks, testState, false)

	err := processor.ProcessFeeds([]string{"http://example.com"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	testInstapaper.assertAddedItems(t, []addedItem{
		{"http://example.com/2", "New article"},
	})
	testHooks.assertNewArticles(t, []newArticle{
		{"Feed 1 Title", "New article"},
	})
	assertNewStateItems(t, testState, []string{
		"http://example.com/2",
	})
}

func TestParserError(t *testing.T) {
	feeds := []*gofeed.Feed{
		{
			Title: "Feed 1 Title",
			Link:  "http://example.com",
			Items: []*gofeed.Item{
				{Link: "http://example.com/1", Title: "Article 1"},
			},
		},
	}
	testParser := createParser(feeds)
	testInstapaper := &instapaper{}
	testState := state.EmptyWithPath("test")
	testHooks := &hooks{}
	processor := New(testParser, testInstapaper, testHooks, testState, false)

	err := processor.ProcessFeeds([]string{"http://example.com", "http://error.com"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	testInstapaper.assertAddedItems(t, []addedItem{
		{"http://example.com/1", "Article 1"},
	})
	testHooks.assertNewArticles(t, []newArticle{
		{"Feed 1 Title", "Article 1"},
	})
	assertNewStateItems(t, testState, []string{
		"http://example.com/1",
	})
}

func TestInstapaperError(t *testing.T) {
	feeds := []*gofeed.Feed{
		{
			Title: "Feed 1 Title",
			Link:  "http://example.com",
			Items: []*gofeed.Item{
				{Link: "http://example.com/1", Title: "Article 1"},
				{Link: "http://example.com/2", Title: "Article 2"},
			},
		},
	}
	testParser := createParser(feeds)
	testInstapaper := &instapaper{err: map[string]error{"http://example.com/1": errors.New("API error")}}
	testState := state.EmptyWithPath("test")
	testHooks := &hooks{}
	processor := New(testParser, testInstapaper, testHooks, testState, false)

	err := processor.ProcessFeeds([]string{"http://example.com"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	testInstapaper.assertAddedItems(t, []addedItem{
		{"http://example.com/2", "Article 2"},
	})
	testHooks.assertNewArticles(t, []newArticle{
		{"Feed 1 Title", "Article 2"},
	})
	assertNewStateItems(t, testState, []string{
		"http://example.com/2",
	})
}

func TestDryRun(t *testing.T) {
	feeds := []*gofeed.Feed{
		{
			Title: "Feed 1 Title",
			Link:  "http://example.com",
			Items: []*gofeed.Item{
				{Link: "http://example.com/1", Title: "Article 1"},
				{Link: "http://example.com/2", Title: "Article 2"},
			},
		},
	}
	testParser := createParser(feeds)
	testInstapaper := &instapaper{}
	testState := state.EmptyWithPath("test")
	testState.MarkProcessed("http://example.com/1")
	testHooks := &hooks{}
	processor := New(testParser, testInstapaper, testHooks, testState, true)

	err := processor.ProcessFeeds([]string{"http://example.com"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	testInstapaper.assertAddedItems(t, []addedItem{})
	testHooks.assertNewArticles(t, []newArticle{})
	assertNewStateItems(t, testState, []string{})
	if testState.MarkProcessed("http://example.com/2") {
		t.Errorf("Expected 'http://example.com/2' to have been marked as processed")
	}
}

func TestProcessFeedsCapsConcurrency(t *testing.T) {
	testParser := &trackingParser{feed: &gofeed.Feed{Items: []*gofeed.Item{}}}
	testState := state.EmptyWithPath("test")
	proc := NewWithLimits(testParser, &instapaper{}, &hooks{}, testState, false, 2, 1000)
	if err := proc.ProcessFeeds([]string{"1", "2", "3", "4", "5"}); err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(&testParser.maxActive); got > 2 {
		t.Fatalf("concurrency = %d, want at most 2", got)
	}
}

func TestProcessFeedsKeepsMostRecentItemsFromOversizedBatch(t *testing.T) {
	feed := &gofeed.Feed{Link: "feed", Items: []*gofeed.Item{
		{Link: "one", PublishedParsed: parseTime("1:00PM")},
		{Link: "three", PublishedParsed: parseTime("3:00PM")},
		{Link: "two", PublishedParsed: parseTime("2:00PM")},
	}}
	testState := state.EmptyWithPath("test")
	testInstapaper := &instapaper{}
	proc := NewWithLimits(createParser([]*gofeed.Feed{feed}), testInstapaper, &hooks{}, testState, false, 1, 2)
	if err := proc.ProcessFeeds([]string{"feed"}); err != nil {
		t.Fatal(err)
	}
	testInstapaper.assertAddedItems(t, []addedItem{{link: "two"}, {link: "three"}})
	assertNewStateItems(t, testState, []string{"two", "three"})
	if !testState.MarkProcessed("one") {
		t.Fatal("older item was marked processed")
	}
}

func TestFeedParserTimeoutAndResponseLimit(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.Write([]byte("<rss version=\"2.0\"><channel></channel></rss>"))
		}))
		defer server.Close()
		_, err := NewFeedParser(10*time.Millisecond, 1024).ParseURL(server.URL)
		if err == nil {
			t.Fatal("expected timeout")
		}
	})
	t.Run("response limit", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(strings.Repeat("x", 1024)))
		}))
		defer server.Close()
		client := &http.Client{Transport: responseLimitTransport{base: http.DefaultTransport, maxBytes: 64}}
		response, err := client.Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		_, err = io.ReadAll(response.Body)
		if err == nil || !errors.Is(err, ErrResponseTooLarge) {
			t.Fatalf("expected response-size error, got %v", err)
		}
	})
}
