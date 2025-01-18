package main

import (
	"os"
	"testing"

	"downer/core"
)

func TestDownload(t *testing.T) {
	d, err := core.NewDp(&core.Config{
		Arch:  "linux/arm64",
		Name:  "nginx:alpine",
		Proxy: "http://127.0.0.1:7890",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err = d.Run(); err != nil {
		t.Fatal(err)
	}

	wantedFilename := "./nginx-alpine-linux-arm64.tar.gz"
	t.Log("download success!")
	fi, err := os.Stat(wantedFilename)
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(wantedFilename)

	if fi.Size() > 0 {
		t.Log("finished")
	}
}
