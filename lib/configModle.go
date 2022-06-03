package lib

import (
	"bufio"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"log"
	"os"
	"time"
)

// 爬虫规则
type ScrapyRule struct {
	StartUrls   []string `json:"startUrls"`   // 开始爬取到urls
	Selector    string   `json:"selector"`    // jQuery's stateful manipulation functions https://github.com/PuerkitoBio/goquery
	CbkUrlReg   string   `json:"cbkUrlReg"`   // Callback url regexp
	CbkTitleReg string   `json:"cbkTitleReg"` // Callback title regexp
}

// 整体配置
type Config struct {
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
	ThreadNum             int           `json:"threadNum"`             // 16
	DomainGlob            string        `json:"domainGlob"`            // *
	RandomDelay           time.Duration `json:"randomDelay"`           // 15
	GetIpUrlFormat        string        `json:"getIpUrlFormat"`        // "http://ip-api.com/json/%s"
	ScrapyRule            []ScrapyRule  `json:"scrapyRule"`            // 多个匹配规则
	IpRegs                []string      `json:"ipRegs"`                // ip regex
	FilterHrefReg         []string      `json:"filterHrefReg"`         // filter href reg,^(#|javascript|mailto:)|(favicon\.ico$)
	Verbose               bool          `json:"verbose"`               // false
}

// default config
var DefaultBindConf = Config{
	Verbose:               false,
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
	ThreadNum:             16,
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
	var bRd = false
	if "" != ConfigName {
		file, err := os.Open(ConfigName)
		if nil == err {
			defer file.Close()
			viper.ReadConfig(bufio.NewReader(file))
			bRd = true
		}
	}
	if !bRd {
		viper.AddConfigPath("./config")
		viper.AddConfigPath("$HOME")
	}
	err := viper.ReadInConfig() // 查找并读取配置文件
	if err != nil {             // 处理读取配置文件的错误
		FnLog(err)
		return
	}
	// 将读取的配置信息保存至全局变量Conf
	if err := viper.Unmarshal(&DefaultBindConf); err != nil {
		FnLog(err)
		return
	}
	// 监控配置文件变化
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		if err := viper.Unmarshal(&DefaultBindConf); err != nil {
			FnLog(err)
		} else {
			// 重启服务
		}
	})
}
