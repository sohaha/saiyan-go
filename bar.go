package saiyan

import (
	"fmt"
	"math"
	"strconv"

	"github.com/sohaha/zlsgo/zstring"
	"github.com/sohaha/zlsgo/zutil"
)

type Bar struct {
	percent     int
	cur         float64
	total       int
	rate        string
	width       int
	graph       string
	tip         string
	placeholder int
}

func NewBar(tip string, graph ...string) *Bar {
	graphText := "#"
	if len(graph) > 0 {
		graphText = graph[0]
	}
	placeholder := 10 + zstring.Len(tip)
	return &Bar{graph: graphText, tip: tip, placeholder: placeholder, total: 100}
}

func (bar *Bar) getPercent() int {
	return int(float32(bar.cur) / float32(bar.total) * 100)
}

func (bar *Bar) Play(cur float64) {
	bar.cur = zutil.IfVal(cur < 0, 0, cur).(float64)
	bar.percent = bar.getPercent()
	w := 80
	nw := w - bar.placeholder
	padLen := math.Round(float64(nw) / 100 * bar.cur)
	bar.rate = zstring.Pad("", int(padLen), bar.graph, zstring.PadLeft)
	bar.width = nw
	fmt.Printf("\r%s[%-"+strconv.Itoa(bar.width)+"s]%3d%%", bar.tip, bar.rate, bar.percent)
}

func (bar *Bar) Done() {
	fmt.Print(zstring.Pad("\r", bar.width+bar.placeholder, " ", zstring.PadRight) + "\r")
}
