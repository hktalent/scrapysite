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

# how use
```bash
./main -url="http://www.xxx1.cn;http://www.xx2.cn"
```