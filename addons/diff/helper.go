package diff

import (
	"bufio"
	"bytes"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/oligo/gvcode/gutter/providers"
)

// GitDiff is a helper that can be used to parse git diff output.
// Use the NewGitDiff function to build a new instance to make sure
// we are dealing with a real git repository.
type GitDiff struct {
	dir      string
	filename string
}

func NewGitDiff(filePath string) *GitDiff {
	if _, err := exec.LookPath("git"); err != nil {
		return nil
	}

	// Get the absolute path and directory
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		log.Printf("Failed to get absolute path: %v", err)
		return nil
	}
	dir := filepath.Dir(absPath)
	filename := filepath.Base(absPath)

	// Run git diff
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(output)) != "true" {
		return nil
	}

	return &GitDiff{
		dir:      dir,
		filename: filename,
	}

}

// ParseGitDiff runs git diff on the given file and parses the output into DiffHunks.
func (d *GitDiff) ParseDiff() []*providers.DiffHunk {
	if d == nil {
		return nil
	}

	// Run git diff
	cmd := exec.Command("git", "diff", "--no-color", "-U0", "--", d.filename)
	cmd.Dir = d.dir
	output, err := cmd.Output()
	if err != nil {
		// git diff returns exit code 1 if there are changes, which is not an error
		if exitErr, ok := err.(*exec.ExitError); ok {
			if len(exitErr.Stderr) > 0 {
				log.Printf("git diff stderr: %s", exitErr.Stderr)
			}
		}
	}

	if len(output) == 0 {
		return nil
	}

	return parseDiffOutput(output)
}

var (
	// Regex to match hunk headers like @@ -10,3 +10,5 @@
	hunkHeaderRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)
)

// finalizeHunkType determines the hunk type based on the actual content
func finalizeHunkType(hunk *providers.DiffHunk) {
	if hunk == nil {
		return
	}

	hasOldLines := len(hunk.OldLines) > 0
	hasNewLines := len(hunk.NewLines) > 0

	if !hasOldLines && hasNewLines {
		hunk.Type = providers.DiffAdded
	} else if hasOldLines && !hasNewLines {
		hunk.Type = providers.DiffDeleted
		// For deleted hunks, the line number is where the deletion occurred
		hunk.EndLine = hunk.StartLine
	} else if hasOldLines && hasNewLines {
		hunk.Type = providers.DiffModified
	}
}

// parseDiffOutput parses unified diff output into DiffHunks.
func parseDiffOutput(output []byte) []*providers.DiffHunk {
	var hunks []*providers.DiffHunk

	scanner := bufio.NewScanner(bytes.NewReader(output))
	var currentHunk *providers.DiffHunk
	var inHunk bool

	for scanner.Scan() {
		line := scanner.Text()

		// Check for hunk header
		if matches := hunkHeaderRe.FindStringSubmatch(line); matches != nil {
			// Save previous hunk if exists
			if currentHunk != nil {
				finalizeHunkType(currentHunk)
				hunks = append(hunks, currentHunk)
			}

			newStart, _ := strconv.Atoi(matches[3])
			newCount := 1
			if matches[4] != "" {
				newCount, _ = strconv.Atoi(matches[4])
			}

			// Convert to 0-based line numbers
			newStart--

			// Create hunk with temporary type - will be determined after parsing content
			currentHunk = &providers.DiffHunk{
				Type:      providers.DiffModified, // Temporary, will be updated
				StartLine: newStart,
				EndLine:   newStart + max(newCount-1, 0),
				OldLines:  make([]string, 0),
				NewLines:  make([]string, 0),
			}

			inHunk = true
			continue
		}

		// Skip diff headers
		if strings.HasPrefix(line, "diff ") ||
			strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "--- ") ||
			strings.HasPrefix(line, "+++ ") {
			continue
		}

		// Process hunk content
		if inHunk && currentHunk != nil {
			if strings.HasPrefix(line, "-") {
				currentHunk.OldLines = append(currentHunk.OldLines, strings.TrimPrefix(line, "-"))
			} else if strings.HasPrefix(line, "+") {
				currentHunk.NewLines = append(currentHunk.NewLines, strings.TrimPrefix(line, "+"))
			}
		}
	}

	// Don't forget the last hunk
	if currentHunk != nil {
		finalizeHunkType(currentHunk)
		hunks = append(hunks, currentHunk)
	}

	return hunks
}
