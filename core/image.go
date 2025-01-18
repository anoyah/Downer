package core

import "strings"

func parseImage(image string) (string, string) {
	imageSplited := strings.Split(image, ":")
	if len(imageSplited) > 1 {
		return imageSplited[0], imageSplited[1]
	}
	return image, "latest"
}
