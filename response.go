package saiyan

import (
	"errors"

	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/znet"
)

type Response struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Cookies [][]interface{}     `json:"cookies"`
	Body    string              `json:"body"`
}

func (e *Engine) newResponse(c *znet.Context, v *saiyanVar, header, result []byte, p Prefix) {
	if !p.HasFlag(PayloadControl) {
		c.WithValue(HttpErrKey, errors.New("error in type"))
		return
	}

	context := v.response
	context.Type = znet.ContentTypePlain
	context.Content = nil

	j := zjson.ParseBytes(header)

	context.Code = j.Get("status").Int()
	if p.HasFlag(PayloadError) || context.Code == 0 {
		context.Code = 500
	} else {
		context.Content = result
	}

	cookies := j.Get("cookies")
	if cookies.IsArray() {
		cookies.ForEach(func(key, value zjson.Res) bool {
			v := value.Array()
			c.SetCookie(v[0].String(), v[1].String(), v[2].Int())
			return true
		})
	}

	showContext := true

	headers := j.Get("headers")
	if headers.IsObject() {
		headers.ForEach(func(key, value zjson.Res) bool {
			v := value.Array()
			k := key.String()
			if k == "Location" {
				showContext = false
				c.Redirect(v[0].String(), 301)
				return true
			}
			for i := range v {
				c.SetHeader(k, v[i].String())
			}
			return true
		})
	}

	if showContext {
		c.SetContent(context)
	}
}
