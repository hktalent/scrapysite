package main

import (
	"flag"
	"log"
	"regexp"
	"strings"

	"github.com/gocolly/colly/v2"
	ss51pwn "github.com/hktalent/scrapysite/lib"
)

var reg1 = regexp.MustCompile(`^(#|javascript|mailto:)|(favicon\.ico$)`)

// 请求到url：e.Request.URL.String()
// e.Request: URL,Headers,Depth,Method,ResponseCharacterEncoding,
// e.Response:
// StatusCode
// Body
// Request
// Headers
func fnCbk(link, text string, e *colly.HTMLElement) bool {
	if "" != reg1.FindString(link) {
		return false
	}
	// fmt.Printf("Link found: %s -> %s  %s\n", text, link, e.Request.URL.String())
	return true
}

func main() {

	url := flag.String("url", "", "scrapy url")
	resUrl := flag.String("resUrl", "", "Elasticsearch url")

	flag.Parse()

	log.Println(*url)
	if "" != *url {
		var scrapysite *ss51pwn.ScrapySite
		scrapysite = ss51pwn.NewScrapySite(*resUrl, fnCbk)
		scrapysite.OnRequest(func(r *colly.Request) {
			// fmt.Println("Visiting", r.URL.String())
		})
		scrapysite.OnResponse(func(r *colly.Response) {
			// 非文本，计算sha1、md5
			if strings.Index(r.Headers.Get("Content-Type"), "text/") == -1 {
				r.Save("./xx/" + r.FileName())
				return
			}
		})
		scrapysite.Start(*url)

		// spew.Dump(scrapysite.GetDomainInfo("http://www.gov.cn"))

		// md5R, sha1R, sha256R := scrapysite.Hash([]byte("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"))
		// log.Println(md5R, sha1R, sha256R)
	}
}
