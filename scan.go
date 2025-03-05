package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/doraemonkeys/doraemon"
	"github.com/sirupsen/logrus"
)

type dirScanner struct {
	ignoreRules      []StIgnoreCheckFunc
	ignoreRulesDir   func(dir string) bool
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

func (d *dirScanner) SetIgnoreRulesDir(ignoreRulesDir func(dir string) bool) {
	d.ignoreRulesDir = ignoreRulesDir
}

func (d *dirScanner) ScanToGenerateStIgnore(dir string, dirFetchFromWeb bool) (updated bool, err error) {
	doneChan := make(chan struct{})
	go d.logScanning(doneChan)

	localRootDir, err := d.prepareDirectory(dir, dirFetchFromWeb)
	if err != nil {
		return false, err
	}

	var stIgnoreFile = filepath.Join(localRootDir, ".stignore")
	// d.logger.Infof("scan to generate stignore: %s", stIgnoreFile)
	stIgnore, err := NewstIgnoreEdit(stIgnoreFile)
	if err != nil {
		return false, err
	}
	d.ignoreRulesDir = stIgnore.GetBaseIgnoreCheckFunc()
	scannedIgnores, err := d.scanDir(localRootDir, "")
	if err != nil {
		return false, err
	}
	close(doneChan)

	stIgnore.OverwriteIgnores(scannedIgnores)

	updated, err = stIgnore.SetChange()
	if err != nil {
		return false, err
	}
	if updated {
		d.logger.Infof("Successfully updated settings in %s", localRootDir)
	} else {
		d.logger.Infof("No updates required for %s", localRootDir)
	}
	fmt.Println()
	return updated, nil
}

// New helper functions
func (d *dirScanner) logScanning(doneChan chan struct{}) {
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
}

func (d *dirScanner) prepareDirectory(dir string, dirFetchFromWeb bool) (string, error) {
	dir = filepath.ToSlash(dir)
	if strings.HasPrefix(dir, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		dir = filepath.Join(homeDir, dir[2:])
	}
	var err error
	if dirFetchFromWeb && strings.HasPrefix(dir, "./") {
		dir, err = d.resolveSyncthingPath(dir)
		if err != nil {
			return "", err
		}
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}
	return absDir, nil
}

func (d *dirScanner) resolveSyncthingPath(dir string) (string, error) {
	if d.syncthingBinPath != "" {
		syncthingBinDir := filepath.Dir(d.syncthingBinPath)
		return filepath.Join(syncthingBinDir, dir[2:]), nil
	}
	cmdPath, err := exec.LookPath("syncthing")
	if err == nil {
		syncthingBinDir := filepath.Dir(cmdPath)
		return filepath.Join(syncthingBinDir, dir[2:]), nil
	}
	currDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	if doraemon.FileIsExist(filepath.Join(currDir, dir[2:], "syncthing.exe")).IsFalse() &&
		doraemon.FileIsExist(filepath.Join(currDir, dir[2:], "syncthing")).IsFalse() {
		return "", fmt.Errorf("can't find syncthing binary in %s, use `-syncthing` to specify or join the syncthing binary to $PATH", currDir)
	}
	return dir, nil
}

func (d *dirScanner) scanDir(dir string, parentsDir string) ([]string, error) {
	if d.ignoreRulesDir != nil && d.ignoreRulesDir(dir) {
		d.logger.Debugf("ignore dir: %s\n", dir)
		return nil, nil
	}

	d.scanningDir = dir
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var ignores []string
	var ignoreNames = make(map[string]bool)
	for _, v := range d.ignoreRules {
		for _, ignoreName := range v(dir, entries) {
			var ignorePath = parentsDir + "/" + ignoreName
			if !*removeD {
				ignorePath = "(?d)" + ignorePath
			}
			ignores = append(ignores, ignorePath) //+"/**"
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
