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

func TestGenerateHunks(t *testing.T) {
	lines1 := []string{"line1", "line2", "line3", "line4", "line5"}
	lines2 := []string{"line1", "modified2", "line3", "line4", "line5"}
	
	hunks := generateHunks(lines1, lines2, 1)
	
	if len(hunks) == 0 {
		t.Error("Expected hunk output")
	}
	
	hunkStr := strings.Join(hunks[0], "\n")
	if !strings.Contains(hunkStr, "@@") {
		t.Error("Hunk should contain hunk header with @@")
	}
	if !strings.Contains(hunkStr, "-line2") {
		t.Error("Hunk should show deleted line")
	}
	if !strings.Contains(hunkStr, "+modified2") {
		t.Error("Hunk should show added line")
	}
}

func TestLargeFileComparison(t *testing.T) {
	config := Config{showContext: 3, showColors: false}
	
	// Test with larger files to check for performance issues
	err := compareFiles("testdata/large_file1.txt", "testdata/large_file2.txt", config)
	if err != nil {
		t.Fatalf("Failed to compare large files: %v", err)
	}
}

func TestCLILargeFileComparison(t *testing.T) {
	// Set a timeout to catch hanging behavior
	cmd := exec.Command("timeout", "30s", "./ddiff", "--color=false", "testdata/large_file1.txt", "testdata/large_file2.txt")
	output, err := cmd.CombinedOutput()
	
	// If timeout command doesn't exist, fall back to regular command
	if strings.Contains(string(output), "timeout: command not found") {
		cmd = exec.Command("./ddiff", "--color=false", "testdata/large_file1.txt", "testdata/large_file2.txt")
		output, err = cmd.CombinedOutput()
	}
	
	if err != nil {
		t.Logf("Output: %s", output)
		t.Fatalf("CLI large file comparison failed or timed out: %v", err)
	}
	
	outputStr := string(output)
	if len(outputStr) == 0 {
		t.Error("Expected some diff output for large files")
	}
	
	// Should contain diff headers
	if !strings.Contains(outputStr, "--- testdata/large_file1.txt") {
		t.Error("CLI output should contain source file header")
	}
	if !strings.Contains(outputStr, "+++ testdata/large_file2.txt") {
		t.Error("CLI output should contain target file header")
	}
}

func BenchmarkLargeFileComparison(b *testing.B) {
	config := Config{showContext: 3, showColors: false}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := compareFiles("testdata/large_file1.txt", "testdata/large_file2.txt", config)
		if err != nil {
			b.Fatalf("Failed to compare large files: %v", err)
		}
	}
}

func TestRecursiveDirectoryComparison(t *testing.T) {
	config := Config{showContext: 3, showColors: false, recursive: true}
	
	err := compareDirs("testdata/deep1", "testdata/deep2", config)
	if err != nil {
		t.Fatalf("Failed to compare recursive directories: %v", err)
	}
}

func TestCLIRecursiveDirectoryComparison(t *testing.T) {
	cmd := exec.Command("./ddiff", "--color=false", "--recursive", "testdata/deep1", "testdata/deep2")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI recursive directory comparison failed: %v\nOutput: %s", err, output)
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "src/models/user.go") {
		t.Error("CLI output should show differences in nested files")
	}
	if !strings.Contains(outputStr, "src/utils/helpers.go") {
		t.Error("CLI output should show files only in deep2")
	}
}

func TestNonRecursiveVsRecursive(t *testing.T) {
	// Test that non-recursive doesn't find nested files
	cmd := exec.Command("./ddiff", "--color=false", "testdata/deep1", "testdata/deep2")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI non-recursive comparison failed: %v\nOutput: %s", err, output)
	}
	
	outputStr := string(output)
	// Non-recursive should not show nested file differences
	if strings.Contains(outputStr, "src/models/user.go") {
		t.Error("Non-recursive mode should not show nested file differences")
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