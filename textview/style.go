package textview

import (
	"github.com/oligo/gvcode/textstyle/decoration"
	"github.com/oligo/gvcode/textstyle/syntax"
)

func (e *TextView) AddDecorations(styles ...decoration.Decoration) error {
	if e.decorations == nil {
		panic("TextView is not properly initialized.")
	}

	return e.decorations.Insert(styles...)
}

func (e *TextView) ClearDecorations(source string) error {
	if e.decorations == nil {
		panic("TextView is not properly initialized.")
	}

	if source == "" {
		return e.decorations.RemoveAll()
	} else {
		return e.decorations.RemoveBySource(source)
	}
}

func (e *TextView) SetColorScheme(scheme *syntax.ColorScheme) {
	e.syntaxStyles = syntax.NewTextTokens(scheme)
}

func (e *TextView) SetSyntaxTokens(tokens ...syntax.Token) {
	if e.syntaxStyles == nil {
		panic("TextView is not properly initialized.")
	}
	e.syntaxStyles.Set(tokens...)
}

// UpdateSyntaxTokensOffset adjusts existing syntax token offsets after a text edit.
// Parameters mirror Editor.replace: start and end are the old replaced range (runes),
// newEnd is start + (number of runes inserted).
//
// This method is necessary when code highlighting occurs in an async way, during the
// short time window we need to keep the highlighting visually stable. When the async
// full highlighting completes, it replaces the shifted tokens with fully correct ones.
func (e *TextView) UpdateSyntaxTokensOffset(start, end, newEnd int) {
	if e.syntaxStyles == nil {
		return
	}
	e.syntaxStyles.AdjustOffsets(start, end, newEnd)
}
