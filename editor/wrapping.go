package editor

import (
	"iter"
	"log"

	"gioui.org/text"
	"github.com/go-text/typesetting/segmenter"
	"golang.org/x/image/math/fixed"
)

// breakOption represents a rune index in rune slice at which it is
// safe to break a line.
type breakOption int

type breaker struct {
	wordSegmenter     *segmenter.LineIterator
	graphemeSegmenter *segmenter.GraphemeIterator
	runes             int
	prevBreak         breakOption
	prevUnread        bool
}

// newBreaker returns a breaker initialized to break the text.
func newBreaker(seg *segmenter.Segmenter, text []rune) *breaker {
	seg.Init(text)
	br := &breaker{
		wordSegmenter:     seg.LineIterator(),
		graphemeSegmenter: seg.GraphemeIterator(),
		runes:             len(text),
	}
	return br
}

func (b *breaker) nextWordBreak() (breakOption, bool) {
	if b.prevUnread {
		b.prevUnread = false
		return b.prevBreak, true
	}

	var opt breakOption
	for b.wordSegmenter.Next() {
		line := b.wordSegmenter.Line()
		opt = breakOption(line.Offset + len(line.Text))
		if opt > b.prevBreak {
			b.prevBreak = opt
			return b.prevBreak, true
		}
	}

	return 0, false
}

func (b *breaker) nextGraphemeBreak() (breakOption, bool) {
	if b.prevUnread {
		b.prevUnread = false
		return b.prevBreak, true
	}

	var opt breakOption
	for b.graphemeSegmenter.Next() {
		grapheme := b.graphemeSegmenter.Grapheme()
		opt = breakOption(grapheme.Offset + len(grapheme.Text))
		if opt > b.prevBreak {
			b.prevBreak = opt
			return b.prevBreak, true
		}
	}

	return 0, false
}

func (b *breaker) markPrevUnread() {
	b.prevUnread = true
}

// glyphReader is a buffered glyph reader to read from the shaped glyphs.
type glyphReader struct {
	nextGlyph func() (text.Glyph, bool)
	buf       []text.Glyph
	// mark the buffer has overflow the maxWidth of the line.
	overflow bool
}

func (b *glyphReader) next() *text.Glyph {
	gl, ok := b.nextGlyph()
	if !ok {
		return nil
	}

	b.buf = append(b.buf, gl)
	return &gl
}

func (b *glyphReader) get() []text.Glyph {
	return b.buf
}

// advance calculates the advance of all glyphs in the buffer.
func (b *glyphReader) advance() fixed.Int26_6 {
	width := fixed.I(0)
	for _, g := range b.buf {
		width += g.Advance
	}

	return width
}

func (b *glyphReader) reset() {
	b.buf = b.buf[:0]
	b.overflow = false
}

// lineWrapper wraps a paragraph of text to lines using the greedy line breaking
// algorithm. Unlike the normal line breaking routine, it expands tab characters
// to the next tabstop before wrapping.
type lineWrapper struct {
	seg             segmenter.Segmenter
	breaker         *breaker
	maxWidth        int
	spaceGlyph      *text.Glyph
	tabStopInterval fixed.Int26_6

	runeOff     int
	currentLine line
	glyphBuf    glyphReader
}

func (w *lineWrapper) setup(nextGlyph func() (text.Glyph, bool), paragraph []rune, maxWidth int, tabStopInterval fixed.Int26_6, spaceGlyph *text.Glyph) {
	w.breaker = newBreaker(&w.seg, paragraph)
	w.maxWidth = maxWidth
	w.tabStopInterval = tabStopInterval
	w.spaceGlyph = spaceGlyph
	w.currentLine = line{}
	w.glyphBuf.nextGlyph = nextGlyph
	w.glyphBuf.reset()
	w.runeOff = 0
}

func (w *lineWrapper) WrapParagraph(glyphsIter iter.Seq[text.Glyph], paragraph []rune, maxWidth int, tabStopInterval fixed.Int26_6, spaceGlyph *text.Glyph) []*line {
	nextGlyph, stop := iter.Pull(glyphsIter)
	defer stop()
	w.setup(nextGlyph, paragraph, maxWidth, tabStopInterval, spaceGlyph)

	lines := make([]*line, 0)

	for {
		l := w.wrapNextLine(paragraph)
		if len(l.glyphs) == 0 {
			break
		}

		lines = append(lines, &l)
		w.currentLine = line{}
	}

	return lines
}

// wrapNextLine breaking lines by looking at the break opportunities defined in https://unicode.org/reports/tr14 first.
// If no break opportunities can be found, it'll try to break at the grapheme cluster bounderies.
func (w *lineWrapper) wrapNextLine(paragraph []rune) line {
	// Handle the remaining glyphs from the previous iteration.
	// The case that a single word exceeds the line width is already handled in
	// the previous iteration, so we are safe to add it to the current line here.
	if len(w.glyphBuf.buf) > 0 {
		if !w.glyphBuf.overflow {
			w.currentLine.append(w.glyphBuf.buf...)
			w.glyphBuf.reset()
		} else {

		}
	}

	var state wrapState

	for {
		// try to break at each word boundaries.
		breakAtIdx, ok := w.breaker.nextWordBreak()
		if !ok {
			state = wrapEnd
			break
		}

		wordOk := w.readToNextBreak(breakAtIdx, paragraph)
		if !wordOk {
			// A single word already exceeds the maxWidth. We have to break inside the word
			// to keep it fit the line width, otherwise it will overflow.
			state = wrapWordOverflow
			w.breaker.markPrevUnread()
			break
		}

		// check if the line will exceeds the maxWidth if we put the glyph in the current line.
		if w.currentLine.width+w.glyphBuf.advance() > fixed.I(w.maxWidth) {
			state = wrapLineOverflow
			break
		}

		w.currentLine.append(w.glyphBuf.get()...)
		w.glyphBuf.reset()
	}

	if len(w.currentLine.glyphs) > 0 {
		return w.currentLine
	}

	log.Println("state", state)

	for {
		// try to break at grapheme cluster boundaries.
		breakAtIdx, ok := w.breaker.nextGraphemeBreak()
		if !ok {
			break
		}

		w.glyphBuf.reset()
		done := w.readToNextBreak(breakAtIdx, paragraph)
		if done {
			break
		}

		// check if the line will exceeds the maxWidth if we put the glyph in the current line.
		if w.currentLine.width+w.glyphBuf.advance() > fixed.I(w.maxWidth) {
			break
		}

		w.currentLine.append(w.glyphBuf.buf...)

	}

	for {
		gl := w.glyphBuf.next()
		if gl == nil {
			break
		}

		isStart := gl.Flags&text.FlagParagraphStart != 0
		log.Println("is start of paragraph: ", isStart)

		w.currentLine.append(w.glyphBuf.buf...)
		log.Println("[1]current line*******: ", w.currentLine)

		w.glyphBuf.reset()
		log.Println("[2]current line*******: ", w.currentLine)

	}

	return w.currentLine
}

// readToNextBreak read glyphs from the iterator until it reached to break option.
// It returns a boolean value indicating whether it has terminated early.
func (w *lineWrapper) readToNextBreak(breakAtIdx breakOption, paragraph []rune) bool {
	for int(breakAtIdx) > w.runeOff {
		gl := w.glyphBuf.next()
		if gl == nil {
			break
		}

		if gl.Flags&text.FlagClusterBreak != 0 {
			w.runeOff += int(gl.Runes)
			isTab := paragraph[w.runeOff-1] == '\t'
			if isTab {
				// the rune is a tab, expand it before line wrapping.
				w.expandTabGlyph(gl)
			}
		}

		if w.glyphBuf.advance() > fixed.I(w.maxWidth) {
			// breakAtIdx may still larger than w.runeOff.
			w.glyphBuf.overflow = true
			return false
		}
	}

	return true
}

// expandTabGlyph expand the tab to the next tab stop.
func (w *lineWrapper) expandTabGlyph(gl *text.Glyph) {
	nextTabStop := (w.currentLine.width/w.tabStopInterval + 1) * w.tabStopInterval
	advance := nextTabStop - w.currentLine.width
	gl.Advance = advance
	gl.Offset = fixed.Point26_6{}
	gl.ID = w.spaceGlyph.ID
	gl.Ascent = w.spaceGlyph.Ascent
	gl.Descent = w.spaceGlyph.Descent
}

type wrapState uint8

const (
	wrapNoFit wrapState = iota
	wrapWordOverflow
	wrapLineOverflow
	wrapEnd
)
