package saiyan

import (
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"sync"

	"github.com/sohaha/zlsgo/zfile"
)

const (
	UploadErrorOK        = 0
	UploadErrorNoFile    = 4
	UploadErrorNoTmpDir  = 5
	UploadErrorCantWrite = 6
	UploadErrorExtension = 7
)

type FileUpload struct {
	Name string `json:"name"`
	Mime string `json:"mime"`
	Size int64  `json:"size"`
	// See http://php.net/manual/en/features.file-upload.errors.php
	Error        int    `json:"error"`
	TempFilename string `json:"tmp_name"`
	header       *multipart.FileHeader
}

func NewUpload(f *multipart.FileHeader) *FileUpload {
	return &FileUpload{
		Name:   f.Filename,
		Mime:   f.Header.Get("Content-Type"),
		Error:  UploadErrorOK,
		header: f,
	}
}

func parseUploads(r *http.Request) ([]*FileUpload, *fileTree) {
	var (
		wg   sync.WaitGroup
		list []*FileUpload
		data = &fileTree{}
	)
	for k, v := range r.MultipartForm.File {
		wg.Add(1)
		files := make([]*FileUpload, 0, len(v))
		for _, f := range v {
			files = append(files, NewUpload(f))
		}
		list = append(list, files...)
		data.push(k, files)
		go func(files []*FileUpload) {
			for _, file := range files {
				_ = file.Open()
			}
			wg.Done()
		}(files)
	}
	wg.Wait()
	return list, data
}

func (f *FileUpload) Open() (err error) {
	file, err := f.header.Open()
	if err != nil {
		f.Error = UploadErrorNoFile
		return
	}
	defer file.Close()
	tmpDir, err := ioutil.TempDir("", "Saiyan_")
	if err != nil {
		tmpDir = zfile.RealPathMkdir("./tmp/")
	}
	tmp, err := ioutil.TempFile(tmpDir, "upload")
	if err != nil {
		f.Error = UploadErrorNoTmpDir
		return
	}
	f.TempFilename = tmp.Name()
	defer tmp.Close()
	if f.Size, err = io.Copy(tmp, file); err != nil {
		f.Error = UploadErrorCantWrite
	}
	return nil
}

func (f *FileUpload) Clear() {
	if f.TempFilename != "" && zfile.FileExist(f.TempFilename) {
		_ = os.Remove(f.TempFilename)
	}
}
