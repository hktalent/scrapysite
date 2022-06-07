package lib

import (
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"log"
	"time"
)

type ExtractorMod struct {
	Reg  string `json:"reg"`
	Type string `json:"type"` // url,body
}

type IpInfo struct {
	Continent     string  `json:"continent,omitempty"`
	ContinentCode string  `json:"continentCode,omitempty"`
	Country       string  `json:"country,omitempty"`
	CountryCode   string  `json:"countryCode,omitempty"`
	Region        string  `json:"region,omitempty"`
	RegionName    string  `json:"regionName,omitempty"`
	City          string  `json:"city,omitempty"`
	District      string  `json:"district,omitempty"`
	Zip           string  `json:"zip,omitempty"`
	Lat           float64 `json:"lat,omitempty"`
	Lon           float64 `json:"lon,omitempty"`
	Timezone      string  `json:"timezone,omitempty"`
	Offset        string  `json:"offset,omitempty"`
	Currency      string  `json:"currency,omitempty"`
	Isp           string  `json:"isp,omitempty"`
	Org           string  `json:"org,omitempty"`
	As            string  `json:"as,omitempty"`
	Asname        string  `json:"asname,omitempty"`
	Mobile        string  `json:"mobile,omitempty"`
	Proxy         string  `json:"proxy,omitempty"`
	Hosting       string  `json:"hosting,omitempty"`
	Query         string  `json:"query"` // IP
}
type Result struct {
	IpInfo  []IpInfo `json:"ipInfo"`
	Title   string   `json:"title"`
	Url     string   `json:"url"`
	Results []string `json:"results"`
	Md5     string   `json:"md5"`
	Sha1    string   `json:"sha1"`
	Sha256  string   `json:"sha256"`
}

// 爬虫规则
type ScrapyRule struct {
	StartUrls   []string       `json:"startUrls"`   // 开始爬取到urls
	Selector    []string       `json:"selector"`    // jQuery's stateful manipulation functions https://github.com/PuerkitoBio/goquery
	CbkUrlReg   []string       `json:"cbkUrlReg"`   // Callback url regexp
	CbkTitleReg string         `json:"cbkTitleReg"` // Callback title regexp
	Extractors  []ExtractorMod `json:"extractors"`  // result extractors
}

// 整体配置
type Config struct {
	Proxys                []string      `json:"proxys"`                // 代理
	SqliteDbName          string        `json:"sqliteDbName"`          // "./db/results.db"
	EsUrl                 string        `json:"esUrl"`                 // Elasticsearch url, http://127.0.0.1:9200/scrapy_index/_doc/
	CacheDir              string        `json:"cacheDir"`              // ./db/coursera_cache
	DnsDbCache            string        `json:"dnsDbCache"`            // ./db/dnsDbCache
	EsCacheName           string        `json:"esCacheName"`           // ./db/EsCache
	Timeout               time.Duration `json:"timeout"`               // default 30s
	KeepAlive             time.Duration `json:"keepAlive"`             // defalt 10s
	MaxIdleConns          int           `json:"maxIdleConns"`          // default 100
	IdleConnTimeout       time.Duration `json:"idleConnTimeout"`       // 90s
	TLSHandshakeTimeout   time.Duration `json:"TLSHandshakeTimeout"`   // 10s
	ExpectContinueTimeout time.Duration `json:"expectContinueTimeout"` // 1s
	ThreadNum             int           `json:"threadNum"`             // 8
	DomainGlob            string        `json:"domainGlob"`            // *
	RandomDelay           time.Duration `json:"randomDelay"`           // 15
	GetIpUrlFormat        string        `json:"getIpUrlFormat"`        // "http://ip-api.com/json/%s"
	ScrapyRule            []ScrapyRule  `json:"scrapyRule"`            // 多个匹配规则
	IpRegs                []string      `json:"ipRegs"`                // ip regex
	FilterHrefReg         []string      `json:"filterHrefReg"`         // filter href reg,^(#|javascript|mailto:)|(favicon\.ico$)
	Verbose               bool          `json:"verbose"`               // true
}

// default config
var DefaultBindConf = Config{
	Verbose:               true,
	Proxys:                []string{},
	SqliteDbName:          "./db/results.db",
	CacheDir:              "./db/coursera_cache",
	DnsDbCache:            "./db/dnsDbCache",
	EsCacheName:           "./db/EsCache",
	Timeout:               30,
	KeepAlive:             10,
	MaxIdleConns:          100,
	IdleConnTimeout:       90,
	TLSHandshakeTimeout:   10,
	ExpectContinueTimeout: 1,
	ThreadNum:             8,
	DomainGlob:            "*",
	RandomDelay:           15,
	EsUrl:                 "http://127.0.0.1:9200/scrapy_index/_doc/",
	GetIpUrlFormat:        "http://ip-api.com/json/%s",
	IpRegs:                []string{`^(\d{1,3}\.){3}\d{1,3}$`},
	FilterHrefReg:         []string{`^(#|javascript|mailto:)|(favicon\.ico$)`},
	ScrapyRule:            []ScrapyRule{},
}

func FnLog(x ...any) {
	if DefaultBindConf.Verbose {
		log.Println(x)
	}
}

var ConfigName string

// init config
func Init() {

	config := viper.New()
	config.SetConfigName("config")
	config.AddConfigPath("./")
	config.AddConfigPath("./config")
	config.AddConfigPath("$HOME")
	config.AddConfigPath("/etc/")
	// 显示调用
	//config.SetConfigType("json")
	if "" != ConfigName {
		config.SetConfigFile(ConfigName)
	}
	err := config.ReadInConfig() // 查找并读取配置文件
	if err != nil {              // 处理读取配置文件的错误
		FnLog(err)
		return
	}
	// 将读取的配置信息保存至全局变量Conf
	if err := config.Unmarshal(&DefaultBindConf); err != nil {
		FnLog(err)
		return
	}
	viper.Set("Verbose", DefaultBindConf.Verbose)
	// 监控配置文件变化
	config.WatchConfig()
	config.OnConfigChange(func(in fsnotify.Event) {
		if err := config.Unmarshal(&DefaultBindConf); err != nil {
			FnLog(err)
		} else {
			// 重启服务
		}
	})
}
