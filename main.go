package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"text/template"

	_ "modernc.org/sqlite"

	"github.com/playwright-community/playwright-go"
	"github.com/scipunch/myfeed/agent"
	"github.com/scipunch/myfeed/cache"
	"github.com/scipunch/myfeed/config"
	"github.com/scipunch/myfeed/fetcher"
	"github.com/scipunch/myfeed/filter"
	"github.com/scipunch/myfeed/parser"
	"github.com/scipunch/myfeed/parser/factory"
)

//go:embed schema.sql
var ddl string

type Newsletter struct {
	Title     string
	Resources []Resource
}

type Resource struct {
	Name  string
	Pages []Page
}

type Page struct {
	Title   string
	Link    string
	Content string
	ID      string // Unique ID for anchor links
}

func main() {
	// TODO: Use embedded templates
	t := template.Must(template.ParseGlob("templates/*.html"))

	if os.Getenv("DEBUG") != "" {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	}

	var cfgPath string
	var cleanCache bool
	flag.StringVar(&cfgPath, "config", config.DefaultPath(), "path to a TOML config")
	flag.BoolVar(&cleanCache, "clean", false, "remove all cache entries")
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

	// Load credentials
	credPath := config.DefaultCredentialsPath()
	creds, err := config.ReadCredentials(credPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("failed to read credentials: %s", err)
	}

	// Initialize filter pipeline
	filterPipeline, err := filter.NewFilterPipeline(conf.Filters)
	if err != nil {
		log.Fatalf("failed to initialize filters: %s", err)
	}
	if len(conf.Filters) > 0 {
		slog.Info("initialized filters", "count", len(conf.Filters))
	}

	var parserTypes []parser.Type
	for _, r := range conf.Resources {
		if r.IsEnabled() {
			parserTypes = append(parserTypes, r.ParserT)
		}
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

	// Initialize database (includes both main and cache schemas)
	db, err := initDB(ctx, conf.DatabasePath)
	if err != nil {
		log.Fatalf("failed to initialize database schema with %v", err)
	}
	defer db.Close()

	// Initialize cache using the shared database connection
	cacheDB, err := cache.NewCacheFromDB(db)
	if err != nil {
		log.Fatalf("failed to initialize cache: %v", err)
	}

	// Handle -clean flag
	if cleanCache {
		if err := cacheDB.Clear(); err != nil {
			log.Fatalf("failed to clear cache: %v", err)
		}
		slog.Info("cache cleared successfully")
		return
	}

	// Show cache stats
	stats, err := cacheDB.Stats()
	if err != nil {
		slog.Warn("failed to get cache stats", "error", err)
	} else {
		slog.Info("cache initialized",
			"parser_entries", stats.ParserEntries,
			"agent_entries", stats.AgentEntries)
	}

	// Initialize agents if any resource requires them
	agentTypes := agent.CollectUniqueAgentTypes(conf.Resources)
	var agents map[string]agent.Agent
	if len(agentTypes) > 0 {
		// Validate Gemini credentials
		if !creds.Gemini.IsValid() {
			log.Fatal("Gemini API key and model required for agents but not found in creds.toml")
		}

		// Initialize agents with fail-fast validation
		agents, err = agent.InitAgents(ctx, agentTypes, creds.Gemini)
		if err != nil {
			log.Fatalf("failed to initialize agents: %s", err)
		}
		slog.Info("initialized agents", "types", agentTypes)
	}

	// Initialize fetchers
	var resourceTypes []config.ResourceType
	for _, r := range conf.Resources {
		if r.IsEnabled() {
			resourceTypes = append(resourceTypes, r.T)
		}
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
		// Skip disabled resources
		if !resource.IsEnabled() {
			slog.Debug("skipping disabled resource", "url", resource.FeedURL)
			continue
		}

		// Check for cancellation before fetching
		select {
		case <-ctx.Done():
			slog.Info("interrupted by user during fetch, exiting gracefully")
			return
		default:
		}

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
	resourceMap := make(map[int]*Resource) // Map index to resource
	for i, feed := range feeds {
		// Check if context was cancelled
		select {
		case <-ctx.Done():
			slog.Info("interrupted by user, exiting gracefully")
			return
		default:
		}

		if feed == nil {
			slog.Debug("skipping failed to parse feed")
			continue
		}
		resource := conf.Resources[i]
		p := parsers[resource.ParserT]
		for _, item := range feed.Items {
			// Check for cancellation before processing each item
			select {
			case <-ctx.Done():
				slog.Info("interrupted by user, exiting gracefully")
				return
			default:
			}

			// Apply filters
			if len(resource.FilterNames) > 0 {
				shouldInclude, reason := filterPipeline.ShouldInclude(item, resource.FilterNames)
				if !shouldInclude {
					slog.Debug("item filtered out", "title", item.Title, "reason", reason, "url", item.Link)
					continue
				}
			}

			var content string
			var parsedData parser.Response
			cacheHit := false

			// Step 1: Check agent cache first (if agents configured)
			if len(resource.Agents) > 0 {
				if cached, hit, err := cacheDB.GetAgentOutput(item.Link, string(resource.ParserT), resource.Agents); err == nil && hit {
					content = cached
					cacheHit = true
					slog.Debug("agent cache hit", "url", item.Link, "agents", resource.Agents)
				}
			}

			// Step 2: If no agent cache, try parser cache
			if !cacheHit {
				if cached, hit, err := cacheDB.GetParserOutput(item.Link, string(resource.ParserT)); err == nil && hit {
					// Deserialize cached parser output
					if data, err := cache.DeserializeParserResponse(string(resource.ParserT), cached); err == nil {
						parsedData = data
						slog.Debug("parser cache hit", "url", item.Link, "parser", resource.ParserT)
					} else {
						slog.Warn("failed to deserialize cached parser output", "error", err)
						// Fall through to re-parse
					}
				}

				// Step 3: If no parser cache, parse now
				if parsedData == nil {
					data, err := p.Parse(item)
					if err != nil {
						errs = append(errs, err)
						continue
					}
					parsedData = data
					slog.Info("feed item parsed", "url", item.Link, "length", len(data.String()))

					// Cache parser output
					if serialized, err := cache.SerializeParserResponse(string(resource.ParserT), parsedData); err == nil {
						if err := cacheDB.SetParserOutput(item.Link, string(resource.ParserT), serialized); err != nil {
							slog.Warn("failed to cache parser output", "error", err)
						}
					} else {
						slog.Warn("failed to serialize parser output", "error", err)
					}
				}

				content = parsedData.String()

				// Step 4: Apply agents if configured
				if len(resource.Agents) > 0 {
					for _, agentName := range resource.Agents {
						agentInstance, ok := agents[agentName]
						if !ok {
							errs = append(errs, fmt.Errorf("agent '%s' not found", agentName))
							continue
						}

						processed, err := agentInstance.Process(ctx, content)
						if err != nil {
							errs = append(errs, fmt.Errorf("agent '%s' processing failed: %w", agentName, err))
							slog.Error("agent processing failed, using original content", "agent", agentName, "error", err)
							// Continue with original content on error
							break
						}

						content = processed
						slog.Info("content processed by agent", "agent", agentName, "original_length", len(parsedData.String()), "processed_length", len(content))
					}

					// Cache final agent output
					if err := cacheDB.SetAgentOutput(item.Link, string(resource.ParserT), resource.Agents, content); err != nil {
						slog.Warn("failed to cache agent output", "error", err)
					}
				}
			}

			// Generate unique ID for anchor link
			hash := sha256.Sum256([]byte(item.Link))
			pageID := hex.EncodeToString(hash[:8])

			// Get or create resource for this feed
			res, exists := resourceMap[i]
			if !exists {
				res = &Resource{
					Name:  conf.Resources[i].FeedURL,
					Pages: []Page{},
				}
				resourceMap[i] = res
			}

			res.Pages = append(res.Pages, Page{
				Title:   item.Title,
				Link:    item.Link,
				Content: content,
				ID:      pageID,
			})
		}
	}
	// Convert resource map to slice in order
	for i := 0; i < len(feeds); i++ {
		if res, exists := resourceMap[i]; exists && len(res.Pages) > 0 {
			newsletter.Resources = append(newsletter.Resources, *res)
		}
	}

	totalPages := 0
	for _, res := range newsletter.Resources {
		totalPages += len(res.Pages)
	}
	slog.Info("newsletter content fetched", "resources", len(newsletter.Resources), "pages", totalPages)
	if len(errs) > 0 {
		slog.Error("failed to parse some pages", "errors", errors.Join(errs...).Error())
	}

	// Generate HTML report
	out, err := os.Create("index.html")
	if err != nil {
		log.Fatal("could not create newsletter HTML file", err)
	}
	defer out.Close()
	err = t.Execute(out, newsletter)
	if err != nil {
		log.Fatal("could not convert newsletter into HTML", err)
	}
	slog.Info("HTML file generated", "path", "index.html")

	// Generate PDF report
	if err := generatePDF(ctx, "index.html", "newsletter.pdf"); err != nil {
		slog.Error("failed to generate PDF", "error", err)
	} else {
		slog.Info("PDF file generated", "path", "newsletter.pdf")
	}
}

func initDB(ctx context.Context, source string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", source)
	if err != nil {
		return nil, fmt.Errorf("failed to open database at '%s' with %w", source, err)
	}

	// Initialize schema (includes main feed table and cache tables)
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return nil, fmt.Errorf("failed to execute DDL with %w", err)
	}

	return db, nil
}

func generatePDF(ctx context.Context, htmlPath, pdfPath string) error {
	// Install playwright if needed
	err := playwright.Install()
	if err != nil {
		return fmt.Errorf("could not install playwright: %w", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("could not start playwright: %w", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch()
	if err != nil {
		return fmt.Errorf("could not launch browser: %w", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		return fmt.Errorf("could not create page: %w", err)
	}
	defer page.Close()

	// Get absolute path to HTML file
	absPath, err := filepath.Abs(htmlPath)
	if err != nil {
		return fmt.Errorf("could not get absolute path: %w", err)
	}

	// Navigate to local HTML file
	fileURL := fmt.Sprintf("file://%s", absPath)
	if _, err = page.Goto(fileURL); err != nil {
		return fmt.Errorf("could not navigate to HTML file: %w", err)
	}

	// Generate PDF with proper settings
	// B5 paper size: 176mm x 250mm
	_, err = page.PDF(playwright.PagePdfOptions{
		Path:            playwright.String(pdfPath),
		Width:           playwright.String("176mm"),
		Height:          playwright.String("250mm"),
		PrintBackground: playwright.Bool(true),
		Margin: &playwright.Margin{
			Top:    playwright.String("15mm"),
			Right:  playwright.String("15mm"),
			Bottom: playwright.String("15mm"),
			Left:   playwright.String("15mm"),
		},
	})
	if err != nil {
		return fmt.Errorf("could not generate PDF: %w", err)
	}

	return nil
}
