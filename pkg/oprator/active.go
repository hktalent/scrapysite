package main

import (
	"github.com/gocolly/colly/v2"
)

// https://stackoverflow.com/questions/9398739/working-with-function-types-in-go
type Active func(c *colly.Collector, e *colly.HTMLElement, data interface{}) (next bool, result interface{})

var ActiveOpter = map[string]Active{
	"a": Active(func(c *colly.Collector, e *colly.HTMLElement, data interface{}) (next bool, result interface{}) {
		return false, nil
	}),
	"cve": Active(func(c *colly.Collector, e *colly.HTMLElement, data interface{}) (next bool, result interface{}) {
		return false, nil
	}),
}
