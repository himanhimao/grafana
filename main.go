package main

import (
	"flag"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/himanhimao/grafana/pkg/cmd"
	"github.com/himanhimao/grafana/pkg/log"
	"github.com/himanhimao/grafana/pkg/metrics"
	"github.com/himanhimao/grafana/pkg/plugins"
	"github.com/himanhimao/grafana/pkg/services/eventpublisher"
	"github.com/himanhimao/grafana/pkg/services/notifications"
	"github.com/himanhimao/grafana/pkg/services/search"
	"github.com/himanhimao/grafana/pkg/services/sqlstore"
	"github.com/himanhimao/grafana/pkg/setting"
	"github.com/himanhimao/grafana/pkg/social"
)

var version = "master"
var commit = "NA"
var buildstamp string

var configFile = flag.String("config", "", "path to config file")
var homePath = flag.String("homepath", "", "path to grafana install/home path, defaults to working directory")
var pidFile = flag.String("pidfile", "", "path to pid file")

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	buildstampInt64, _ := strconv.ParseInt(buildstamp, 10, 64)

	setting.BuildVersion = version
	setting.BuildCommit = commit
	setting.BuildStamp = buildstampInt64

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		os.Exit(0)
	}()

	flag.Parse()

	writePIDFile()
	initRuntime()

	search.Init()
	social.NewOAuthService()
	eventpublisher.Init()
	plugins.Init()

	if err := notifications.Init(); err != nil {
		log.Fatal(3, "Notification service failed to initialize", err)
	}

	if setting.ReportingEnabled {
		go metrics.StartUsageReportLoop()
	}

	cmd.StartServer()

	log.Close()
}

func initRuntime() {
	setting.NewConfigContext(&setting.CommandLineArgs{
		Config:   *configFile,
		HomePath: *homePath,
		Args:     flag.Args(),
	})

	log.Info("Starting Grafana")
	log.Info("Version: %v, Commit: %v, Build date: %v", setting.BuildVersion, setting.BuildCommit, time.Unix(setting.BuildStamp, 0))
	setting.LogConfigurationInfo()

	sqlstore.NewEngine()
	sqlstore.EnsureAdminUser()
}

func writePIDFile() {
	if *pidFile == "" {
		return
	}

	// Ensure the required directory structure exists.
	err := os.MkdirAll(filepath.Dir(*pidFile), 0700)
	if err != nil {
		log.Fatal(3, "Failed to verify pid directory", err)
	}

	// Retrieve the PID and write it.
	pid := strconv.Itoa(os.Getpid())
	if err := ioutil.WriteFile(*pidFile, []byte(pid), 0644); err != nil {
		log.Fatal(3, "Failed to write pidfile", err)
	}
}
