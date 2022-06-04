# ScrapySite
ScrapySite
<img width="1082" alt="image" src="https://user-images.githubusercontent.com/18223385/162614808-b58ce2a8-41cc-498c-ab68-08c5ed75fdef.png">
<img width="874" alt="image" src="https://user-images.githubusercontent.com/18223385/162614856-72d7406f-d988-4043-b25b-9941354cb6f3.png">



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
```bash
./tools/CreateEs.sh scrapy
```

http://127.0.0.1:9200/scrapy_index/_doc/

# how use
```bash
./main -url="http://www.xxx1.cn;http://www.xx2.cn" -resUrl="http://127.0.0.1:9200/st_index/_doc/"
```
http://127.0.0.1:9200/st_index/_search?q=edu&pretty=true
