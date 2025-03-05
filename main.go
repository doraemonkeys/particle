package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/doraemonkeys/mylog"
	"github.com/sirupsen/logrus"
)

var (
	targetDir    = flag.String("dir", "", "target directory")
	web          = flag.Bool("web", false, "get all dir from syncthing web api")
	host         = flag.String("host", "http://127.0.0.1:8384", "syncthing host")
	user         = flag.String("user", "", "syncthing user")
	pwdFile      = flag.String("pwdFile", "", "syncthing password file")
	syncthing    = flag.String("syncthing", "", "syncthing executable file")
	sleepSeconds = flag.Int("sleep", 0, "sleep seconds after scan")
	// remove ignore with '(?d)' prefix
	removeD  = flag.Bool("removeD", false, "remove ignore with '(?d)' prefix")
	logLevel = flag.String("logLevel", "info", "log level")
)

var logger *logrus.Logger

func init() {
	flag.Parse()
	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}
	l, err := mylog.NewLogger(mylog.LogConfig{
		LogFileDisable: true,
		LogLevel:       *logLevel,
		// DateSplit:      true,
	})
	if err != nil {
		panic(err)
	}
	logger = l
}

func parseFlags() ([]string, *syncThingConn, error) {

	if *web {
		conn, err := NewSyncThingConn(*user, *host)
		if err != nil {
			return nil, nil, err
		}
		pwd, err := conn.ReadPassword(*pwdFile)
		if err != nil {
			return nil, nil, err
		}
		err = conn.Connect(pwd)
		if err != nil {
			return nil, nil, err
		}
		dirs, err := conn.FetchDirectories()
		if err != nil {
			return nil, nil, err
		}
		return dirs, conn, nil
	}
	return []string{*targetDir}, nil, nil
}

func main() {
	dirs, conn, err := parseFlags()
	if err != nil {
		logger.Fatalf("parse flags error: %v", err)
	}
	for _, dir := range dirs {
		logger.Infof("ready to scan: %s", dir)
	}
	logger.Info("start scanning...")
	scanner := NewDirScanner(StIgnoreCheckList, *syncthing)
	for _, dir := range dirs {
		logger.Infof("scan dir: %s", dir)
		err = scanner.ScanToGenerateStIgnore(dir, *web, conn)
		if err != nil {
			logger.Fatalf("scan dir: %s error: %v", dir, err)
		}
	}
	fmt.Println()
	logger.Info("done")
	if *sleepSeconds > 0 {
		time.Sleep(time.Duration(*sleepSeconds) * time.Second)
	}
}
