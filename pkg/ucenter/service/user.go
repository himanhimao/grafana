package service

import (
	"github.com/himanhimao/grafana/pkg/ucenter/model"
)


const (
	RPC_USER = "user/rpc"
	METHOD_GET_USER_BY_ID = "getUserById"
)

type User struct {
	Id            int16
	DepamentId    int16
	OfficeId      int16
	Sex           int8
	Number        string
	Account       string
	Name          string
	Email         string
	Phone         string
	JobStatus     int8
	CreatedTimeTs int
	UpdateTimeTs  int
}

type Uid interface{}

func GetUserById(c *model.Client, uid Uid) (User, error){
	params := make(map[string]interface{})
	params["id"] = uid
	var user User
	err := c.Call(RPC_USER, METHOD_GET_USER_BY_ID, params, &user)
	return user, err
}

