package main

import (
	"github.com/adrian-grassl/inx-notarizer/components/app"
	"github.com/adrian-grassl/inx-notarizer/pkg/notarizer"
)

func init() {
	notarizer.LoadEnvVariables()
}

func main() {
	app.App().Run()
}
