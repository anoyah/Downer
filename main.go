package main

import (
	"flag"

	"downer/core"
)

var (
	archFlag  = flag.String("arch", "linux/amd64", "--arch linux/amd64")
	imageFlag = flag.String("image", "", "--image nginx:alpine")
)

func main() {
	flag.Parse()

	d, err := core.NewDp(*archFlag, *imageFlag)
	if err != nil {
		panic(err)
	}

	if err := d.Run(); err != nil {
		panic(err)
	}
}
