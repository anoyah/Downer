package core

import "fmt"

type AuthMD struct {
	AuthUrl string `json:"auth_url"`
	Service string `json:"service"`
	Scope   string `json:"scope"`
}

func (a *AuthMD) buildAuthUrl() string {
	return fmt.Sprintf("%s?service=%s&scope=%s", a.AuthUrl, a.Service, a.Scope)
}
