package main

import (
	"flag"

	"downer/core"
)

var (
	archFlag  = flag.String("arch", "linux/amd64", "--arch linux/amd64")
	imageFlag = flag.String("image", "", "--image nginx:alpine")
	proxyFlag = flag.String("proxy", "", "--proxy http://127.0.0.1.7890")
)

func main() {
	flag.Parse()

	d, err := core.NewDp(&core.Config{
		Arch:  *archFlag,
		Name:  *imageFlag,
		Proxy: *proxyFlag,
	})
	if err != nil {
		panic(err)
	}

	if err := d.Run(); err != nil {
		panic(err)
	}
}
