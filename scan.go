package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/doraemonkeys/doraemon"
	"github.com/sirupsen/logrus"
)

type dirScanner struct {
	ignoreRules      []StIgnoreCheckFunc
	logger           *logrus.Logger
	scanningDir      string
	syncthingBinPath string
}

func NewDirScanner(ignoreRules []StIgnoreCheckFunc, syncthingBin string) *dirScanner {
	return &dirScanner{
		ignoreRules:      ignoreRules,
		logger:           logger,
		syncthingBinPath: syncthingBin,
	}
}

func (d *dirScanner) ScanToGenerateStIgnore(dir string, isSyncthingRelativeDir bool, conn *syncThingConn) error {
	doneChan := make(chan struct{})
	defer close(doneChan)
	go func() {
		ticker := time.NewTicker(time.Millisecond * 500)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				d.logger.Info("scanning: ", d.scanningDir)
			case <-doneChan:
				return
			}
		}
	}()

	dir = filepath.ToSlash(dir)
	if strings.HasPrefix(dir, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		dir = filepath.Join(homeDir, dir[2:])
	}
	if isSyncthingRelativeDir && strings.HasPrefix(dir, "./") {
		if d.syncthingBinPath != "" {
			syncthingBinDir := filepath.Dir(d.syncthingBinPath)
			dir = filepath.Join(syncthingBinDir, dir[2:])
		} else {
			currDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
			if doraemon.FileIsExist(filepath.Join(currDir, dir[2:], "syncthing.exe")).IsFalse() &&
				doraemon.FileIsExist(filepath.Join(currDir, dir[2:], "syncthing")).IsFalse() {
				return fmt.Errorf("can't find syncthing binary in %s, use `-syncthing` to specify", currDir)
			}
		}
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	scannedIgnores, err := d.scanDir(dir, "")
	if err != nil {
		return err
	}
	var stIgnoreFile = filepath.Join(dir, ".stignore")
	// d.logger.Infof("scan to generate stignore: %s", stIgnoreFile)
	stIgnore, err := NewstIgnoreEdit(stIgnoreFile)
	if err != nil {
		return err
	}
	stIgnore.OverwriteIgnores(scannedIgnores)

	updated, err := stIgnore.SetChange()
	if err != nil {
		return err
	}
	if updated {
		d.logger.Infof("set ok in %s", dir)
	} else {
		d.logger.Infof("don't need to set in %s", dir)
	}
	if updated && conn != nil {
		err = conn.RestartSyncThing()
		if err != nil {
			d.logger.Warnf("restart sync thing error: %v", err)
		}
	}
	return nil
}

func (d *dirScanner) scanDir(dir string, parentsDir string) ([]string, error) {
	d.scanningDir = dir
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var ignores []string
	var ignoreNames = make(map[string]bool)
	for _, v := range d.ignoreRules {
		for _, ignoreName := range v(entries) {
			ignores = append(ignores, parentsDir+"/"+ignoreName) //+"/**"
			ignoreNames[ignoreName] = true
		}
	}

	// scan child dir
	for _, v := range entries {
		if v.IsDir() && !ignoreNames[v.Name()] {
			childIgnores, err := d.scanDir(filepath.Join(dir, v.Name()), parentsDir+"/"+v.Name())
			if err != nil {
				d.logger.Warnf("skip dir: %s, because: %s", dir, err.Error())
				continue
			}
			ignores = append(ignores, childIgnores...)
		}
	}

	return ignores, nil
}
