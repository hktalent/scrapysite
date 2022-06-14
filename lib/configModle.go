package lib

import (
	"github.com/fsnotify/fsnotify"
	"github.com/hktalent/scrapysite/pkg/model"
	"github.com/spf13/viper"
	"log"
	"time"
)

// 整体配置
type Config struct {
	Proxys                []string        `json:"proxys"`                // 代理
	SqliteDbName          string          `json:"sqliteDbName"`          // "./db/results.db"
	EsUrl                 string          `json:"esUrl"`                 // Elasticsearch url, http://127.0.0.1:9200/scrapy_index/_doc/
	CacheDir              string          `json:"cacheDir"`              // ./db/coursera_cache
	DnsDbCache            string          `json:"dnsDbCache"`            // ./db/dnsDbCache
	EsCacheName           string          `json:"esCacheName"`           // ./db/EsCache
	Timeout               time.Duration   `json:"timeout"`               // default 30s
	KeepAlive             time.Duration   `json:"keepAlive"`             // defalt 10s
	MaxIdleConns          int             `json:"maxIdleConns"`          // default 100
	IdleConnTimeout       time.Duration   `json:"idleConnTimeout"`       // 90s
	TLSHandshakeTimeout   time.Duration   `json:"TLSHandshakeTimeout"`   // 10s
	ExpectContinueTimeout time.Duration   `json:"expectContinueTimeout"` // 1s
	ThreadNum             int             `json:"threadNum"`             // 8
	DomainGlob            string          `json:"domainGlob"`            // *
	RandomDelay           time.Duration   `json:"randomDelay"`           // 15
	GetIpUrlFormat        string          `json:"getIpUrlFormat"`        // "http://ip-api.com/json/%s"
	ScrapyRule            []model.SecRule `json:"scrapyRule"`            // 多个匹配规则
	IpRegs                []string        `json:"ipRegs"`                // ip regex
	FilterHrefReg         []string        `json:"filterHrefReg"`         // filter href reg,^(#|javascript|mailto:)|(favicon\.ico$)
	Verbose               bool            `json:"verbose"`               // true
	CloseEsSave           bool            `json:"closeEsSave"`           // false
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
	ScrapyRule:            []model.SecRule{},
	CloseEsSave:           false,
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
