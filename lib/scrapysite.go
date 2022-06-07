package lib

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gocolly/colly/v2"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	// "github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/proxy"
	"github.com/gocolly/colly/v2/queue"
)

var CacheDir = DefaultBindConf.CacheDir

// 缓存，基于缓存，避免相同url重复请求，可以修改为redis
var Cache = NewKvDbOp(DefaultBindConf.DnsDbCache)

// es存储的线程控制 nThreads / 2
var EsThreas = make(chan struct{}, DefaultBindConf.ThreadNum)

// 获取ip归宿信息的线程数nThreads / 4
var getIpThread = make(chan struct{}, DefaultBindConf.ThreadNum/4)

// 缓存下次开始到断点
var state = NewState(10)

// 定义爬虫结构
type ScrapySite struct {
	Scrapy *colly.Collector
}

// 获取缓存
func GetCache[T any](k string) (T, error) {
	s, err := Cache.Get(k)
	var aRs T
	if nil == err {
		json.Unmarshal(s, &aRs)
		return aRs, nil
	}
	return aRs, err
}

// 设置缓存
func SetCache(k string, o interface{}) {
	b, err := json.Marshal(o)
	if nil == err {
		Cache.Put(k, b)
	}
}

// 回调决定是否继续请求url
// 链接、标题、选择器,正则，对象
func (r *ScrapySite) CallBack(link, title string, regUrl []string, regTitle string, i interface{}) bool {
	szUrl := link // e.Request.URL.String()
	// 过滤
	if r.FilterHref(szUrl) {
		return false
	}
	if 0 <= len(regUrl) {
		for _, j := range regUrl {
			r1, err := regexp.Compile(j)
			if nil == err {
				if 0 < len(r1.FindAllString(szUrl, -1)) {
					//FnLog("CallBack: ", title, link)
					return true
				}
			} else {
				FnLog("CallBack: ", title, link, j, err)
			}
		}
	}
	if "" != regTitle {
		r1, err := regexp.Compile(regTitle)
		if nil == err {
			if 0 < len(r1.FindAllString(title, -1)) {
				//FnLog("CallBack: ", title, link)
				return true
			}
		} else {
			FnLog("CallBack: ", title, link, regTitle, err)
		}
	}
	//FnLog("CallBack: ", title, link, regUrl)
	return false
}
func (r *ScrapySite) FilterHref(s string) bool {
	for _, x := range DefaultBindConf.FilterHrefReg {
		reg1 := regexp.MustCompile(x)
		if reg1.MatchString(s) {
			return true
		}
	}
	return false
}

// 创建爬虫对象
// 默认开启异步
// 默认MaxDepth 微0，不受深度限制
// 默认允许、开启AllowURLRevisit
// 默认 忽略了 IgnoreRobotsTxt
// default: IgnoreRobotsTxt
// 默认Timeout 30秒, KeepAlive 15秒
// 最大空闲连接数 100
// 空闲连接超时 90秒
// TLS 握手超时 10秒
// 默认不限制域名
func NewScrapySite() *ScrapySite {
	var r *ScrapySite
	r = &ScrapySite{}
	// Cache = cache.New(10000*time.Hour, 10000*time.Hour)
	r.Scrapy = colly.NewCollector(colly.CacheDir(CacheDir), colly.MaxDepth(0), colly.Async(), colly.AllowURLRevisit(), colly.IgnoreRobotsTxt())
	r.Scrapy.WithTransport(&http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   DefaultBindConf.Timeout * time.Second,   // 超时时间
			KeepAlive: DefaultBindConf.KeepAlive * time.Second, // keepAlive 超时时间
		}).DialContext,
		MaxIdleConns:          DefaultBindConf.MaxIdleConns,                      // 最大空闲连接数
		IdleConnTimeout:       DefaultBindConf.IdleConnTimeout * time.Second,     // 空闲连接超时
		TLSHandshakeTimeout:   DefaultBindConf.TLSHandshakeTimeout * time.Second, // TLS 握手超时
		ExpectContinueTimeout: DefaultBindConf.ExpectContinueTimeout * time.Second,
	})

	r.Scrapy.Limit(&colly.LimitRule{
		DomainGlob:  DefaultBindConf.DomainGlob,
		Parallelism: DefaultBindConf.ThreadNum,
		RandomDelay: DefaultBindConf.RandomDelay * time.Second})
	// extensions.RandomUserAgent(r.Scrapy)
	// extensions.Referer(r.Scrapy)
	// http://go-colly.org/docs/best_practices/multi_collector/
	// c2 := r.Scrapy.Clone()
	r.init()
	return r
}

// 获取domain的所有ip地址
// 注意：一个域名可能解析出多个ip
func (r *ScrapySite) GetDomainIps(domain string) (aRst []string) {
	for _, x := range DefaultBindConf.IpRegs {
		re := regexp.MustCompile(x)
		aRst = []string{}
		// 找到了，说明是ip，就不在转换，直接返回
		if "" != re.FindString(domain) {
			aRst = []string{domain}
		} else {
			oRst, err := GetCache[[]string](domain)
			// 有缓存,且有效、有数据
			if err == nil && nil != oRst && 0 < len(oRst) {
				aRst = oRst
				return
			}
			// bug: 如果第一次没有获取到ip，后续非一直获取
			ips, err := net.LookupIP(domain)
			if nil == err {
				m11 := make(map[string]interface{})
				for _, ip := range ips {
					if ipv4 := ip.To4(); ipv4 != nil {
						if _, ok := m11[ipv4.String()]; ok {
							continue
						}
						aRst = append(aRst, ipv4.String())
						m11[ipv4.String()] = "1"
						continue
					}
					if ipv4 := ip.To16(); ipv4 != nil {
						if _, ok := m11[ipv4.String()]; ok {
							continue
						}
						aRst = append(aRst, ipv4.String())
						m11[ipv4.String()] = "1"
					}
				}
				m11 = nil
				if 0 < len(aRst) {
					SetCache(domain, aRst)
				}
			}
		}
	}
	return
}

// 获取url的domain对应的ip，及ip归宿信息
func (r *ScrapySite) GetDomainInfo(szUrl string, result *Result) []IpInfo {
	var aR = []IpInfo{}
	var domain string
	oUrl, err := url.Parse(szUrl)
	if nil != err {
		FnLog(err)
		return aR
	}
	domain = oUrl.Host
	//FnLog("GetDomainInfo domain: ", domain)
	if "" != domain {
		result.IpInfo = aR
		aRst := r.GetDomainIps(domain)
		if 0 < len(aRst) {
			for _, x := range aRst {
				xD, err := r.GetIpInfo(x)
				if nil != err {
					FnLog(err)
					continue
				}
				if nil != xD {
					aR = append(aR, *xD)
				} else { // 没有找到，或者网络异常的情况
					aR = append(aR, IpInfo{Query: x})
				}
			}
		}
		return aR
	}
	return aR
}

// 获取ip归宿信息
func (ss *ScrapySite) GetIpInfo(ip string) (*IpInfo, error) {
	oRst, err := GetCache[IpInfo](ip)
	if nil == err {
		return &oRst, nil
	}
	getIpThread <- struct{}{}
	defer func() {
		<-getIpThread
	}()
	req, err := http.NewRequest("GET", fmt.Sprintf(DefaultBindConf.GetIpUrlFormat, ip), nil)
	if err != nil {
		FnLog("GetIpInfo NewRequest ", err)
		return nil, err
	}
	req.Header.Set("User-Agent", "curl/11")
	req.Header.Add("Content-Type", "application/json;charset=UTF-8")
	// keep-alive
	req.Header.Add("Connection", "close")
	req.Close = true

	resp, err := http.DefaultClient.Do(req)
	if resp != nil {
		defer resp.Body.Close() // resp 可能为 nil，不能读取 Body
	}
	if err != nil {
		FnLog("GetIpInfo DefaultClient ", err)
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		FnLog("GetIpInfo ReadAll ", err)
		return nil, err
	}
	var rst IpInfo
	err = json.Unmarshal(body, &rst)
	if nil == err {
		SetCache(ip, rst)
	} else {
		FnLog("GetIpInfo Unmarshal ", err)
	}
	return &rst, err
}

//var esCache = NewKvDbOp("EsCache")
// 指定id发送数据到ES url，，data为json转换后到数据
func (ss *ScrapySite) SendReq(data *bytes.Buffer, id string) {
	EsThreas <- struct{}{}
	defer func() {
		<-EsThreas
	}()
	//// 处理过了
	//szR, err := esCache.Get(id)
	//if nil == err && nil != szR {
	//	return
	//}
	//esCache.Put(id, []byte("1"))
	req, err := http.NewRequest("POST", DefaultBindConf.EsUrl+url.QueryEscape(id), data)
	if err != nil {
		FnLog(err)
		return
	}
	//FnLog(data.String())
	// 取消全局复用连接
	// tr := http.Transport{DisableKeepAlives: true}
	// client := http.Client{Transport: &tr}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.2 Safari/605.1.15")
	req.Header.Add("Content-Type", "application/json;charset=UTF-8")
	// keep-alive
	req.Header.Add("Connection", "close")
	req.Close = true
	defer req.Body.Close()

	resp, err := http.DefaultClient.Do(req)
	if resp != nil {
		defer resp.Body.Close() // resp 可能为 nil，不能读取 Body
	}
	if err != nil {
		FnLog("SendReq", DefaultBindConf.EsUrl, id, err)
		return
	}
	s2, err := ioutil.ReadAll(resp.Body)
	if nil == err {
		FnLog("Elasticsearch save ok: ", id, string(s2))
	} else {
		FnLog("Elasticsearch save err: ", id, err)
	}
}

// 发送man到ES
func (ss *ScrapySite) SendJsonReq(data *Result) {
	jsonValue, err := json.Marshal(*data)
	if nil == err {
		ss.SendReq(bytes.NewBuffer(jsonValue), data.Url)
	} else {
		FnLog("SendJsonReq", err)
	}
}

// 添加不重复url
func (ss *ScrapySite) AddUrls(url string, data []string) []string {
	a := data
	for _, x := range a {
		if x == url {
			return data
		}
	}
	return append(a, url)
}

// 处理非文本文件
// 计算md5、sha1、sha256 到map
func (ss *ScrapySite) DoNotText4Parms(body []byte, ContentType string, r *Result) {
	// 非文本，计算sha1、md5
	if nil != body {
		if strings.Index(ContentType, "text/") == -1 && 0 < len(body) {
			x1 := body
			md5R, sha1R, sha256R := ss.Hash(x1)
			r.Md5 = md5R
			r.Sha1 = sha1R
			r.Sha256 = sha256R
		}
	}
}

// 处理非文本文件
// 计算md5、sha1、sha256 到map
func (ss *ScrapySite) DoNotText(r *colly.Response, result *Result) {
	ss.DoNotText4Parms(r.Body, r.Headers.Get("Content-Type"), result)
}

func (ss *ScrapySite) DoExtractors(r *Result, body []byte, x1 *ScrapyRule, e *colly.HTMLElement) bool {
	if nil != e && nil != x1 && nil != r && nil != body && 0 < len(body) {
		szB := string(body)
		aRstE := []string{}
		for _, j := range x1.Extractors {
			r1, err := regexp.Compile(j.Reg)
			if nil != err {
				FnLog(j)
				continue
			}
			var aR1 []string
			if "body" == j.Type {
				aR1 = r1.FindAllString(szB, -1)
			} else if "url" == j.Type {
				aR1 = r1.FindAllString(r.Url, -1)
			}
			//e.Response.Headers
			if nil != aR1 && 0 < len(aR1) {
				aRstE = append(aRstE, aR1...)
				if "url" == j.Type {
					r.Url = "https://" + aR1[0]
					r.IpInfo = ss.GetDomainInfo(r.Url, r)
					if nil == r.IpInfo || 0 == len(r.IpInfo) {
						return false
					}
				}
			}
		}
		if 0 < len(aRstE) {
			FnLog("DoExtractors ", aRstE)
			r.Results = aRstE
			return true
		}
	}
	return false
}

func (ss *ScrapySite) DoResponseMap(absUrl, url, ContentType string, StatusCode int, body []byte, x1 *ScrapyRule, e *colly.HTMLElement) {
	var result = Result{Url: url}
	oRst, err := GetCache[Result](url)
	if nil == err {
		result = oRst
	} else {
		if xx := ss.GetDomainInfo(url, &result); 0 < len(xx) {
			result.IpInfo = xx
		}
	}

	if ss.DoExtractors(&result, body, x1, e) {
		ss.DoNotText4Parms(body, ContentType, &result)
		SetCache(url, result)
		go ss.SendJsonReq(&result)
	}
}

// 处理请求响应，并转换为ES需要到结构
// id长度不能小雨14
func (ss *ScrapySite) DoResponse2Es(r *colly.Response, x1 *ScrapyRule, e *colly.HTMLElement) {
	ss.DoResponseMap(r.Request.AbsoluteURL(r.Request.URL.String()), r.Request.URL.String(), r.Headers.Get("Content-Type"), r.StatusCode, r.Body, x1, e)
}

func (ss *ScrapySite) SetProxys(proxys []string) {
	rp, err := proxy.RoundRobinProxySwitcher(proxys...)
	if err != nil {
		return
	}
	ss.Scrapy.SetProxyFunc(rp)
}

var doOnce sync.Once

// 请求到url：e.Request.URL.String()
// https://github.com/gocolly/colly/blob/master/_examples/multipart/multipart.go
// e.Request.PostMultipart("http://localhost:8080/", generateFormData())
func (ss *ScrapySite) init() {
	doOnce.Do(func() {
		if nil != DefaultBindConf.Proxys && 0 < len(DefaultBindConf.Proxys) {
			ss.SetProxys(DefaultBindConf.Proxys)
		}
		ss.Scrapy.OnResponse(func(r *colly.Response) {
			ss.DoResponse2Es(r, nil, nil)
		})
		// 爬虫规则
		for _, x := range DefaultBindConf.ScrapyRule {
			func(x1 ScrapyRule) {
				for _, j := range x1.Selector {
					ss.Scrapy.OnHTML(j, func(e *colly.HTMLElement) {
						link := e.Attr("href")
						if "//" == link[0:2] {
							link = "https:" + link
						}
						title := strings.TrimSpace(e.Text)
						if ss.CallBack(link, title, x1.CbkUrlReg, x1.CbkTitleReg, e) {
							go state.Push(link)
							go ss.DoResponse2Es(e.Response, &x1, e)
							FnLog("start : ", link)
							SetCache(link, Result{Url: link, Title: title})
							e.Request.Visit(link)
						}
					})
				}
			}(x)
		}
		// Set error handler
		//ss.Scrapy.OnError(func(r *colly.Response, err error) {
		//	ss.DoResponse2Es(r, nil, nil)
		//	fmt.Println("Request URL:", r.Request.URL, "failed with response:", "Error:", err)
		//})
	})
}

// 求hash md5,sha1,sha256,SM3: 国密hash算法库
func (ss *ScrapySite) Hash(data []byte) (md5R string, sha1R string, sha256R string) {
	md5R = fmt.Sprintf("%x", md5.Sum(data))
	sha1R = fmt.Sprintf("%x", sha1.Sum(data))
	sha256R = fmt.Sprintf("%x", sha256.Sum256(data))
	return
}

// base64 编码到string
func (ss *ScrapySite) Base64EncodeToString(data []byte) string {
	return b64.StdEncoding.EncodeToString(data)
}

// base64 解码到string
func (ss *ScrapySite) Base64DecodeString(data string) string {
	return string(ss.Base64DecodeString2byte(data))
}

// base64 解码到[]byte
func (ss *ScrapySite) Base64DecodeString2byte(data string) []byte {
	szR, _ := b64.StdEncoding.DecodeString(data)
	return szR
}

// 注册处理响应数据的回调
func (ss *ScrapySite) OnResponse(onResponse func(r *colly.Response)) {
	ss.Scrapy.OnResponse(func(r *colly.Response) {
		// r.Ctx.Get("url")
		onResponse(r)
		// d := ss.Scrapy.Clone()
		// d.Request("GET", u, nil, ctx, nil)
	})
}

// https://github.com/gocolly/colly/blob/b151a08fbde2b67d960bd9991c1f346e5a1cdd77/_examples/instagram/instagram.go#L94
// r.URL.String()
func (ss *ScrapySite) OnRequest(onRequest func(*colly.Request)) {
	ss.Scrapy.OnRequest(func(r *colly.Request) {
		// r.Ctx.Put("url", r.URL.String())
		// FnLog("我的请求", r.URL.String(), r.AbsoluteURL(r.URL.String()), r.Headers.Get("User-Agent"))
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.2 Safari/605.1.15")
		onRequest(r)
	})
}

//"a[href]"
//"script[src]"
//"form[action]"
//c.AllowedDomains = nil
//c.URLFilters = []*regexp.Regexp{regexp.MustCompile(".*(\\.|\\/\\/)" + strings.ReplaceAll(hostname, ".", "\\.") + "((#|\\/|\\?).*)?")
// 支持多个url，分隔符号：[;,| ]
func (ss *ScrapySite) Start() {
	storage := &Storage{
		Filename: DefaultBindConf.SqliteDbName,
	}
	defer storage.Close()
	ss.Scrapy.SetStorage(storage)
	q, _ := queue.New(
		DefaultBindConf.ThreadNum, // Number of consumer threads
		storage,
		// &queue.InMemoryQueueStorage{MaxSize: nThreads}, // Use default queue storage
	)
	var nHv int = 0
	for _, x := range state.Urls {
		if "" != x {
			nHv += 1
			q.AddURL(x)
		}
	}
	// 从断点开始
	if 0 == nHv {
		for _, x := range DefaultBindConf.ScrapyRule {
			for _, j := range x.StartUrls {
				q.AddURL(j)
			}
			// ss.Scrapy.Visit(url)
		}
	}
	q.Run(ss.Scrapy)
	ss.Scrapy.Wait()
}
