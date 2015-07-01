package setting

type UcenterSettings struct {
	Enabled    bool
	Api_Url    string
	Api_Key   string
	Api_Secret string
}

func readUcenterSettings() {
	sec := Cfg.Section("cas.ucenter")
	Ucenter.Enabled = sec.Key("enabled").MustBool(false)
	Ucenter.Api_Url = sec.Key("api_url").String()
	Ucenter.Api_Key  = sec.Key("api_key").String()
	Ucenter.Api_Secret = sec.Key("api_secret").String()

}
