package model

import (
	"net/http"
	"encoding/json"
	"fmt"
	"math/rand"
	"io"
	"crypto/md5"
	"bytes"
	"strings"
	"github.com/himanhimao/grafana/pkg/ucenter/util"
)


const (
	JSONRPC_VERSION = "2.0"
	HEADER_DATE = "date"
	HEADER_AUTHORIZATION = "authorization"
	HEADER_AUTHORIZATION_PREFIX = "PLAYCRAB"
	HTTP_PATH = "/"
)

// Client.
type Client struct {
	Http   *http.Client
	Addr   string
	Key    string
	Secret string
}

// /
type clientRequest struct {
	//jsonrpc protocol version
	Jsonrpc string `json:"jsonrpc"`
	// A String containing the name of the method to be invoked.
	Method  string `json:"method"`
	// Object to pass as request parameter to the method.
	Params  map[string]interface{} `json:"params"`
	// The request id. This can be of any type. It is used to match the
	// response with the request that it is replying to.
	Id      uint64 `json:"id"`
}

// clientResponse represents a JSON-RPC response returned to a client.
type clientResponse struct {
	Result *json.RawMessage `json:"result"`
	Error  interface{}      `json:"error"`
	Id     uint64           `json:"id"`
}

// Call RPC method with args.
func (c *Client) Call(rpc string, method string, args map[string]interface{}, res interface{}) error {
	rpc = strings.TrimLeft(rpc, HTTP_PATH)
	clientRequest, buf, err := EncodeClientRequest(method, args)
	if err != nil {
		return err
	}

	body := bytes.NewBuffer(buf)

	r, err := http.NewRequest("POST", c.Addr + HTTP_PATH + rpc, body)
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	c.generateSignature(r, clientRequest)
	resp, err := c.Http.Do(r)

	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("received status code %d with status: %s", resp.StatusCode, resp.Status)
	}

	err = DecodeClientResponse(resp.Body, res)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}


func (c *Client)generateSignature(r *http.Request, clientRequest *clientRequest) {
	dateAtom  := util.GetAtomTime()
	h := md5.New()
	io.WriteString(h, util.StringValue(clientRequest) + c.Secret + dateAtom)
	token := h.Sum(nil)
	authorization := fmt.Sprintf("%s %s:%x", HEADER_AUTHORIZATION_PREFIX, c.Key, string(token))
	r.Header.Set(HEADER_DATE, dateAtom)
	r.Header.Set(HEADER_AUTHORIZATION, authorization)
}

// EncodeClientRequest encodes parameters for a JSON-RPC client request.
func EncodeClientRequest(method string, args map[string]interface{}) (*clientRequest, []byte, error) {
	c := &clientRequest{
		Jsonrpc: JSONRPC_VERSION,
		Method: method,
		Params: args,
		Id:     uint64(rand.Int63()),
	}
	b, err := json.Marshal(c)
	return c, b, err
}

// DecodeClientResponse decodes the response body of a client request into
// the interface reply.
func DecodeClientResponse(r io.Reader, reply interface{}) error {
	var c clientResponse
	if err := json.NewDecoder(r).Decode(&c); err != nil {
		return err
	}
	if c.Error != nil {
		return fmt.Errorf("%v", c.Error)
	}
	if c.Result == nil {
		return fmt.Errorf("result is null")
	}
	return json.Unmarshal(*c.Result, reply)
}
