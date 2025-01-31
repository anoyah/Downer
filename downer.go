package main

import (
	"flag"

	"github.com/anoyah/downer/core"
)

var (
	archFlag    = flag.String("arch", "linux/amd64", "--arch linux/amd64")
	imageFlag   = flag.String("image", "", "--image nginx:alpine")
	proxyFlag   = flag.String("proxy", "", "--proxy http://127.0.0.1.7890")
	verboseFlag = flag.Bool("verbose", false, "--verbose")
	outputFlag  = flag.String("output", "", "--output ./images/xx.tar.gz")
)

func main() {
	flag.Parse()

	var debug bool
	if verboseFlag != nil && *verboseFlag {
		debug = true
	}

	d, err := core.NewDp(&core.Config{
		Arch:   *archFlag,
		Name:   *imageFlag,
		Proxy:  *proxyFlag,
		Debug:  debug,
		Output: *outputFlag,
	})
	if err != nil {
		panic(err)
	}

	if err := d.Run(); err != nil {
		panic(err)
	}
}
