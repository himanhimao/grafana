package service

import (
	"github.com/himanhimao/grafana/pkg/ucenter/model"
)

const (
	RPC_COMMON = "common/rpc"
	METHOD_LOGIN_URL = "loginUrl"
	METHOD_LOGOUT_URL = "logoutUrl"
	METHOD_CHECK_TOKEN = "checkToken"
)
type Url string

func (u Url) String()(string) {
	return string(u)
}

func LoginUrl(c *model.Client, callback string) (Url, error) {
	params := make(map[string]interface{})
	params["callback"] = callback
	var loginUrl Url
	err := c.Call(RPC_COMMON, METHOD_LOGIN_URL, params, &loginUrl)
	return loginUrl, err
}

func LogoutUrl(c *model.Client, callback string) (Url, error) {
	params := make(map[string]interface{})
	params["callback"] = callback
	var logoutUrl Url
	err := c.Call(RPC_COMMON, METHOD_LOGOUT_URL, params, &logoutUrl)
	return logoutUrl, err
}

func CheckToken(c *model.Client, token string) (Uid, error) {
	params := make(map[string]interface{})
	params["token"] = token
	var uid Uid
	err := c.Call(RPC_COMMON, METHOD_CHECK_TOKEN, params, &uid)
	return uid, err
}