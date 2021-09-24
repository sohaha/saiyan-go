package saiyan

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/sohaha/zlsgo/znet"
)

type Request struct {
	RemoteAddr  string            `json:"remoteAddr"`
	Protocol    string            `json:"protocol"`
	Method      string            `json:"method"`
	URI         string            `json:"uri"`
	Header      http.Header       `json:"headers"`
	Cookies     map[string]string `json:"cookies"`
	RawQuery    string            `json:"rawQuery"`
	Parsed      bool              `json:"parsed"`
	Uploads     *fileTree         `json:"uploads"`
	UploadFiles []*FileUpload     `json:"-"`
	body        interface{}
}

const (
	defaultMaxMemory = 32 << 20 // 32 MB
	contentNone      = iota + 900
	contentStream
	contentMultipart
	contentFormData
)

func (r *Request) contentType() int {
	if r.Method == "HEAD" || r.Method == "OPTIONS" {
		return contentNone
	}
	ct := r.Header.Get("content-type")
	if strings.Contains(ct, "application/x-www-form-urlencoded") {
		return contentFormData
	}

	if strings.Contains(ct, "multipart/form-data") {
		return contentMultipart
	}

	return contentStream
}

func (e *Engine) newRequest(c *znet.Context, r *http.Request, v *saiyanVar) (err error) {
	req := v.request
	req.RemoteAddr = c.GetClientIP()
	req.Method = r.Method
	req.URI = r.URL.Path
	req.Header = r.Header
	req.body = ""
	req.Parsed = false
	req.Cookies = map[string]string{}
	req.RawQuery = r.URL.RawQuery
	req.UploadFiles = []*FileUpload{}
	req.Uploads = &fileTree{}

	host := strings.Split(c.Host(), "://")
	req.Protocol = "HTTP/1.0"
	if host[0] == "https" {
		req.Protocol = "HTTP/2.0"
	}
	req.Header.Add("host", host[1])

	if e.conf.TrimPrefix != "" {
		req.URI = strings.TrimPrefix(req.URI, e.conf.TrimPrefix)
	}

	for _, c := range r.Cookies() {
		if v, err := url.QueryUnescape(c.Value); err == nil {
			req.Cookies[c.Name] = v
		}
	}

	switch req.contentType() {
	case contentNone:
		return nil
	case contentStream:
		body, err := ioutil.ReadAll(r.Body)
		if err == nil {
			req.body = body
		}
		return err
	case contentMultipart:
		if err = r.ParseMultipartForm(defaultMaxMemory); err != nil {
			return err
		}
		req.UploadFiles, req.Uploads = parseUploads(r)
		fallthrough
	case contentFormData:
		if err = r.ParseForm(); err != nil {
			return err
		}
		req.body = parseData(r)
	}
	req.Parsed = true
	return nil
}
