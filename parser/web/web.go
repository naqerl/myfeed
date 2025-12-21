package web

import (
	"fmt"

	"github.com/mackee/go-readability"
	"github.com/playwright-community/playwright-go"

	"github.com/scipunch/myfeed/fetcher/types"
	"github.com/scipunch/myfeed/parser"
)

type Parser struct {
	pw      *playwright.Playwright
	browser playwright.Browser
}

func New() (Parser, error) {
	var p Parser
	err := playwright.Install()
	if err != nil {
		return p, err
	}
	pw, err := playwright.Run()
	if err != nil {
		return p, fmt.Errorf("could not start playwright: %w", err)
	}
	browser, err := pw.Chromium.Launch()
	if err != nil {
		return p, fmt.Errorf("could not launch browser: %w", err)
	}
	p.pw = pw
	p.browser = browser
	return p, nil
}

func (p Parser) Close() error {
	if err := p.browser.Close(); err != nil {
		return err
	}
	return p.pw.Stop()
}

type Response struct {
	HTML string
}

func (r Response) String() string {
	return r.HTML
}

func (p Parser) Parse(item types.FeedItem) (parser.Response, error) {
	var resp Response
	page, err := p.browser.NewPage()
	if err != nil {
		return resp, fmt.Errorf("could not create page: %w", err)
	}
	defer page.Close()
	if _, err = page.Goto(item.Link); err != nil {
		return resp, fmt.Errorf("could not go to '%s': %w", item.Link, err)
	}
	rawHtml, err := page.Content()
	if err != nil {
		return resp, fmt.Errorf("could not read page content at '%s': %w", item.Link, err)
	}
	resp.HTML = rawHtml

	options := readability.DefaultOptions()
	article, err := readability.Extract(string(rawHtml), options)
	if err != nil {
		return resp, fmt.Errorf("could not use readability for '%s': %w", item.Link, err)
	}

	if article.Root == nil {
		return resp, fmt.Errorf("readability returned empty article for '%s'", item.Link)
	}

	resp.HTML = readability.ToHTML(article.Root)
	return resp, nil
}
