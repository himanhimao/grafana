package api

import (
	"reflect"
	"net/url"
	"github.com/himanhimao/grafana/pkg/ucenter"
	"github.com/himanhimao/grafana/pkg/ucenter/service"
	"github.com/chnlr/baseurl"
	"github.com/himanhimao/grafana/pkg/api/dtos"
	"github.com/himanhimao/grafana/pkg/bus"
	"github.com/himanhimao/grafana/pkg/log"
	"github.com/himanhimao/grafana/pkg/metrics"
	"github.com/himanhimao/grafana/pkg/middleware"
	m "github.com/himanhimao/grafana/pkg/models"
	"github.com/himanhimao/grafana/pkg/setting"
	"github.com/himanhimao/grafana/pkg/util"
	"errors"
	"strings"
)

const (
	VIEW_INDEX = "index"
)

func LoginView(c *middleware.Context) {
	if err := setIndexViewData(c); err != nil {
		c.Handle(500, "Failed to get settings", err)
		return
	}

	//尝试监测是否开启ucenterLogin
	checkUcenterLogin(c)

	settings := c.Data["Settings"].(map[string]interface{})
	settings["googleAuthEnabled"] = setting.OAuthService.Google
	settings["githubAuthEnabled"] = setting.OAuthService.GitHub
	settings["disableUserSignUp"] = !setting.AllowUserSignUp

	if !tryLoginUsingRememberCookie(c) {
		c.HTML(200, VIEW_INDEX)
		return
	}

	if redirectTo, _ := url.QueryUnescape(c.GetCookie("redirect_to")); len(redirectTo) > 0 {
		c.SetCookie("redirect_to", "", -1, setting.AppSubUrl+"/")
		c.Redirect(redirectTo)
		return
	}

	c.Redirect(setting.AppSubUrl + "/")
}

func checkUcenterLogin(c *middleware.Context) {
	if !setting.Ucenter.Enabled {
		return
	}
	uclient := ucenter.NewClient(setting.Ucenter.Api_Url, setting.Ucenter.Api_Key, setting.Ucenter.Api_Secret)
	callbackUrl := baseurl.BaseUrl(c.Req.Request) + "/login/ucenter/callback"
	loginUrl, err := service.LoginUrl(uclient, callbackUrl)

	if err != nil {
		c.Handle(500, "loginUrl get failed", err)
		return
	}
	c.Redirect(loginUrl.String())

}

func checkUcenterLogout(c *middleware.Context) {
	if !setting.Ucenter.Enabled {
		return
	}
	uclient := ucenter.NewClient(setting.Ucenter.Api_Url, setting.Ucenter.Api_Key, setting.Ucenter.Api_Secret)
	callbackUrl :=  baseurl.BaseUrl(c.Req.Request) + "/login"
	logoutUrl, err := service.LogoutUrl(uclient, callbackUrl)

	if err != nil {
		c.Handle(500, "logoutUrl get failed", err)
		return
	}
	c.Redirect(logoutUrl.String())
}

func LoginUcenterCallback(c *middleware.Context) {
	token := c.QueryStrings("token")
	if len(token) == 0 {
		c.Handle(500, "ucenter api request error", errors.New("token params error"))
		return
	}

	uclient := ucenter.NewClient(setting.Ucenter.Api_Url, setting.Ucenter.Api_Key, setting.Ucenter.Api_Secret)

	if uid, err := service.CheckToken(uclient, strings.Join(token, "")); err != nil {
		c.Handle(500, "ucenter api request error", err)
		return
	} else {
		t := reflect.TypeOf(uid)
		switch t.Kind(){
		case reflect.Bool:
			c.Handle(500, "ucenter api request error", errors.New("token is expired or invaild"))
			return
		case reflect.Float64:
			var uidFloat float64 = uid.(float64)
			var uidInt int64 = int64(uidFloat)
			if user, err := service.GetUserById(uclient, service.Uid(uidInt)); err != nil {
				c.Handle(500, "ucenter api request error", err)
				return
			} else {
				//都是登录状态了
				//首先查询下用户是否在数据库内
				queryUser:
					userQuery := m.GetUserByLoginQuery{LoginOrEmail: user.Name}
					err := bus.Dispatch(&userQuery)

					if err != nil {
						//如果用户不存在,则插入数据
						cmd := m.CreateUserCommand{}
						cmd.Login = user.Name
						cmd.Email = user.Email
						cmd.Password = setting.AdminPassword  //暂用管理员密码吧
						cmd.IsAdmin = false

						if err := bus.Dispatch(&cmd); err != nil {
							log.Error(3, "Failed to create user" + user.Name, err)
							return
						}

						log.Info("Created user: %v", user.Name)
						goto queryUser
					}

				userModel := userQuery.Result
				//记录状态
				loginUserWithUser(userModel, c)
				//跳转页面
				c.Redirect(setting.AppSubUrl + "/")
			}
		}
	}


}

func tryLoginUsingRememberCookie(c *middleware.Context) bool {
	// Check auto-login.
	uname := c.GetCookie(setting.CookieUserName)
	if len(uname) == 0 {
		return false
	}

	isSucceed := false
	defer func() {
		if !isSucceed {
			log.Trace("auto-login cookie cleared: %s", uname)
			c.SetCookie(setting.CookieUserName, "", -1, setting.AppSubUrl+"/")
			c.SetCookie(setting.CookieRememberName, "", -1, setting.AppSubUrl+"/")
			return
		}
	}()

	userQuery := m.GetUserByLoginQuery{LoginOrEmail: uname}
	if err := bus.Dispatch(&userQuery); err != nil {
		return false
	}

	user := userQuery.Result

	// validate remember me cookie
	if val, _ := c.GetSuperSecureCookie(
		util.EncodeMd5(user.Rands+user.Password), setting.CookieRememberName); val != user.Login {
		return false
	}

	isSucceed = true
	loginUserWithUser(user, c)
	return true
}

func LoginApiPing(c *middleware.Context) {
	if !tryLoginUsingRememberCookie(c) {
		c.JsonApiErr(401, "Unauthorized", nil)
		return
	}

	c.JsonOK("Logged in")
}

func LoginPost(c *middleware.Context, cmd dtos.LoginCommand) {
	userQuery := m.GetUserByLoginQuery{LoginOrEmail: cmd.User}
	err := bus.Dispatch(&userQuery)

	if err != nil {
		c.JsonApiErr(401, "Invalid username or password", err)
		return
	}

	user := userQuery.Result

	passwordHashed := util.EncodePassword(cmd.Password, user.Salt)
	if passwordHashed != user.Password {
		c.JsonApiErr(401, "Invalid username or password", err)
		return
	}

	loginUserWithUser(user, c)

	result := map[string]interface{}{
		"message": "Logged in",
	}

	if redirectTo, _ := url.QueryUnescape(c.GetCookie("redirect_to")); len(redirectTo) > 0 {
		result["redirectUrl"] = redirectTo
		c.SetCookie("redirect_to", "", -1, setting.AppSubUrl+"/")
	}

	metrics.M_Api_Login_Post.Inc(1)

	c.JSON(200, result)
}

func loginUserWithUser(user *m.User, c *middleware.Context) {
	if user == nil {
		log.Error(3, "User login with nil user")
	}

	days := 86400 * setting.LogInRememberDays
	c.SetCookie(setting.CookieUserName, user.Login, days, setting.AppSubUrl+"/")
	c.SetSuperSecureCookie(util.EncodeMd5(user.Rands+user.Password), setting.CookieRememberName, user.Login, days, setting.AppSubUrl+"/")

	c.Session.Set(middleware.SESS_KEY_USERID, user.Id)
}

func Logout(c *middleware.Context) {
	c.SetCookie(setting.CookieUserName, "", -1, setting.AppSubUrl+"/")
	c.SetCookie(setting.CookieRememberName, "", -1, setting.AppSubUrl+"/")
	c.Session.Destory(c)
	checkUcenterLogout(c)
	c.Redirect(setting.AppSubUrl + "/login")
}
