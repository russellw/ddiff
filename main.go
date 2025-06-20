package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Config struct {
	showColors    bool
	showContext   int
	recursive     bool
	showBinary    bool
	ignoreSpace   bool
	showStats     bool
}

func main() {
	config := Config{}
	
	flag.BoolVar(&config.showColors, "color", true, "Show colored output")
	flag.IntVar(&config.showContext, "context", 3, "Number of context lines")
	flag.BoolVar(&config.recursive, "recursive", false, "Compare directories recursively")
	flag.BoolVar(&config.showBinary, "binary", false, "Show binary file differences")
	flag.BoolVar(&config.ignoreSpace, "ignore-space", false, "Ignore whitespace changes")
	flag.BoolVar(&config.showStats, "stats", false, "Show diff statistics")
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <file1|dir1> <file2|dir2>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}
	
	flag.Parse()
	
	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(1)
	}
	
	path1, path2 := flag.Arg(0), flag.Arg(1)
	
	info1, err1 := os.Stat(path1)
	info2, err2 := os.Stat(path2)
	
	if err1 != nil {
		fmt.Fprintf(os.Stderr, "Error accessing %s: %v\n", path1, err1)
		os.Exit(1)
	}
	
	if err2 != nil {
		fmt.Fprintf(os.Stderr, "Error accessing %s: %v\n", path2, err2)
		os.Exit(1)
	}
	
	if info1.IsDir() && info2.IsDir() {
		err := compareDirs(path1, path2, config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error comparing directories: %v\n", err)
			os.Exit(1)
		}
	} else if !info1.IsDir() && !info2.IsDir() {
		err := compareFiles(path1, path2, config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error comparing files: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Cannot compare file with directory\n")
		os.Exit(1)
	}
}

func compareFiles(file1, file2 string, config Config) error {
	content1, err := readFileLines(file1)
	if err != nil {
		return fmt.Errorf("reading %s: %v", file1, err)
	}
	
	content2, err := readFileLines(file2)
	if err != nil {
		return fmt.Errorf("reading %s: %v", file2, err)
	}
	
	if isBinary(content1) || isBinary(content2) {
		if config.showBinary {
			fmt.Printf("Binary files %s and %s differ\n", file1, file2)
		}
		return nil
	}
	
	diff := generateUnifiedDiff(file1, file2, content1, content2, config)
	if len(diff) > 0 {
		fmt.Print(strings.Join(diff, "\n") + "\n")
	}
	
	return nil
}

func compareDirs(dir1, dir2 string, config Config) error {
	files1, err := getFileList(dir1, config.recursive)
	if err != nil {
		return fmt.Errorf("listing %s: %v", dir1, err)
	}
	
	files2, err := getFileList(dir2, config.recursive)
	if err != nil {
		return fmt.Errorf("listing %s: %v", dir2, err)
	}
	
	allFiles := make(map[string]bool)
	for _, f := range files1 {
		allFiles[f] = true
	}
	for _, f := range files2 {
		allFiles[f] = true
	}
	
	var sortedFiles []string
	for f := range allFiles {
		sortedFiles = append(sortedFiles, f)
	}
	sort.Strings(sortedFiles)
	
	for _, relPath := range sortedFiles {
		path1 := filepath.Join(dir1, relPath)
		path2 := filepath.Join(dir2, relPath)
		
		info1, err1 := os.Stat(path1)
		info2, err2 := os.Stat(path2)
		
		if err1 != nil && err2 != nil {
			continue
		} else if err1 != nil {
			printColor(config, "green", fmt.Sprintf("+++ %s\n", path2))
			continue
		} else if err2 != nil {
			printColor(config, "red", fmt.Sprintf("--- %s\n", path1))
			continue
		}
		
		if info1.IsDir() || info2.IsDir() {
			continue
		}
		
		if info1.Size() != info2.Size() || info1.ModTime() != info2.ModTime() {
			err := compareFiles(path1, path2, config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error comparing %s: %v\n", relPath, err)
			}
		}
	}
	
	return nil
}

func readFileLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	
	return lines, scanner.Err()
}

func isBinary(lines []string) bool {
	for _, line := range lines {
		for _, r := range line {
			if r == 0 {
				return true
			}
		}
	}
	return false
}

func getFileList(dir string, recursive bool) ([]string, error) {
	var files []string
	
	walkFunc := func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		
		if relPath == "." {
			return nil
		}
		
		if !recursive && strings.Contains(relPath, string(filepath.Separator)) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		if !info.IsDir() {
			files = append(files, relPath)
		}
		
		return nil
	}
	
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return walkFunc(path, info, err)
	})
	
	return files, err
}

func generateUnifiedDiff(file1, file2 string, lines1, lines2 []string, config Config) []string {
	var result []string
	
	result = append(result, fmt.Sprintf("--- %s", file1))
	result = append(result, fmt.Sprintf("+++ %s", file2))
	
	hunks := generateHunks(lines1, lines2, config.showContext)
	
	for _, hunk := range hunks {
		result = append(result, hunk...)
	}
	
	return result
}

func generateHunks(lines1, lines2 []string, context int) [][]string {
	var hunks [][]string
	
	i, j := 0, 0
	for {
		startI := i
		
		for i < len(lines1) && j < len(lines2) && lines1[i] == lines2[j] {
			i++
			j++
		}
		
		if i >= len(lines1) && j >= len(lines2) {
			break
		}
		
		deleteStart := i
		for i < len(lines1) && (j >= len(lines2) || lines1[i] != lines2[j]) {
			found := false
			for k := j; k < len(lines2); k++ {
				if lines1[i] == lines2[k] {
					found = true
					break
				}
			}
			if !found {
				i++
			} else {
				break
			}
		}
		deleteEnd := i
		
		insertStart := j
		for j < len(lines2) && (i >= len(lines1) || lines1[i] != lines2[j]) {
			found := false
			for k := i; k < len(lines1); k++ {
				if lines2[j] == lines1[k] {
					found = true
					break
				}
			}
			if !found {
				j++
			} else {
				break
			}
		}
		insertEnd := j
		
		if deleteStart < deleteEnd || insertStart < insertEnd {
			hunk := generateHunk(lines1, lines2, startI, deleteStart, deleteEnd, insertStart, insertEnd, context)
			if len(hunk) > 0 {
				hunks = append(hunks, hunk)
			}
		}
	}
	
	return hunks
}

func generateHunk(lines1, lines2 []string, contextStart, deleteStart, deleteEnd, insertStart, insertEnd, context int) []string {
	var hunk []string
	
	actualStart := max(0, deleteStart-context)
	actualEnd1 := min(len(lines1), deleteEnd+context)
	actualEnd2 := min(len(lines2), insertEnd+context)
	
	hunkStart1 := actualStart + 1
	hunkLen1 := actualEnd1 - actualStart
	hunkStart2 := insertStart - (deleteStart - actualStart) + 1
	hunkLen2 := actualEnd2 - (insertStart - (deleteStart - actualStart))
	
	hunk = append(hunk, fmt.Sprintf("@@ -%d,%d +%d,%d @@", hunkStart1, hunkLen1, hunkStart2, hunkLen2))
	
	for i := actualStart; i < deleteStart; i++ {
		hunk = append(hunk, " "+lines1[i])
	}
	
	for i := deleteStart; i < deleteEnd; i++ {
		hunk = append(hunk, "-"+lines1[i])
	}
	
	for i := insertStart; i < insertEnd; i++ {
		hunk = append(hunk, "+"+lines2[i])
	}
	
	return hunk
}


func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func printColor(config Config, color, text string) {
	if !config.showColors {
		fmt.Print(text)
		return
	}
	
	var colorCode string
	switch color {
	case "red":
		colorCode = "\033[31m"
	case "green":
		colorCode = "\033[32m"
	case "yellow":
		colorCode = "\033[33m"
	case "blue":
		colorCode = "\033[34m"
	default:
		colorCode = ""
	}
	
	if colorCode != "" {
		fmt.Printf("%s%s\033[0m", colorCode, text)
	} else {
		fmt.Print(text)
	}
}