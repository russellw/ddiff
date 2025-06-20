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
		printDiff(diff, config)
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
	edits := computeDiff(lines1, lines2)
	return createHunksFromEdits(lines1, lines2, edits, context)
}

type Edit struct {
	Type   string // "equal", "delete", "insert"
	Start1 int
	End1   int
	Start2 int
	End2   int
}

func computeDiff(lines1, lines2 []string) []Edit {
	n, m := len(lines1), len(lines2)
	
	// Use simple O(n*m) algorithm for reasonable performance
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	
	// Fill DP table
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if lines1[i-1] == lines2[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}
	
	// Backtrack to find edits
	var edits []Edit
	i, j := n, m
	
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && lines1[i-1] == lines2[j-1] {
			// Equal
			start1, start2 := i-1, j-1
			for i > 0 && j > 0 && lines1[i-1] == lines2[j-1] {
				i--
				j--
			}
			edits = append([]Edit{{Type: "equal", Start1: i, End1: start1 + 1, Start2: j, End2: start2 + 1}}, edits...)
		} else if i > 0 && (j == 0 || dp[i-1][j] >= dp[i][j-1]) {
			// Delete
			start1 := i - 1
			for i > 0 && (j == 0 || dp[i-1][j] >= dp[i][j-1]) {
				i--
			}
			edits = append([]Edit{{Type: "delete", Start1: i, End1: start1 + 1, Start2: j, End2: j}}, edits...)
		} else {
			// Insert
			start2 := j - 1
			for j > 0 && (i == 0 || dp[i-1][j] < dp[i][j-1]) {
				j--
			}
			edits = append([]Edit{{Type: "insert", Start1: i, End1: i, Start2: j, End2: start2 + 1}}, edits...)
		}
	}
	
	return edits
}

func createHunksFromEdits(lines1, lines2 []string, edits []Edit, context int) [][]string {
	var hunks [][]string
	
	i := 0
	for i < len(edits) {
		// Skip equal sections until we find changes
		for i < len(edits) && edits[i].Type == "equal" {
			i++
		}
		
		if i >= len(edits) {
			break
		}
		
		// Found changes, create a hunk
		hunkStart := i
		
		// Include changes until we have enough context
		for i < len(edits) && (edits[i].Type != "equal" || (edits[i].End1-edits[i].Start1) < context*2) {
			i++
		}
		
		hunk := createSingleHunk(lines1, lines2, edits[hunkStart:i], context)
		if len(hunk) > 0 {
			hunks = append(hunks, hunk)
		}
	}
	
	return hunks
}

func createSingleHunk(lines1, lines2 []string, edits []Edit, context int) []string {
	if len(edits) == 0 {
		return nil
	}
	
	// Calculate hunk boundaries
	start1 := edits[0].Start1
	end1 := edits[len(edits)-1].End1
	start2 := edits[0].Start2
	end2 := edits[len(edits)-1].End2
	
	// Add context
	contextStart1 := max(0, start1-context)
	contextEnd1 := min(len(lines1), end1+context)
	contextStart2 := max(0, start2-context)
	contextEnd2 := min(len(lines2), end2+context)
	
	var hunk []string
	hunk = append(hunk, fmt.Sprintf("@@ -%d,%d +%d,%d @@", 
		contextStart1+1, contextEnd1-contextStart1,
		contextStart2+1, contextEnd2-contextStart2))
	
	// Add context before changes
	for i := contextStart1; i < start1; i++ {
		hunk = append(hunk, " "+lines1[i])
	}
	
	// Add the changes
	for _, edit := range edits {
		switch edit.Type {
		case "equal":
			for i := edit.Start1; i < edit.End1; i++ {
				hunk = append(hunk, " "+lines1[i])
			}
		case "delete":
			for i := edit.Start1; i < edit.End1; i++ {
				hunk = append(hunk, "-"+lines1[i])
			}
		case "insert":
			for i := edit.Start2; i < edit.End2; i++ {
				hunk = append(hunk, "+"+lines2[i])
			}
		}
	}
	
	// Add context after changes
	for i := end1; i < contextEnd1; i++ {
		hunk = append(hunk, " "+lines1[i])
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

func printDiff(diff []string, config Config) {
	for _, line := range diff {
		if len(line) == 0 {
			fmt.Println()
			continue
		}
		
		switch line[0] {
		case '-':
			if strings.HasPrefix(line, "---") {
				// File header
				printColor(config, "white", line+"\n")
			} else {
				// Deleted line
				printColor(config, "red", line+"\n")
			}
		case '+':
			if strings.HasPrefix(line, "+++") {
				// File header
				printColor(config, "white", line+"\n")
			} else {
				// Added line
				printColor(config, "green", line+"\n")
			}
		case '@':
			// Hunk header
			printColor(config, "cyan", line+"\n")
		case ' ':
			// Context line
			fmt.Println(line)
		default:
			// Other lines (shouldn't happen in normal diff)
			fmt.Println(line)
		}
	}
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
	case "cyan":
		colorCode = "\033[36m"
	case "white":
		colorCode = "\033[37m"
	default:
		colorCode = ""
	}
	
	if colorCode != "" {
		fmt.Printf("%s%s\033[0m", colorCode, text)
	} else {
		fmt.Print(text)
	}
}