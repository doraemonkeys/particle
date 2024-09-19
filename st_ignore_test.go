package main

import (
	"os"
	"reflect"
	"testing"

	"github.com/doraemonkeys/doraemon"
)

// 测试用的临时文件路径
const testFilePath = "test_stignore.tmp"

// 清理测试文件
func cleanupTestFile() {
	os.Remove(testFilePath)
}

func TestNewstIgnoreEdit(t *testing.T) {
	// 测试文件不存在的情况
	t.Run("FileNotExist", func(t *testing.T) {
		cleanupTestFile()
		defer cleanupTestFile()

		sie, err := NewstIgnoreEdit(testFilePath)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(sie.baseLines) != 0 || len(sie.particleLines) != 0 {
			t.Errorf("Expected empty lines, got baseLines: %v, particleLines: %v", sie.baseLines, sie.particleLines)
		}
	})

	// 测试文件存在且格式正确的情况
	t.Run("FileExistWithCorrectFormat", func(t *testing.T) {
		cleanupTestFile()
		defer cleanupTestFile()

		content := "base1\nbase2\n" + ParticleSeparatorLine + "\nparticle1\nparticle2\n" + ParticleSeparatorLine
		err := os.WriteFile(testFilePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		sie, err := NewstIgnoreEdit(testFilePath)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !reflect.DeepEqual(sie.baseLines, []string{"base1", "base2"}) {
			t.Errorf("Unexpected baseLines: %v", sie.baseLines)
		}
		if !reflect.DeepEqual(sie.particleLines, []string{"particle1", "particle2"}) {
			t.Errorf("Unexpected particleLines: %v", sie.particleLines)
		}
	})

	// 测试文件格式错误的情况
	t.Run("FileWithIncorrectFormat", func(t *testing.T) {
		cleanupTestFile()
		defer cleanupTestFile()

		content := "base1\n" + ParticleSeparatorLine + "\nparticle1\n"
		err := os.WriteFile(testFilePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		_, err = NewstIgnoreEdit(testFilePath)
		if err == nil {
			t.Fatalf("Expected error for incorrect format, got nil")
		}
	})
}

func TestCreateEmptyFile(t *testing.T) {
	cleanupTestFile()
	defer cleanupTestFile()

	sie := &stIgnoreEdit{filePath: testFilePath}
	err := sie.createEmptyFile()
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	if !doraemon.FileIsExist(testFilePath).IsTrue() {
		t.Errorf("File was not created")
	}

	content, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}
	if len(content) != 0 {
		t.Errorf("Expected empty file, got content: %s", content)
	}
}

func TestWriteToFile(t *testing.T) {
	cleanupTestFile()
	defer cleanupTestFile()

	sie := &stIgnoreEdit{
		baseLines:     []string{"base1", "base2"},
		particleLines: []string{"particle1", "particle2"},
		filePath:      testFilePath,
	}
	sie.particleLinesChanged = true
	_, err := sie.SetChange()
	if err != nil {
		t.Fatalf("Failed to write to file: %v", err)
	}

	content, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	expectedContent := "base1\nbase2\n\n" + ParticleSeparatorLine + "\nparticle1\nparticle2\n\n" + ParticleSeparatorLine + "\n"
	if string(content) != expectedContent {
		t.Errorf("Unexpected file content. Got:\n%s\nExpected:\n%s", content, expectedContent)
	}
}

func TestAddIgnores(t *testing.T) {
	sie := &stIgnoreEdit{
		particleLines: []string{"existing1", "existing2"},
	}

	newIgnores := []string{"new1", "existing1", "new2"}
	sie.AddIgnores(newIgnores)

	expectedLines := []string{"existing1", "existing2", "new1", "new2"}
	if !reflect.DeepEqual(sie.particleLines, expectedLines) {
		t.Errorf("Unexpected particleLines after AddIgnores. Got %v, expected %v", sie.particleLines, expectedLines)
	}
}

func TestOverwriteIgnores(t *testing.T) {
	sie := &stIgnoreEdit{
		particleLines: []string{"old1", "old2"},
	}

	newIgnores := []string{"new1", "new2"}
	sie.OverwriteIgnores(newIgnores)

	if !reflect.DeepEqual(sie.particleLines, newIgnores) {
		t.Errorf("Unexpected particleLines after OverwriteIgnores. Got %v, expected %v", sie.particleLines, newIgnores)
	}
}

func TestWriteToFileWithModification(t *testing.T) {
	cleanupTestFile()
	defer cleanupTestFile()

	// 创建初始文件
	sie, err := NewstIgnoreEdit(testFilePath)
	if err != nil {
		t.Fatalf("Failed to create initial stIgnoreEdit: %v", err)
	}

	sie.baseLines = []string{"base1", "base2"}
	sie.particleLines = []string{"particle1", "particle2"}
	sie.particleLinesChanged = true
	_, err = sie.SetChange()
	if err != nil {
		t.Fatalf("Failed to write initial file: %v", err)
	}

	// 尝试修改文件内容
	err = os.WriteFile(testFilePath, []byte("modified content"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// 尝试再次写入，应该失败
	sie.particleLinesChanged = true
	_, err = sie.SetChange()
	if err == nil {
		t.Fatalf("Expected error when writing to modified file, got nil")
	}
}
