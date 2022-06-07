package lib

import "encoding/json"

type State struct {
	Urls []string `json:"urls"`
}

var cacheS = NewKvDbOp("db/cacheSt")
var szKey1 = "__NewState__"

func NewState(size int) *State {
	var r State
	o, err := cacheS.Get(szKey1)
	if nil == err {
		json.Unmarshal(o, &r)
	} else {
		r = State{Urls: make([]string, size)}
	}
	return &r
}

func (r *State) Push(url string) *State {
	r.Urls = append(r.Urls[1:len(r.Urls)-1], url)
	go cacheS.PutAny(szKey1, r)
	return r
}
