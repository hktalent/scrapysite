export PATH := $(GOPATH)/bin:$(PATH)
export GO111MODULE=on
LDFLAGS := -s -w

all: build

build: all

all:
	env CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/ScrapySite_51pwn ./main.go

	
clean:
	rm -f ./bin/ScrapySite_51pwn

