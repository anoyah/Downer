package tools

import "strings"

// TODO needs to optimize deal with diffrent type.
// ParseImage parse name to image and tag
func ParseImage(name string) (image string, tag string) {
	imageSplited := strings.Split(name, ":")
	if len(imageSplited) > 1 {
		return imageSplited[0], imageSplited[1]
	}

	return name, "latest"
}
