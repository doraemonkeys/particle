package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/doraemonkeys/doraemon"
	"github.com/syncthing/syncthing/lib/fs" // For fs.Filesystem
	"github.com/syncthing/syncthing/lib/ignore"
)

const ParticleSeparatorLine = "// ---------------- AUTO GENRATE BY PARTICLE ----------------"

type stIgnoreEdit struct {
	baseLines            []string
	particleLines        []string
	stFileMd5Hex         []byte
	filePath             string
	particleLinesChanged bool
}

func NewstIgnoreEdit(filePath string) (*stIgnoreEdit, error) {
	baseLines := make([]string, 0)
	particleLines := make([]string, 0)
	foundParticleSeparatorCount := 0
	if doraemon.FileIsExist(filePath).IsFalse() {
		return &stIgnoreEdit{
			baseLines:     baseLines,
			particleLines: particleLines,
			stFileMd5Hex:  []byte(""),
			filePath:      filePath,
		}, nil
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	fileMd5, err := doraemon.ComputeMD5(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to compute file md5: %w", err)
	}
	lines, err := doraemon.ReadLines(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to read lines: %w", err)
	}
	for _, line := range lines {
		if !strings.Contains(string(line), ParticleSeparatorLine) {
			if foundParticleSeparatorCount != 1 {
				if len(line) > 0 {
					baseLines = append(baseLines, string(line))
				}
			} else {
				particleLines = append(particleLines, string(line))
			}
			continue
		}
		foundParticleSeparatorCount++
	}
	if foundParticleSeparatorCount != 0 && foundParticleSeparatorCount != 2 {
		return nil, fmt.Errorf("invalid file format, found %d separator lines", foundParticleSeparatorCount)
	}
	return &stIgnoreEdit{
		baseLines:     baseLines,
		particleLines: particleLines,
		stFileMd5Hex:  fileMd5,
		filePath:      filePath,
	}, nil
}

func (s *stIgnoreEdit) createEmptyFile() error {
	err := os.WriteFile(s.filePath, []byte(""), 0666)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	md5, err := doraemon.ComputeFileMd5(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to compute file md5: %w", err)
	}
	s.stFileMd5Hex = md5
	return nil
}

func (s *stIgnoreEdit) SetChange() (updated bool, err error) {
	if !s.NeedUpdate() {
		return false, nil
	}
	if len(s.baseLines) == 0 && len(s.particleLines) == 0 {
		if doraemon.FileOrDirIsExist(s.filePath) {
			_ = os.Remove(s.filePath)
			return true, nil
		}
		return false, nil
	}
	if doraemon.FileIsExist(s.filePath).IsFalse() {
		err := s.createEmptyFile()
		if err != nil {
			return false, err
		}
	}
	fileMd5, err := doraemon.ComputeFileMd5(s.filePath)
	if err != nil {
		return false, fmt.Errorf("failed to compute file md5: %w", err)
	}
	if !bytes.Equal(fileMd5, s.stFileMd5Hex) {
		return false, fmt.Errorf("file md5 mismatch, original .stignore file has been modified")
	}
	writer := bytes.NewBuffer(nil)
	for _, line := range s.baseLines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return false, fmt.Errorf("failed to write line: %w", err)
		}
	}
	err = writer.WriteByte('\n')
	if err != nil {
		return false, fmt.Errorf("failed to write newline: %w", err)
	}
	if len(s.particleLines) > 0 {
		_, err = writer.WriteString(ParticleSeparatorLine + "\n")
		if err != nil {
			return false, fmt.Errorf("failed to write separator line: %w", err)
		}
		for _, line := range s.particleLines {
			_, err := writer.WriteString(line + "\n")
			if err != nil {
				return false, fmt.Errorf("failed to write line: %w", err)
			}
		}
		err = writer.WriteByte('\n')
		if err != nil {
			return false, fmt.Errorf("failed to write newline: %w", err)
		}
		_, err = writer.WriteString(ParticleSeparatorLine + "\n")
		if err != nil {
			return false, fmt.Errorf("failed to write separator line: %w", err)
		}
	}

	writerBytes := writer.Bytes()

	newContentMd5, err := doraemon.ComputeMD5(bytes.NewReader(writerBytes))
	if err != nil {
		return false, fmt.Errorf("failed to compute new content md5: %w", err)
	}
	if bytes.Equal(newContentMd5, s.stFileMd5Hex) {
		// no need to update
		return false, nil
	}

	// Get original file permissions
	fileInfo, err := os.Stat(s.filePath)
	if err != nil {
		return false, fmt.Errorf("failed to stat file: %w", err)
	}
	originalPerm := fileInfo.Mode()
	// Change file permissions to writable
	err = os.Chmod(s.filePath, 0666)
	if err != nil {
		return false, fmt.Errorf("failed to change file permissions: %w", err)
	}
	err = os.WriteFile(s.filePath, writerBytes, 0666)
	if err != nil {
		if strings.Contains(err.Error(), "Access is denied") {
			_ = os.Remove(s.filePath)
			err = os.WriteFile(s.filePath, writerBytes, 0666)
		}
		if err != nil {
			return false, fmt.Errorf("failed to write file: %w", err)
		}
	}
	// Restore original file permissions
	err = os.Chmod(s.filePath, originalPerm)
	if err != nil {
		return true, fmt.Errorf("failed to restore file permissions: %w", err)
	}

	newFileMd5, err := doraemon.ComputeFileMd5(s.filePath)
	if err != nil {
		return true, fmt.Errorf("failed to compute file md5: %w", err)
	}
	s.stFileMd5Hex = newFileMd5
	s.particleLinesChanged = false
	return true, nil
}

func (s *stIgnoreEdit) AddIgnores(ignores []string) bool {
	var linesMap = make(map[string]bool, len(s.particleLines))
	for _, line := range s.particleLines {
		linesMap[line] = true
	}
	added := false
	for _, ignore := range ignores {
		if _, ok := linesMap[ignore]; !ok {
			s.particleLines = append(s.particleLines, ignore)
			added = true
		}
	}
	s.particleLinesChanged = added
	return added
}

func (s *stIgnoreEdit) OverwriteIgnores(ignores []string) {
	s.particleLines = make([]string, 0, len(ignores))
	s.AddIgnores(ignores)
}

func (s *stIgnoreEdit) NeedUpdate() bool {
	return s.particleLinesChanged
}
func (s *stIgnoreEdit) GetBaseIgnoreCheckFunc() func(path string) bool {
	baseIgnores := bytes.NewBuffer(nil)
	for _, line := range s.baseLines {
		baseIgnores.WriteString(line + "\n")
	}
	rootDir := filepath.Dir(s.filePath)
	myFS := fs.NewFilesystem(fs.FilesystemTypeBasic, rootDir)
	matcher := ignore.New(myFS)

	err := matcher.Parse(baseIgnores, ".stignore")
	if err != nil {
		panic(err)
	}
	return func(path string) bool {
		path = filepath.ToSlash(path)
		rootDir = filepath.ToSlash(rootDir)
		path = strings.TrimPrefix(path, rootDir)
		path = strings.Trim(path, "/")
		return matcher.Match(path).CanSkipDir()
	}
}
