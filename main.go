package main

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"syscall"
	"text/template"

	_ "modernc.org/sqlite"

	"github.com/scipunch/myfeed/config"
	"github.com/scipunch/myfeed/fetcher"
	"github.com/scipunch/myfeed/parser"
	"github.com/scipunch/myfeed/parser/factory"
)

//go:embed schema.sql
var ddl string

type Newsletter struct {
	Title string
	Pages []Page
}

type Page struct {
	Title   string
	Content string
}

func main() {
	// TODO: Use embedded templates
	t := template.Must(template.ParseGlob("templates/*.html"))

	if os.Getenv("DEBUG") != "" {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	}

	var cfgPath string
	flag.StringVar(&cfgPath, "config", config.DefaultPath(), "path to a TOML config")
	flag.Parse()

	// Read config and create if default is missing
	conf, err := config.Read(cfgPath)
	if errors.Is(err, os.ErrNotExist) && cfgPath == config.DefaultPath() {
		if err := config.Write(cfgPath, conf); err != nil {
			log.Fatalf("failed to write default config with %s", err)
		}
	} else if err != nil {
		log.Fatalf("failed to read config with %s", err)
	}
	var parserTypes []parser.Type
	for _, r := range conf.Resources {
		parserTypes = append(parserTypes, r.ParserT)
	}
	parsers, err := factory.Init(parserTypes)
	if err != nil {
		log.Fatalf("failed to initialize some parsers with %s", err)
	}

	// Connect to database & initialize schema
	dbBasePath := path.Dir(conf.DatabasePath)
	err = os.MkdirAll(dbBasePath, os.ModePerm)
	if err != nil {
		log.Fatalf("failed to create base shared directory at '%s' with %s", dbBasePath, err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	_, err = initDB(ctx, conf.DatabasePath)
	if err != nil {
		log.Fatalf("failed to initialize database schema with %v", err)
	}

	// Initialize fetchers
	var resourceTypes []config.ResourceType
	for _, r := range conf.Resources {
		resourceTypes = append(resourceTypes, r.T)
	}
	configDir := path.Dir(cfgPath)
	fetchers, err := fetcher.GetFetchers(resourceTypes, configDir)
	if err != nil {
		log.Fatalf("failed to initialize fetchers with %s", err)
	}

	// Fetch configured feeds
	var errs []error
	feeds := make([]*fetcher.Feed, len(conf.Resources))
	for i, resource := range conf.Resources {
		f := fetchers[resource.T]
		feed, err := f.Fetch(ctx, resource.FeedURL)
		if err != nil {
			errs = append(errs, fmt.Errorf("'%s' fetch failed with %w", resource.FeedURL, err))
			continue
		}
		feeds[i] = &feed
	}
	slog.Info("fetched feeds", "amount", len(feeds))
	if len(errs) > 0 {
		slog.Error("several feeds were not parsed", "feeds", errors.Join(errs...))
	}

	// Process new items
	errs = nil
	newsletter := Newsletter{Title: "Test newsletter"}
	for i, feed := range feeds {
		if feed == nil {
			slog.Debug("skipping failed to parse feed")
			continue
		}
		resource := conf.Resources[i]
		p := parsers[resource.ParserT]
		for _, item := range feed.Items {
			// For Telegram messages, pass the description (message content) instead of the link
			// For other parsers (web, youtube), pass the link to fetch and parse
			var parseInput string
			if resource.ParserT == parser.Telegram {
				parseInput = item.Description
			} else {
				parseInput = item.Link
			}

			data, err := p.Parse(parseInput)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			slog.Info("feed item parsed", "length", len(data.String()))
			newsletter.Pages = append(newsletter.Pages, Page{
				Title:   item.Title,
				Content: data.String(),
			})
		}
	}
	slog.Info("newsletter content fetched", "pages", len(newsletter.Pages))
	if len(errs) > 0 {
		slog.Error("failed to parse some pages", "errors", errors.Join(errs...).Error())
	}

	// TODO: Generate PDF report
	out, err := os.Create("index.html")
	if err != nil {
		log.Fatal("could not create newsletter HTML file", err)
	}
	defer out.Close()
	err = t.Execute(out, newsletter)
	if err != nil {
		log.Fatal("could not convert newsletter into HTML", err)
	}
}

func initDB(ctx context.Context, source string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", source)
	if err != nil {
		return nil, fmt.Errorf("failed to open database at '%s' with %w", source, err)
	}
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return nil, fmt.Errorf("failed to execute DDL with %w", err)
	}
	return db, nil
}
