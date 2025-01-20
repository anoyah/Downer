package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anoyah/downer/compress"
	"github.com/anoyah/downer/http"
	"github.com/anoyah/downer/tools"
)

var (
	arch2Manifest   = map[string]*http.Manifest{}
	tempDir         = "images"
	containerConfig = map[string]any{
		"Hostname":     "",
		"Domainname":   "",
		"User":         "",
		"AttachStdin":  false,
		"AttachStdout": false,
		"AttachStderr": false,
		"Tty":          false,
		"OpenStdin":    false,
		"StdinOnce":    false,
		"Env":          nil,
		"Cmd":          nil,
		"Image":        "",
		"Volumes":      nil,
		"WorkingDir":   "",
		"Entrypoint":   nil,
		"OnBuild":      nil,
		"Labels":       nil,
	}
)

const (
	registryUrl         = "https://registry-1.docker.io/v2/library/%s/%s/%s"
	OutFileTmpl         = "%s-%s-%s.tar.gz"
	repositoriesContent = `{"%s":{"%s":"%s"}}`

	UNKNOWN         = "unknown"
	WwwAuthenticate = "Www-Authenticate"
	AcceptRefresh   = "application/vnd.docker.distribution.manifest.v2+json,application/vnd.docker.distribution.manifest.list.v2+json"
	DefaultTime     = "1970-01-01T08:00:00+08:00"
	LayerName       = "/layer.tar"
	Repositories    = "repositories"
	ManifestJson    = "manifest.json"
	MANIFESTS       = "manifests"
	BLOBS           = "blobs"
	VERSION         = "VERSION"
)

type (
	Dp struct {
		client *http.Client
		log    *logger
		image  *Image
	}

	Image struct {
		name   string
		tag    string
		arch   string
		output string
	}
)

func init() {
	var err error
	tempDir, err = os.MkdirTemp("", "DockerDown")
	if err != nil {
		os.Exit(0)
	}

	fmt.Printf("created temporary folder: %s\n", tempDir)
}

type Config struct {
	Arch   string
	Name   string
	Proxy  string
	Debug  bool
	Output string
}

// NewDp ...
func NewDp(cfg *Config) (*Dp, error) {
	log, err := newLogger(cfg.Debug)
	if err != nil {
		panic(err)
	}

	log.Debugf("get arch: %s", cfg.Arch)

	name, tag := tools.ParseImage(cfg.Name)
	log.Debugf("image: %s -> tag: %s", name, tag)

	// TODO create specify directory
	defualtClientOpts := []http.ClientOption{http.WithProxy(cfg.Proxy)}
	client, err := http.NewClient(defualtClientOpts...)
	if err != nil {
		log.Errorf("create client error: %s", err)
		panic(err)
	}

	if cfg.Output != "" {
		// checkout output whether exist
		if err := tools.CreatePathWithFilepath(cfg.Output); err != nil {
			if errors.Is(err, tools.ErrFileExist) {
				return nil, err
			}
		}
	}

	return &Dp{
		client: client,
		log:    log,
		image: &Image{
			name:   name,
			tag:    tag,
			arch:   cfg.Arch,
			output: cfg.Output,
		},
	}, nil
}

// Run main process to download image with http request
func (d *Dp) Run() error {
	clean, err := d.init()
	if err != nil {
		d.log.Error(err)
		return err
	}
	defer clean()

	meta, err := d.getRequstMeta(d.image.name, d.image.tag)
	if err != nil {
		d.log.Errorf("get request meta: %s", err)
		panic(err)
	}
	d.log.Debugf("auth meta: %#v", meta)

	token, err := d.getTokenInfo(meta)
	if err != nil {
		d.log.Errorf("get token info: ", err)
		panic(err)
	}
	d.log.Debugf("token info: %#v", token)

	b, err := d.refreshToken(token)
	if err != nil {
		d.log.Errorf("fresh token: ", err)
		panic(err)
	}

	if err = parseManifests(b); err != nil {
		d.log.Errorf("parse manifests: ", err)
		panic(err)
	}

	// TODO 解析Manifest
	for k, v := range arch2Manifest {
		d.log.Debugf("%s: %+v\n", k, v)
	}

	// TODO choose arch with people
	manifest, ok := arch2Manifest[d.image.arch]
	if !ok {
		d.log.Infof("don't found arch: %s", d.image.arch)
		return err
	}

	digestSource, err := d.getDigestSource(manifest.Digest, token.Token)
	if err != nil {
		d.log.Errorf("get digest source: ", err)
		panic(err)
	}
	layers := digestSource.Layers

	fmt.Printf("load layers length: %d, start download...\n", len(layers))

	digestModel, err := d.saveDegistFile(digestSource.Config.Digest, digestSource.Config.MediaType, token.Token)
	if err != nil {
		d.log.Errorf("get blobs: ", err)
		panic(err)
	}

	var parentID string
	layersID := make([]string, 0, len(layers))
	for index, layer := range layers {
		data := make(map[string]any)

		tempJson := digestModel

		currentID := tools.GenLayerID(parentID, layer.Digest)
		layersID = append(layersID, fmt.Sprintf("%s%s", currentID, LayerName))

		if index == len(layers)-1 {
			delete(tempJson, "history")
			delete(tempJson, "rootfs")
			data = tempJson
		} else {
			data["created"] = DefaultTime
		}
		data["container_config"] = containerConfig
		data["id"] = currentID
		if parentID != "" {
			data["parent"] = parentID
		}
		parentID = currentID

		fmt.Printf("downloading %d/%d: %s\n", index+1, len(layers), layer.Digest[7:])
		if err = d.saveSingleLayer(currentID, layer.Digest, layer.MediaType, token.Token, data); err != nil {
			d.log.Errorf("save single layer: ", err)
			return err
		}
	}

	repoFile, err := os.Create(d.buildSavePath(Repositories))
	if err != nil {
		d.log.Error(err)
		return err
	}
	defer repoFile.Close()

	_, err = repoFile.WriteString(fmt.Sprintf(repositoriesContent, d.image.name, d.image.tag, parentID))
	if err != nil {
		d.log.Error(err)
		return err
	}

	manifests := make([]http.RootManifest, 1)
	manifests[0].Config = fmt.Sprintf("%s.json", digestSource.Config.Digest[7:])
	manifests[0].RepoTags = []string{fmt.Sprintf("%s:%s", d.image.name, d.image.tag)}
	manifests[0].Layers = layersID

	manifestJson, err := os.Create(d.buildSavePath(ManifestJson))
	if err != nil {
		d.log.Error(err)
		return err
	}
	defer manifestJson.Close()

	c, err := json.Marshal(manifests)
	if err != nil {
		d.log.Error(err)
		return err
	}
	_, err = manifestJson.Write(c)
	if err != nil {
		d.log.Error(err)
		return err
	}

	fmt.Printf("start merge all layers...\n")
	savedFilePath, err := d.compress()
	if err != nil {
		return err
	}

	fmt.Printf("exported images: %s\n", savedFilePath)
	fmt.Printf("you can use `docker load -i %s` to load to Docker\n", savedFilePath)

	return nil
}

// compress folder with tar and gzip
func (d *Dp) compress() (string, error) {
	var output string
	if d.image.output != "" {
		output = d.image.output
	} else {
		output = fmt.Sprintf(OutFileTmpl, d.image.name, d.image.tag, strings.ReplaceAll(d.image.arch, "/", "-"))
	}

	if err := compress.Build(d.getDefaultPath(), output); err != nil {
		d.log.Error(err)
		return "", err
	}

	return output, nil
}

func (d *Dp) init() (func() error, error) {
	if err := tools.CreateDirWithPath((d.getDefaultPath())); err != nil {
		return nil, err
	}

	return func() error {
		defer fmt.Printf("removed temporary folder: %s\n", tempDir)
		return os.RemoveAll(tempDir)
	}, nil
}

func (d *Dp) saveSingleLayer(id string, digest string, mediaType string, token string, data map[string]any) error {
	path := d.buildSavePath(id)
	if err := tools.CreateDirWithPath(path); err != nil {
		d.log.Errorf("create directory: ", err)
		return err
	}

	f, err := os.Create(filepath.Join(path, VERSION))
	if err != nil {
		d.log.Errorf(err.Error())
		return err
	}
	defer f.Close()
	f.WriteString("1.0")

	blogResponse, err := d.getBlob(digest, mediaType, token)
	if err != nil {
		d.log.Errorf(err.Error())
		return err
	}

	layerFile, err := os.Create(fmt.Sprintf("%s/%s", path, "layer.tar"))
	if err != nil {
		d.log.Errorf(err.Error())
	}
	defer layerFile.Close()

	layerFile.Write(blogResponse.Body())

	dataMarshaled, err := json.Marshal(data)
	if err != nil {
		return err
	}

	jsonFile, err := os.Create(fmt.Sprintf("%s/json", path))
	if err != nil {
		d.log.Errorf(err.Error())
	}
	defer jsonFile.Close()

	jsonFile.Write(dataMarshaled)

	return nil
}

// saveDegistFile
func (d *Dp) saveDegistFile(digest, mediaType, token string) (map[string]any, error) {
	r, err := d.getBlob(digest, mediaType, token)
	if err != nil {
		d.log.Errorf("get registery request: ", err)
		return nil, err
	}

	var digestModel map[string]any
	if err := json.Unmarshal(r.Body(), &digestModel); err != nil {
		d.log.Errorf("unmarshal digest model: ", err)
		return nil, err
	}

	return digestModel, d.saveWithPath(r.Body(), fmt.Sprintf("%s.json", digest[7:]))
}

func (d *Dp) getBlob(digest, mediaType, token string) (*http.Response, error) {
	response, err := d.buildRegistryRequest(BLOBS, d.image.name, digest, http.SetAccept(mediaType), http.SetAuthToken(token))
	if err != nil {
		d.log.Errorf("get registery request: ", err)
		return nil, err
	}

	return response, nil
}

func (d *Dp) saveWithPath(content []byte, path string) error {
	targetPath := filepath.Join(d.getDefaultPath(), path)
	f, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Write(content)
	return nil
}

func (d *Dp) getDigestSource(digest, token string) (*http.AutoGenerated, error) {
	r, err := d.manifestsRequest(digest, http.SetAccept("application/vnd.docker.distribution.manifest.v2+json"), http.SetAuthToken(token))
	if err != nil {
		d.log.Errorf("manifestsRequest: ", err)
		return nil, err
	}

	var data http.AutoGenerated
	if err := json.Unmarshal(r.Body(), &data); err != nil {
		d.log.Errorf("manifestsRequest: ", err)
		return nil, err
	}

	d.log.Debugf("%s", r.Body())
	return &data, nil
}

func (d *Dp) refreshToken(token *http.TokenInfo) ([]byte, error) {
	r, err := d.buildRegistryRequest(MANIFESTS,
		d.image.name,
		d.image.tag,
		http.SetAccept(AcceptRefresh),
		http.SetAuthToken(token.Token),
	)
	if err != nil {
		d.log.Errorf("get registery request: ", err)
		return nil, err
	}

	return r.Body(), nil
}

func (d *Dp) getTokenInfo(meta *http.AuthMD) (*http.TokenInfo, error) {
	authUrl := meta.BuildAuthUrl()

	r, err := d.client.Do(context.Background(), authUrl)
	if err != nil {
		d.log.Error(err)
		return nil, err
	}

	var token http.TokenInfo
	if err := json.Unmarshal(r.Body(), &token); err != nil {
		d.log.Error(err)
		return nil, err
	}

	return &token, nil
}

func (d *Dp) getRequstMeta(image, tag string) (*http.AuthMD, error) {
	r, err := d.buildRegistryRequest(MANIFESTS, image, tag)
	if err != nil {
		d.log.Errorf("get registery request: %s", err)
		return nil, err
	}

	var md http.AuthMD
	authenticate := r.Header.Get(WwwAuthenticate)
	authenticateSplited := strings.Split(authenticate, "\"")
	if len(authenticateSplited) > 2 {
		md.AuthUrl = authenticateSplited[1]
	}
	if len(authenticateSplited) > 4 {
		md.Service = authenticateSplited[3]
	}
	if len(authenticateSplited) > 6 {
		md.Scope = authenticateSplited[5]
	}

	return &md, err
}

func (d *Dp) manifestsRequest(digest string, opts ...http.HeaderOption) (*http.Response, error) {
	r, err := d.buildRegistryRequest(MANIFESTS, d.image.name, digest, opts...)
	if err != nil {
		d.log.Errorf("get registery request: ", err)
		return nil, err
	}
	return r, nil
}

func (d *Dp) buildRegistryRequest(kind string, image, tag string, opts ...http.HeaderOption) (*http.Response, error) {
	url := fmt.Sprintf(registryUrl, image, kind, tag)
	d.log.Debugf("send request with url: %s", url)

	return d.client.Do(context.Background(), url, opts...)
}

func (d *Dp) buildSavePath(path string) string {
	return filepath.Join(tempDir, fmt.Sprintf("%s-%s-%s", d.image.name, d.image.tag, strings.ReplaceAll(d.image.arch, "/", "-")), path)
}

func (d *Dp) getDefaultPath() string {
	return d.buildSavePath("")
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

			arch2Manifest[fmt.Sprintf("%s/%s", os, architecture)] = &http.Manifest{
				Arch:      architecture.(string),
				Digest:    manifestItem["digest"].(string),
				MediaType: manifestItem["mediaType"].(string),
			}
		}
	}
	return nil
}
