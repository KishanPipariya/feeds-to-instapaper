package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kupospelov/feeds-to-instapaper/config"
	"github.com/kupospelov/feeds-to-instapaper/hooks"
	"github.com/kupospelov/feeds-to-instapaper/instapaper"
	"github.com/kupospelov/feeds-to-instapaper/processor"
	"github.com/kupospelov/feeds-to-instapaper/state"
)

func scheduleSave(state *state.State) func() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	save := func() {
		err := state.Save()
		if err != nil {
			log.Fatalf("Failed to save state: %v", err)
		}
	}

	var saveOnce sync.Once
	go func() {
		<-c
		saveOnce.Do(save)
		os.Exit(0)
	}()
	return func() {
		saveOnce.Do(save)
	}
}

func main() {
	log.SetFlags(0)

	var dryRun = flag.Bool("dry-run", false, "Only fetch and print new feeds")
	flag.Parse()

	config, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	state, err := state.Load()
	if err != nil {
		log.Fatalf("Failed to load state: %v", err)
	}

	if !*dryRun {
		save := scheduleSave(state)
		defer save()
	}

	parser := processor.NewFeedParser(time.Duration(config.Feeds.RequestTimeoutSeconds)*time.Second, config.Feeds.MaxResponseBytes)
	instapaper := instapaper.New(config.Instapaper.Username, config.Instapaper.Password)

	hooks, err := hooks.New(config.Hooks)
	if err != nil {
		log.Fatalf("Failed to create hooks: %v", err)
	}

	proc := processor.NewWithLimits(parser, instapaper, hooks, state, *dryRun, config.Feeds.MaxConcurrency, config.Feeds.MaxItems)

	err = proc.ProcessFeeds(config.Feeds.URLs)
	if err != nil {
		log.Fatalf("Error processing feeds: %v", err)
	}
}
