package saiyan

import (
	"archive/zip"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/sohaha/zlsgo/zhttp"
	"github.com/sohaha/zlsgo/zlog"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sohaha/zlsgo/zfile"
	"github.com/sohaha/zlsgo/zjson"
	"github.com/sohaha/zlsgo/zshell"
	"github.com/sohaha/zlsgo/zutil"
)

const (
	BufferSize          = 10485760 // 10 Mb
	VERSUION            = "v1.0.1"
	HttpErrKey          = "Saiyan_Err"
	PayloadEmpty   byte = 2
	PayloadRaw     byte = 4
	PayloadError   byte = 8
	PayloadControl byte = 16
)

var (
	ErrExecTimeout  = errors.New("maximum execution time")
	ErrProcessDeath = errors.New("process death")
	ErrWorkerBusy   = errors.New("worker busy")
	ErrWorkerClose  = errors.New("worker close")
	ErrWorkerFailed = errors.New("failed to initialize worker")
)

var log = zlog.New()

func init() {
	log.ResetFlags(zlog.BitLevel)
}

type Prefix [17]byte

func NewPrefix() Prefix {
	return [17]byte{}
}

func (p Prefix) String() string {
	return fmt.Sprintf("[%08b: %v]", p.Flags(), p.Size())
}

func (p Prefix) Flags() byte {
	return p[0]
}

func (p Prefix) HasFlag(flag byte) bool {
	return p[0]&flag == flag
}

func (p Prefix) Valid() bool {
	return binary.LittleEndian.Uint64(p[1:]) == binary.BigEndian.Uint64(p[9:])
}

func (p Prefix) Size() uint64 {
	if p.HasFlag(PayloadEmpty) {
		return 0
	}

	return binary.LittleEndian.Uint64(p[1:])
}

func (p Prefix) HasPayload() bool {
	return p.Size() != 0
}

func (p Prefix) WithFlag(flag byte) Prefix {
	p[0] = p[0] | flag
	return p
}

func (p Prefix) WithFlags(flags byte) Prefix {
	p[0] = flags
	return p
}

func (p Prefix) WithSize(size uint64) Prefix {
	binary.LittleEndian.PutUint64(p[1:], size)
	binary.BigEndian.PutUint64(p[9:], size)
	return p
}

func mainWork(e *Engine) (*exec.Cmd, error) {
	p, err := e.newWorker(false)
	if err != nil {
		return nil, err
	}
	cmd := p.Cmd
	errTip := fmt.Errorf("php service is illegal. Docs: %v\n", "https://docs.73zls.com/zlsgo/#/bd5f3e29-b914-4d20-aa48-5f7c9d629d2b")
	pid := cmd.Process.Pid
	json, _ := zjson.SetBytes([]byte(""), "pid", pid)
	data, _, err := p.send(json, PayloadEmpty, 2)
	if err != nil {
		code, _, errStr, _ := zshell.Run(e.phpPath + " " + e.conf.Command)
		if code != 0 && errStr != "" {
			errTip = errors.New(errStr)
		}
		return cmd, errTip
	}
	rPid := zjson.GetBytes(data, "pid").Int()
	if pid != rPid {
		return cmd, errTip
	}
	go func() {
		err := cmd.Wait()
		if err == nil {
			e.stop <- struct{}{}
			return
		}
		errMsg := err.Error()
		if strings.Contains(errMsg, "interrupt") || strings.Contains(errMsg, "exit status 1") || strings.Contains(errMsg, "terminated") {
			e.restart <- struct{}{}
		} else {
			e.stop <- struct{}{}
		}
	}()
	go func() {
		select {
		case <-e.restart:
			e.release(0)
			cmd, err := mainWork(e)
			if err != nil {
				e.stop <- struct{}{}
				return
			}
			e.mainCmd = cmd
		case <-e.stop:
			e.release(0)
			close(e.pool)
			if e.mainCmd != nil {
				_ = e.mainCmd.Process.Kill()
			}
		}
	}()
	return cmd, nil
}

func getPHP(phpPath string, autoInstall bool) (string, error) {
	isWin := zutil.IsWin()
	php := zutil.IfVal(isWin, "php.exe", "php").(string)
	phpPaths := []string{php, zfile.RealPath("./bin/" + php), zfile.RealPath("./bin/php/" + php)}
	if phpPath != "" {
		phpPaths = append([]string{phpPath}, phpPaths...)
	}
	for _, v := range phpPaths {
		code, _, _, _ := zshell.Run(v + " -v")
		if code == 0 {
			return v, nil
		}
	}
	// todo Support for automatic installation of php on Windows
	if isWin && autoInstall {
		err := download()
		if err != nil {
			return "", err
		}
		return getPHP(zfile.RealPath("./php_bin/php.exe"), false)
	}
	return "", errors.New("please install PHP first")
}

func download() (err error) {
	p := zfile.RealPath("php_bin", true)
	u := "https://windows.php.net/downloads/releases/php-7.4.16-nts-Win32-vc15-x64.zip"
	log.Tipsf("downloading and installing PHP\n")
	var res *zhttp.Res
	bar := NewBar(zlog.ColorTextWrap(zlog.ColorWhite, "[TIPS] ") + " downloading: ")
	http := zhttp.New()
	http.SetTimeout(time.Minute * 6)
	defer func() {
		fmt.Println()
	}()
	res, err = http.Get(u, zhttp.DownloadProgress(func(current, total int64) {
		bar.Play(float64(current) / float64(total) * 100)
	}))
	if err != nil {
		return errors.New("download failed, please install it manually")
	}
	path := zfile.RealPath(zfile.TmpPath("php"), true) + "php.zip"
	err = res.ToFile(path)
	if err != nil {
		return
	}
	defer zfile.Rmdir(path)
	r, err := zip.OpenReader(path)
	if err != nil {
		return
	}
	defer r.Close()
	for _, innerFile := range r.File {
		info := innerFile.FileInfo()
		if info.IsDir() {
			err = os.MkdirAll(p+innerFile.Name, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}
		srcFile, err := innerFile.Open()
		if err != nil {
			continue
		}
		defer srcFile.Close()
		newFile, err := os.Create(p + innerFile.Name)
		if err != nil {
			continue
		}
		_, _ = io.Copy(newFile, srcFile)
		_ = newFile.Close()
	}
	return nil
}
