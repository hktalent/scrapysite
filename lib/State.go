package lib

import "encoding/json"

type State struct {
	Urls []string `json:"urls"`
	Size int      `json:"size"`
}

var cacheS = NewKvDbOp("db/cacheSt")
var szKey1 = "__NewState__"

func NewState(size int) *State {
	var r State
	o, err := cacheS.Get(szKey1)
	if nil == err {
		json.Unmarshal(o, &r)
	} else {
		r = State{Urls: []string{}}
	}
	r.Size = size
	return &r
}

func (r *State) Push(url string) *State {
	if len(r.Urls) < r.Size {
		r.Urls = append(r.Urls, url)
	} else {
		r.Urls = append(r.Urls[1:], url)
	}
	go cacheS.PutAny(szKey1, r)
	return r
}
