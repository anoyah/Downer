package core

import (
	"encoding/json"
	"fmt"
)

type Manifest struct {
	Arch      string `json:"arch"`
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
}

type RootManifest struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
}

func parseManifests(manifests []byte) error {
	var tem map[string]any

	if err := json.Unmarshal(manifests, &tem); err != nil {
		return err
	}
	if data, ok := tem["manifests"]; ok {
		for _, item := range data.([]any) {
			manifestItem := item.(map[string]any)
			platform := manifestItem["platform"].(map[string]any)
			os := platform["os"]
			architecture := platform["architecture"]
			if os == UNKNOWN || architecture == UNKNOWN {
				continue
			}

			Arch2Manifest[fmt.Sprintf("%s/%s", os, architecture)] = &Manifest{
				Arch:      architecture.(string),
				Digest:    manifestItem["digest"].(string),
				MediaType: manifestItem["mediaType"].(string),
			}
		}
	}
	return nil
}
