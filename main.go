package main

import (
	"fmt"

	"github.com/gocolly/colly"
	scrapysite "github.com/hktalent/scrapysite/lib"
)

func fnCbk(link, text string) bool {
	fmt.Printf("Link found: %q -> %s\n", text, link)
	return true
}

func main() {
	var scrapysite *scrapysite.Scrapysite
	scrapysite = scrapysite.NewScrapysite()
	scrapysite.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})
	scrapysite.Init(fnCbk)
	scrapysite.Visit("http://www.xxx.cn/")
}
