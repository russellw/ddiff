package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestReadFileLines(t *testing.T) {
	testFile := "testdata/file1.txt"
	lines, err := readFileLines(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	
	expected := []string{"line 1", "line 2", "line 3", "line 4", "line 5"}
	if len(lines) != len(expected) {
		t.Fatalf("Expected %d lines, got %d", len(expected), len(lines))
	}
	
	for i, line := range lines {
		if line != expected[i] {
			t.Errorf("Line %d: expected %q, got %q", i, expected[i], line)
		}
	}
}

func TestIsBinary(t *testing.T) {
	textLines := []string{"hello", "world", "test"}
	if isBinary(textLines) {
		t.Error("Text lines incorrectly identified as binary")
	}
	
	binaryLines := []string{"hello", "wor\x00ld", "test"}
	if !isBinary(binaryLines) {
		t.Error("Binary lines not identified as binary")
	}
}

func TestGenerateUnifiedDiff(t *testing.T) {
	lines1 := []string{"line 1", "line 2", "line 3"}
	lines2 := []string{"line 1", "modified line 2", "line 3"}
	
	config := Config{showContext: 1}
	diff := generateUnifiedDiff("file1", "file2", lines1, lines2, config)
	
	if len(diff) == 0 {
		t.Error("Expected diff output, got empty result")
	}
	
	diffStr := strings.Join(diff, "\n")
	if !strings.Contains(diffStr, "--- file1") {
		t.Error("Diff should contain source file header")
	}
	if !strings.Contains(diffStr, "+++ file2") {
		t.Error("Diff should contain target file header")
	}
	if !strings.Contains(diffStr, "-line 2") {
		t.Error("Diff should show removed line")
	}
	if !strings.Contains(diffStr, "+modified line 2") {
		t.Error("Diff should show added line")
	}
}

func TestGetFileList(t *testing.T) {
	files, err := getFileList("testdata/dir1", false)
	if err != nil {
		t.Fatalf("Failed to get file list: %v", err)
	}
	
	expectedFiles := map[string]bool{
		"shared.txt":        true,
		"only_in_dir1.txt": true,
	}
	
	if len(files) != len(expectedFiles) {
		t.Fatalf("Expected %d files, got %d", len(expectedFiles), len(files))
	}
	
	for _, file := range files {
		if !expectedFiles[file] {
			t.Errorf("Unexpected file in list: %s", file)
		}
	}
}

func TestCompareFiles(t *testing.T) {
	config := Config{showContext: 3, showColors: false}
	
	err := compareFiles("testdata/file1.txt", "testdata/file2.txt", config)
	if err != nil {
		t.Fatalf("Failed to compare files: %v", err)
	}
}

func TestCLIFileComparison(t *testing.T) {
	cmd := exec.Command("./ddiff", "--color=false", "testdata/file1.txt", "testdata/file2.txt")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI command failed: %v\nOutput: %s", err, output)
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "--- testdata/file1.txt") {
		t.Error("CLI output should contain source file header")
	}
	if !strings.Contains(outputStr, "+++ testdata/file2.txt") {
		t.Error("CLI output should contain target file header")
	}
}

func TestCLIDirectoryComparison(t *testing.T) {
	cmd := exec.Command("./ddiff", "--color=false", "testdata/dir1", "testdata/dir2")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI directory comparison failed: %v\nOutput: %s", err, output)
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "shared.txt") {
		t.Error("CLI output should show differences in shared files")
	}
}

func TestCLIInvalidArguments(t *testing.T) {
	cmd := exec.Command("./ddiff")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected CLI to fail with no arguments")
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "Usage:") {
		t.Error("CLI should show usage message when arguments are invalid")
	}
}

func TestCLINonExistentFile(t *testing.T) {
	cmd := exec.Command("./ddiff", "nonexistent1.txt", "nonexistent2.txt")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected CLI to fail with non-existent files")
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "Error accessing") {
		t.Error("CLI should show error message for non-existent files")
	}
}

func TestMaxMin(t *testing.T) {
	if max(5, 3) != 5 {
		t.Error("max(5, 3) should return 5")
	}
	if max(2, 8) != 8 {
		t.Error("max(2, 8) should return 8")
	}
	if min(5, 3) != 3 {
		t.Error("min(5, 3) should return 3")
	}
	if min(2, 8) != 2 {
		t.Error("min(2, 8) should return 2")
	}
}

func TestGenerateHunk(t *testing.T) {
	lines1 := []string{"line1", "line2", "line3", "line4", "line5"}
	lines2 := []string{"line1", "modified2", "line3", "line4", "line5"}
	
	hunk := generateHunk(lines1, lines2, 0, 1, 2, 1, 2, 1)
	
	if len(hunk) == 0 {
		t.Error("Expected hunk output")
	}
	
	hunkStr := strings.Join(hunk, "\n")
	if !strings.Contains(hunkStr, "@@") {
		t.Error("Hunk should contain hunk header with @@")
	}
}

func init() {
	if _, err := os.Stat("ddiff"); os.IsNotExist(err) {
		cmd := exec.Command("go", "build", "-o", "ddiff")
		if err := cmd.Run(); err != nil {
			panic("Failed to build ddiff binary for testing: " + err.Error())
		}
	}
}