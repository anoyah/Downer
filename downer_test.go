package main

import (
	"os"
	"testing"

	"github.com/anoyah/downer/core"
)

func TestLibrary(t *testing.T) {
	wantedFilename := "./nginx-alpine-linux-arm64.tar.gz"

	d, err := core.NewDp(&core.Config{
		Arch:   "linux/arm64",
		Name:   "nginx:alpine",
		Proxy:  "http://127.0.0.1:7890",
		Debug:  true,
		Output: wantedFilename,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err = d.Run(); err != nil {
		t.Fatal(err)
	}

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

func TestDownload(t *testing.T) {
	wantedFilename := "./memos.tar.gz"

	d, err := core.NewDp(&core.Config{
		Arch:   "linux/amd64",
		Name:   "neosmemo/memos:stable",
		Proxy:  "http://127.0.0.1:7890",
		Output: wantedFilename,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err = d.Run(); err != nil {
		t.Fatal(err)
	}

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
