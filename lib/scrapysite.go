package lib

import (
	"fmt"
	"log"

	"github.com/gocolly/colly/v2"
)

type ScrapySite struct {
	Scrapy *colly.Collector
}

func (ss *ScrapySite) NewScrapySite() *ScrapySite {
	return &ScrapySite{}
}

func (ss *ScrapySite) Init(fnCbk func(string, string) bool) {
	// Cache responses to prevent multiple download of pages
	// even if the collector is restarted
	ss.Scrapy = colly.NewCollector(colly.CacheDir("./coursera_cache"))
	ss.Scrapy.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		if fnCbk(link, e.Text) {
			requestIDURL := e.Request.AbsoluteURL(e.ChildAttr(`link[as="script"]`, "href"))
			log.Println(requestIDURL)
			// ss.Scrapy.Visit(e.Request.AbsoluteURL(link))
		}
	})
	// Set error handler
	ss.Scrapy.OnError(func(r *colly.Response, err error) {
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

}

func (ss *ScrapySite) OnResponse(onResponse func(r *colly.Response) bool) {
	ss.Scrapy.OnResponse(func(r *colly.Response) {
		onResponse(r)
		// d := ss.Scrapy.Clone()
		// d.Request("GET", u, nil, ctx, nil)
	})

}

// https://github.com/gocolly/colly/blob/b151a08fbde2b67d960bd9991c1f346e5a1cdd77/_examples/instagram/instagram.go#L94
func (ss *ScrapySite) OnRequest(onRequest func(r *colly.Request) bool) {
	ss.Scrapy.OnRequest(func(r *colly.Request) {
		onRequest(r)
	})

}
