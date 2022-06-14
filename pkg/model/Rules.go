package model

// 规则元数据
type Rule struct {
	Rule      string   `json:"rule"`
	Type      string   `json:"type"`      // selector：（0 css selector, 1 xpath selector） ； 2 regxp 为抽取器, 3 json 选择器
	Part      string   `json:"part"`      // (e.Response: title, head, body, e.Request.URL.String() -> url),(e *colly.HTMLElement: text,curlHtml,url),
	FilterReg []string `json:"filterReg"` // CbkUrlReg 过滤,满足条件的过滤，返回false
	Active    []string `json:"active"`    /* 行为、动作，允许写多个动作动作的结果作为下一个输入,例如:
	CVE 编号提取成功，则动作为获取漏洞信息，自动找poc
	a 规则，则动作是去重
	poc 做一些常规预判
	*/
}

//type ExtractorMod struct {
//	Reg  string `json:"reg"`
//	Type string `json:"type"` // url,body
//}

// 情报提取规则
type SecRule struct {
	Urls         []string `json:"urls"` // start
	SameDomain   bool     `json:"sameDomain"`
	A            []Rule   `json:"a"`
	Pages        []Rule   `json:"pages"`
	Title        []Rule   `json:"title"`
	PublishDate  []Rule   `json:"publishDate"`
	LastModified []Rule   `json:"lastModified"`
	Tags         []Rule   `json:"tags"`
	Cve          []Rule   `json:"cve"`
	Content      []Rule   `json:"content"`
	Poc          []Rule   `json:"poc"`
}

// ip信息模型
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

// https://www.rapid7.com/blog/post/2022/03/30/spring4shell-zero-day-vulnerability-in-spring-framework/
//https://www.rapid7.com/blog/tag/research/
// 漏洞情报结果数据模型
type SecData struct {
	Url          string `json:"url"`
	Title        string `json:"title"`
	PublishDate  string `json:"publishDate"`
	LastModified string `json:"lastModified"`
	Tags         string `json:"tags"`
	Cve          string `json:"cve"`
	Content      string `json:"content"`
	Poc          string `json:"poc"`
	PocLink      string `json:"pocLink"` // poc 链接，poc不是来自当前情报，而是来自其他网站
	Md5          string `json:"md5"`
	Sha1         string `json:"sha1"`
	Sha256       string `json:"sha256"`
	//IpInfo       []IpInfo `json:"ipInfo"`  // 域名对应的ip信息，使用es 处理 id关联
}

// 数源属性模型
// 虽然能自动识别是境外还是境内，但是还是手工配置是否需要从境外访问
type DataSource struct {
	Url       string
	LimitType string `json:"limit_type"` // 年月日时分秒，并发限制
	Limit     int64  `json:"limit"`      // 在LimitType周期内允许的请求数，例如 github每小时时5000
	UseVps    bool   `json:"use_vps"`    // 是否需要从境外访问
}
