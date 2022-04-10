package main

import (
	"flag"
	"fmt"

	"github.com/gocolly/colly/v2"
	ss51pwn "github.com/hktalent/scrapysite/lib"
)

func fnCbk(link, text string) bool {
	fmt.Printf("Link found: %q -> %s\n", text, link)
	return true
}

func main() {

	url := flag.String("url", "", "scrapy url")

	flag.Parse()

	var scrapysite *ss51pwn.ScrapySite
	scrapysite = ss51pwn.NewScrapySite()
	scrapysite.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})
	scrapysite.Init(fnCbk)
	scrapysite.Start(*url)
}
