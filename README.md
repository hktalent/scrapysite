# ScrapySite
ScrapySite

# how build
```bash
git clone git@github.com:hktalent/scrapysite.git
cd scrapysite
go build main.go
#or build for all palteform
make all -f Makefile.cross-compiles
ls -lah release/
# or build 
make all
ls -lah bin/
ls -lah main
```

# how use Elasticsearch
http://127.0.0.1:9200/_cat/indices?v
1、create index
```
PUT /st_index HTTP/1.1
host:127.0.0.1:9200
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.2 Safari/605.1.15
Connection: close
Content-Type: application/json;charset=UTF-8
Content-Length: 413

{
  "settings": {
   "analysis": {
     "analyzer": {
       "default": {
         "type": "custom",
         "tokenizer": "ik_max_word",
         "char_filter": [
            "html_strip"
          ]
       },
       "default_search": {
         "type": "custom",
         "tokenizer": "ik_max_word",
         "char_filter": [
            "html_strip"
          ]
      }
     }
   }
  }
}
```
2、settings
```
PUT /st_index/_settings HTTP/1.1
host:127.0.0.1:9200
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.2 Safari/605.1.15
Connection: close
Content-Type: application/json;charset=UTF-8
Content-Length: 291

{
  "index.mapping.total_fields.limit": 10000,
 "number_of_replicas" : 0,
"index.translog.durability": "async",
"index.blocks.read_only_allow_delete":"false",
    "index.translog.sync_interval": "5s",
    "index.translog.flush_threshold_size":"100m",
   "refresh_interval": "30s"

}
```
http://127.0.0.1:9200/st_index/_doc/

# how use
```bash
./main -url="http://www.xxx1.cn;http://www.xx2.cn" -resUrl="http://127.0.0.1:9200/st_index/_doc/"
```
http://127.0.0.1:9200/st_index/_search?q=edu&pretty=true
