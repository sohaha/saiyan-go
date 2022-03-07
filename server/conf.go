package main

import (
	"github.com/sohaha/zlsgo/zutil"
	"github.com/zlsgo/conf"
)

type projectConf struct {
	Debug bool
}

func initConf() *projectConf {
	return zutil.Once(func() interface{} {
		cfg := conf.New(*projectPath + "/zls.ini")
		err := cfg.Read()
		pconf := &projectConf{}
		if err != nil {
			return pconf
		}
		pconf.Debug = cfg.Core.GetBool("base.debug")
		return pconf
	})().(*projectConf)
}
