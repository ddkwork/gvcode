// Package folding implements code folding functionality for the editor.
// It provides code structure analysis and manages fold regions.
package folding

import (
	"regexp"
	"sort"
	"strings"
	"sync"
)

// FoldType represents the type of foldable region.
type FoldType int

const (
	// FoldTypeFunction represents a function/method fold.
	FoldTypeFunction FoldType = iota
	// FoldTypeType represents a type definition fold.
	FoldTypeType
	// FoldTypeComment represents a multi-line comment fold.
	FoldTypeComment
	// FoldTypeImport represents an import block fold.
	FoldTypeImport
	// FoldTypeConst represents a const block fold.
	FoldTypeConst
	// FoldTypeVar represents a var block fold.
	FoldTypeVar
	// FoldTypeRegion represents a user-defined region fold.
	FoldTypeRegion
)

// String returns the string representation of the fold type.
func (t FoldType) String() string {
	switch t {
	case FoldTypeFunction:
		return "function"
	case FoldTypeType:
		return "type"
	case FoldTypeComment:
		return "comment"
	case FoldTypeImport:
		return "import"
	case FoldTypeConst:
		return "const"
	case FoldTypeVar:
		return "var"
	case FoldTypeRegion:
		return "region"
	default:
		return "unknown"
	}
}

// FoldRange represents a foldable region in the code.
type FoldRange struct {
	// StartLine is the 0-based starting line number.
	StartLine int
	// EndLine is the 0-based ending line number (inclusive).
	EndLine int
	// Type is the type of fold.
	Type FoldType
	// Name is a descriptive name for the fold (e.g., function name).
	Name string
	// Collapsed indicates whether the fold is currently collapsed.
	Collapsed bool
	// Level is the nesting level of the fold.
	Level int
}

// Manager manages code folding regions and their states.
type Manager struct {
	mu sync.RWMutex

	// foldRanges contains all detected fold ranges.
	foldRanges []FoldRange

	// collapsedLines tracks which lines are hidden due to folding.
	// A line is considered collapsed if it's within a collapsed fold range.
	collapsedLines map[int]bool

	// lineCache caches the last analyzed lines.
	lineCache []string

	// foldMarkers caches the positions of fold markers in the text.
	foldMarkers []FoldMarker
}

// FoldMarker represents a fold marker (opening or closing brace).
type FoldMarker struct {
	Line  int
	Type  MarkerType
	Level int
}

// MarkerType represents the type of fold marker.
type MarkerType int

const (
	// MarkerOpen represents an opening brace or start of fold.
	MarkerOpen MarkerType = iota
	// MarkerClose represents a closing brace or end of fold.
	MarkerClose
)

// NewManager creates a new folding manager.
func NewManager() *Manager {
	return &Manager{
		foldRanges:     make([]FoldRange, 0),
		collapsedLines: make(map[int]bool),
	}
}

// AnalyzeLines analyzes the given lines and detects foldable regions.
// This should be called whenever the document content changes.
func (m *Manager) AnalyzeLines(lines []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if lines have changed
	if m.linesEqual(m.lineCache, lines) {
		return
	}

	m.lineCache = make([]string, len(lines))
	copy(m.lineCache, lines)

	// Clear previous analysis
	m.foldRanges = m.foldRanges[:0]
	m.foldMarkers = m.foldMarkers[:0]

	// Analyze the code structure
	m.detectFolds(lines)

	// Rebuild collapsed lines map
	m.rebuildCollapsedLines()
}

// linesEqual checks if two line slices are equal.
func (m *Manager) linesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// detectFolds detects all foldable regions in the code.
func (m *Manager) detectFolds(lines []string) {
	// Track brace depth and fold stack
	braceDepth := 0
	type foldStackEntry struct {
		line       int
		foldType   FoldType
		name       string
		braceLevel int
	}
	foldStack := make([]foldStackEntry, 0)

	// Track multi-line comment state
	inMultiLineComment := false
	commentStartLine := -1

	// Track import/const/var block state
	inBlock := false
	blockStartLine := -1
	blockType := FoldTypeConst

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Handle multi-line comments
		if strings.HasPrefix(trimmed, "/*") && !inMultiLineComment {
			inMultiLineComment = true
			commentStartLine = i
		}

		if inMultiLineComment {
			if strings.Contains(trimmed, "*/") {
				// End of multi-line comment
				if i > commentStartLine {
					m.foldRanges = append(m.foldRanges, FoldRange{
						StartLine: commentStartLine,
						EndLine:   i,
						Type:      FoldTypeComment,
						Name:      "comment",
						Level:     0,
					})
				}
				inMultiLineComment = false
			}
			continue
		}

		// Skip single-line comments and empty lines for fold detection
		if strings.HasPrefix(trimmed, "//") || trimmed == "" {
			continue
		}

		// Detect block starts (import, const, var)
		if !inBlock {
			if strings.HasPrefix(trimmed, "import (") {
				inBlock = true
				blockStartLine = i
				blockType = FoldTypeImport
			} else if strings.HasPrefix(trimmed, "const (") {
				inBlock = true
				blockStartLine = i
				blockType = FoldTypeConst
			} else if strings.HasPrefix(trimmed, "var (") {
				inBlock = true
				blockStartLine = i
				blockType = FoldTypeVar
			}

			if inBlock {
				continue
			}
		}

		// Detect block end
		if inBlock && trimmed == ")" {
			if i > blockStartLine {
				m.foldRanges = append(m.foldRanges, FoldRange{
					StartLine: blockStartLine,
					EndLine:   i,
					Type:      blockType,
					Name:      blockType.String(),
					Level:     0,
				})
			}
			inBlock = false
			continue
		}

		if inBlock {
			continue
		}

		// Count braces to track nesting
		openCount := strings.Count(line, "{")
		closeCount := strings.Count(line, "}")

		// Detect function/method/type starts
		if openCount > 0 && braceDepth == 0 {
			foldType, name := m.detectFoldType(line)
			if foldType != -1 {
				foldStack = append(foldStack, foldStackEntry{
					line:       i,
					foldType:   foldType,
					name:       name,
					braceLevel: braceDepth,
				})
			}
		}

		// Update brace depth
		braceDepth += openCount - closeCount

		// Check for fold ends
		if closeCount > 0 && len(foldStack) > 0 {
			// Pop folds that end at this brace level
			for len(foldStack) > 0 {
				entry := foldStack[len(foldStack)-1]
				if braceDepth <= entry.braceLevel {
					break
				}

				// End the fold
				if i > entry.line {
					m.foldRanges = append(m.foldRanges, FoldRange{
						StartLine: entry.line,
						EndLine:   i,
						Type:      entry.foldType,
						Name:      entry.name,
						Level:     entry.braceLevel,
					})
				}
				foldStack = foldStack[:len(foldStack)-1]
			}
		}
	}

	// Handle any remaining open folds (malformed code)
	for len(foldStack) > 0 {
		entry := foldStack[len(foldStack)-1]
		if len(lines)-1 > entry.line {
			m.foldRanges = append(m.foldRanges, FoldRange{
				StartLine: entry.line,
				EndLine:   len(lines) - 1,
				Type:      entry.foldType,
				Name:      entry.name,
				Level:     entry.braceLevel,
			})
		}
		foldStack = foldStack[:len(foldStack)-1]
	}

	// Sort fold ranges by start line
	sort.Slice(m.foldRanges, func(i, j int) bool {
		return m.foldRanges[i].StartLine < m.foldRanges[j].StartLine
	})
}

// detectFoldType detects the type of fold from a line of code.
func (m *Manager) detectFoldType(line string) (FoldType, string) {
	trimmed := strings.TrimSpace(line)

	// Function pattern: func Name(...) or func (recv) Name(...)
	funcPattern := regexp.MustCompile(`^func\s+(?:\([^)]+\)\s+)?(\w+)`)
	if matches := funcPattern.FindStringSubmatch(trimmed); matches != nil {
		return FoldTypeFunction, matches[1]
	}

	// Type pattern: type Name struct/interface/...
	typePattern := regexp.MustCompile(`^type\s+(\w+)`)
	if matches := typePattern.FindStringSubmatch(trimmed); matches != nil {
		return FoldTypeType, matches[1]
	}

	// Region pattern: //region Name or // region Name
	regionPattern := regexp.MustCompile(`^//\s*region\s+(\w+)`)
	if matches := regionPattern.FindStringSubmatch(trimmed); matches != nil {
		return FoldTypeRegion, matches[1]
	}

	return -1, ""
}

// rebuildCollapsedLines rebuilds the map of collapsed lines.
func (m *Manager) rebuildCollapsedLines() {
	m.collapsedLines = make(map[int]bool)
	for _, fold := range m.foldRanges {
		if fold.Collapsed {
			for i := fold.StartLine + 1; i <= fold.EndLine; i++ {
				m.collapsedLines[i] = true
			}
		}
	}
}

// GetFoldRanges returns all fold ranges.
func (m *Manager) GetFoldRanges() []FoldRange {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]FoldRange, len(m.foldRanges))
	copy(result, m.foldRanges)
	return result
}

// GetFoldAtLine returns the fold range at the given line (if any).
func (m *Manager) GetFoldAtLine(line int) *FoldRange {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for i := range m.foldRanges {
		if m.foldRanges[i].StartLine == line {
			return &m.foldRanges[i]
		}
	}
	return nil
}

// GetDeepestFoldAtLine returns the deepest fold range containing the given line.
func (m *Manager) GetDeepestFoldAtLine(line int) *FoldRange {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var deepest *FoldRange
	maxLevel := -1

	for i := range m.foldRanges {
		fold := &m.foldRanges[i]
		if line >= fold.StartLine && line <= fold.EndLine {
			if fold.Level > maxLevel {
				maxLevel = fold.Level
				deepest = fold
			}
		}
	}

	return deepest
}

// ToggleFold toggles the collapsed state of the fold at the given line.
func (m *Manager) ToggleFold(startLine int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.foldRanges {
		if m.foldRanges[i].StartLine == startLine {
			m.foldRanges[i].Collapsed = !m.foldRanges[i].Collapsed
			m.rebuildCollapsedLines()
			return m.foldRanges[i].Collapsed
		}
	}
	return false
}

// CollapseFold collapses the fold at the given line.
func (m *Manager) CollapseFold(startLine int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.foldRanges {
		if m.foldRanges[i].StartLine == startLine {
			changed := !m.foldRanges[i].Collapsed
			m.foldRanges[i].Collapsed = true
			if changed {
				m.rebuildCollapsedLines()
			}
			return changed
		}
	}
	return false
}

// ExpandFold expands the fold at the given line.
func (m *Manager) ExpandFold(startLine int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.foldRanges {
		if m.foldRanges[i].StartLine == startLine {
			changed := m.foldRanges[i].Collapsed
			m.foldRanges[i].Collapsed = false
			if changed {
				m.rebuildCollapsedLines()
			}
			return changed
		}
	}
	return false
}

// CollapseAll collapses all foldable regions.
func (m *Manager) CollapseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	changed := false
	for i := range m.foldRanges {
		if !m.foldRanges[i].Collapsed {
			m.foldRanges[i].Collapsed = true
			changed = true
		}
	}
	if changed {
		m.rebuildCollapsedLines()
	}
}

// ExpandAll expands all foldable regions.
func (m *Manager) ExpandAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	changed := false
	for i := range m.foldRanges {
		if m.foldRanges[i].Collapsed {
			m.foldRanges[i].Collapsed = false
			changed = true
		}
	}
	if changed {
		m.rebuildCollapsedLines()
	}
}

// IsLineVisible returns true if the given line is visible (not collapsed).
func (m *Manager) IsLineVisible(line int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return !m.collapsedLines[line]
}

// GetVisibleLineCount returns the number of visible lines in the given range.
func (m *Manager) GetVisibleLineCount(start, end int) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for i := start; i <= end; i++ {
		if !m.collapsedLines[i] {
			count++
		}
	}
	return count
}

// MapLineToVisible maps an actual line number to its visible position.
// Returns -1 if the line is collapsed.
func (m *Manager) MapLineToVisible(line int) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.collapsedLines[line] {
		return -1
	}

	visiblePos := 0
	for i := range line {
		if !m.collapsedLines[i] {
			visiblePos++
		}
	}
	return visiblePos
}

// MapVisibleToLine maps a visible position to an actual line number.
func (m *Manager) MapVisibleToLine(visiblePos int) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	visibleCount := 0
	for i := 0; i < len(m.lineCache); i++ {
		if !m.collapsedLines[i] {
			if visibleCount == visiblePos {
				return i
			}
			visibleCount++
		}
	}
	return -1
}
