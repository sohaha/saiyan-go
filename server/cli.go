package main

import (
	"github.com/sohaha/zlsgo/zcli"
	"github.com/sohaha/zlsgo/zfile"
)

func initDefaultFlags() {
	projectPath = zcli.SetVar("path", "Project Path").String(zfile.RealPath("."))
}

func initRunFlags() {
	port = zcli.SetVar("port", "Server Port").String(":8181")
	max = zcli.SetVar("max", "Max Worker Quantity").Int()
}
