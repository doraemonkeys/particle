package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

type dirScanner struct {
	ignoreRules []StIgnoreCheckFunc
	logger      *logrus.Logger
	scanningDir string
}

func NewDirScanner(ignoreRules []StIgnoreCheckFunc) *dirScanner {
	return &dirScanner{
		ignoreRules: ignoreRules,
		logger:      logger,
	}
}

func (d *dirScanner) ScanToGenerateStIgnore(dir string, conn *syncThingConn) error {
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
