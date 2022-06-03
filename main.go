package main

import (
	"strings"

	"github.com/gocolly/colly/v2"
	ss51pwn "github.com/hktalent/scrapysite/lib"
)

// 请求到url：e.Request.URL.String()
// e.Request: URL,Headers,Depth,Method,ResponseCharacterEncoding,
// e.Response:
// StatusCode
// Body
// Request
// Headers
func fnCbk(link, text string, e interface{}) bool {
	szUrl := link // e.Request.URL.String()
	if -1 < strings.Index(szUrl, "https://seo.chinaz.com/") {
		return true
	}

	return false
}

func main() {
	var scrapysite = ss51pwn.NewScrapySite()
	scrapysite.OnRequest(func(r *colly.Request) {
		//fmt.Println("Visiting", r.URL.String())
	})
	scrapysite.OnResponse(func(r *colly.Response) {
		// 非文本，计算sha1、md5
		if strings.Index(r.Headers.Get("Content-Type"), "text/") == -1 {
			//r.Save("./xx/" + r.FileName())
			return
		}
	})
	scrapysite.Start()
	// spew.Dump(scrapysite.GetDomainInfo("http://www.gov.cn"))
	// md5R, sha1R, sha256R := scrapysite.Hash([]byte("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"))
	// log.Println(md5R, sha1R, sha256R)

}
