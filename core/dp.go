package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"downer/compress"
	"downer/http"
	"downer/tools"
)

var (
	Arch2Manifest   = map[string]*Manifest{}
	tempDir         = "images"
	ContainerConfig = map[string]any{
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
	UNKNOWN             = "unknown"
	registryUrl         = "https://registry-1.docker.io/v2/library/%s/%s/%s"
	WwwAuthenticate     = "Www-Authenticate"
	AcceptRefresh       = "application/vnd.docker.distribution.manifest.v2+json,application/vnd.docker.distribution.manifest.list.v2+json"
	DefaultTime         = "1970-01-01T08:00:00+08:00"
	LayerName           = "/layer.tar"
	Repositories        = "repositories"
	RepositoriesContent = `{"%s":{"%s":"%s"}}`
	ManifestJson        = "manifest.json"
	OutFileTmpl         = "%s-%s-%s.tar.gz"
)

type (
	Dp struct {
		client *http.Client
		log    *logger
		image  *Image
	}

	Image struct {
		name string
		tag  string
		arch string
	}
)

func init() {
	var err error
	tempDir, err = os.MkdirTemp("", "DockerDown")
	if err != nil {
		os.Exit(0)
	}

	fmt.Printf("created temp dir: %s\n", tempDir)
}

func NewDp(arch, name string) (*Dp, error) {
	log, err := newLogger()
	if err != nil {
		panic(err)
	}

	log.Debugf("get arch: %s", arch)

	name, tag := parseImage(name)
	log.Debugf("image: %s -> tag: %s", name, tag)

	// TODO create specify directory
	client, err := http.NewClient()
	if err != nil {
		log.Errorf("create client error: %s", err)
		panic(err)
	}

	return &Dp{
		client: client,
		log:    log,
		image: &Image{
			name: name,
			tag:  tag,
			arch: arch,
		},
	}, nil
}

func (d *Dp) Run() error {
	if err := d.init(); err != nil {
		d.log.Error(err)
		return err
	}

	defer os.RemoveAll(tempDir)

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
	for k, v := range Arch2Manifest {
		d.log.Debugf("%s: %+v\n", k, v)
	}

	// TODO choose arch with people
	manifest, ok := Arch2Manifest[d.image.arch]
	if !ok {
		d.log.Infof("don't found arch: %s", d.image.arch)
		return err
	}

	digestSource, err := d.getDigestSource(manifest.Digest, token.Token)
	if err != nil {
		d.log.Errorf("get digest source: ", err)
		panic(err)
	}

	digestModel, err := d.saveDegistFile(digestSource.Config.Digest, digestSource.Config.MediaType, token.Token)
	if err != nil {
		d.log.Errorf("get blobs: ", err)
		panic(err)
	}

	var parentID string
	layersID := make([]string, 0, len(digestSource.Layers))
	for index, layer := range digestSource.Layers {
		data := make(map[string]any)

		tempJson := digestModel

		currentID := tools.GenLayerID(parentID, layer.Digest)
		layersID = append(layersID, fmt.Sprintf("%s%s", currentID, LayerName))

		if index == len(digestSource.Layers)-1 {
			delete(tempJson, "history")
			delete(tempJson, "rootfs")
			data = tempJson
		} else {
			data["created"] = DefaultTime
		}
		data["container_config"] = ContainerConfig
		data["id"] = currentID
		if parentID != "" {
			data["parent"] = parentID
		}
		parentID = currentID

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

	_, err = repoFile.WriteString(fmt.Sprintf(RepositoriesContent, d.image.name, d.image.tag, parentID))
	if err != nil {
		d.log.Error(err)
		return err
	}

	manifests := make([]RootManifest, 1)
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

	output := fmt.Sprintf(OutFileTmpl, d.image.name, d.image.tag, strings.ReplaceAll(d.image.arch, "/", "-"))
	if err := compress.Build(d.getDefaultPath(), output); err != nil {
		d.log.Error(err)
		return err
	}

	d.log.Infof("exported images: %s", output)

	return nil
}

func (d *Dp) init() error {
	if err := tools.CreateDirWithPath((d.getDefaultPath())); err != nil {
		return err
	}
	return nil
}

func (d *Dp) saveSingleLayer(id string, digest string, mediaType string, token string, data map[string]any) error {
	path := d.buildSavePath(id)
	if err := tools.CreateDirWithPath(path); err != nil {
		d.log.Errorf("create directory: ", err)
		return err
	}

	f, err := os.Create(filepath.Join(path, "VERSION"))
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
	response, err := d.buildRegistryRequest("blobs", d.image.name, digest, http.SetAccept(mediaType), http.SetAuthToken(token))
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

func (d *Dp) refreshToken(token *TokenInfo) ([]byte, error) {
	r, err := d.buildRegistryRequest("manifests",
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

func (d *Dp) getTokenInfo(meta *AuthMD) (*TokenInfo, error) {
	authUrl := meta.buildAuthUrl()

	r, err := d.client.Do(context.Background(), authUrl)
	if err != nil {
		d.log.Error(err)
		return nil, err
	}

	var token TokenInfo
	if err := json.Unmarshal(r.Body(), &token); err != nil {
		d.log.Error(err)
		return nil, err
	}

	return &token, nil
}

func (d *Dp) getRequstMeta(image, tag string) (*AuthMD, error) {
	r, err := d.buildRegistryRequest("manifests", image, tag)
	if err != nil {
		d.log.Errorf("get registery request: %s", err)
		return nil, err
	}

	var md AuthMD
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

func (d *Dp) manifestsRequest(digest string, opts ...http.Option) (*http.Response, error) {
	r, err := d.buildRegistryRequest("manifests", d.image.name, digest, opts...)
	if err != nil {
		d.log.Errorf("get registery request: ", err)
		return nil, err
	}
	return r, nil
}

func (d *Dp) buildRegistryRequest(kind string, image, tag string, opts ...http.Option) (*http.Response, error) {
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
