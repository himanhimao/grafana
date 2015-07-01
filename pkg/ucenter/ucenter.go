package ucenter

import (
	"net/http"
	"strings"
	"github.com/himanhimao/grafana/pkg/ucenter/model"
)

// Create new Client.
func NewClient(addr string, key string, secret string) * model.Client {
	return &model.Client{
		Addr: strings.TrimRight(addr, model.HTTP_PATH),
		Key: key,
		Secret : secret,
		Http: &http.Client{},
	}
}







