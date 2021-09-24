package saiyan

import (
	"net/http"
)

const MaxLevel = 127

type (
	dataTree map[string]interface{}
	fileTree map[string]interface{}
)

func parseData(r *http.Request) dataTree {
	data := make(dataTree)
	if r.PostForm != nil {
		for k, v := range r.PostForm {
			data.push(k, v)
		}
	}
	if r.MultipartForm != nil {
		for k, v := range r.MultipartForm.Value {
			data.push(k, v)
		}
	}
	return data
}

func (d dataTree) push(k string, v []string) {
	keys := fetchIndexes(k)
	if len(keys) <= MaxLevel {
		d.mount(keys, v)
	}
}

func (d dataTree) mount(i []string, v []string) {
	if len(i) == 1 {
		d[i[0]] = v[len(v)-1]
		return
	}
	if len(i) == 2 && i[1] == "" {
		d[i[0]] = v
		return
	}
	if p, ok := d[i[0]]; ok {
		p.(dataTree).mount(i[1:], v)
		return
	}
	d[i[0]] = make(dataTree)
	d[i[0]].(dataTree).mount(i[1:], v)
}

func (d fileTree) push(k string, v []*FileUpload) {
	keys := fetchIndexes(k)
	if len(keys) <= MaxLevel {
		d.mount(keys, v)
	}
}

func (d fileTree) mount(i []string, v []*FileUpload) {
	if len(i) == 1 {
		d[i[0]] = v[0]
		return
	}
	if len(i) == 2 && i[1] == "" {
		d[i[0]] = v
		return
	}
	if p, ok := d[i[0]]; ok {
		p.(fileTree).mount(i[1:], v)
		return
	}
	d[i[0]] = make(fileTree)
	d[i[0]].(fileTree).mount(i[1:], v)
}

func fetchIndexes(s string) []string {
	var (
		pos  int
		ch   string
		keys = make([]string, 1)
	)
	for _, c := range s {
		ch = string(c)
		switch ch {
		case " ":
			continue
		case "[":
			pos = 1
			continue
		case "]":
			if pos == 1 {
				keys = append(keys, "")
			}
			pos = 2
		default:
			if pos == 1 || pos == 2 {
				keys = append(keys, "")
			}
			keys[len(keys)-1] += ch
			pos = 0
		}
	}
	return keys
}
