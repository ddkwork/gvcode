package gvcode

import (
	"image"
	"io"
	"math"
	"strings"

	"gioui.org/gesture"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/transfer"
	"gioui.org/layout"
)

func (e *Editor) processEvents(gtx layout.Context) (ev EditorEvent, ok bool) {
	if len(e.pending) > 0 {
		out := e.pending[0]
		e.pending = e.pending[:copy(e.pending, e.pending[1:])]
		return out, true
	}
	selStart, selEnd := e.Selection()
	defer func() {
		afterSelStart, afterSelEnd := e.Selection()
		if selStart != afterSelStart || selEnd != afterSelEnd {
			if ok {
				e.pending = append(e.pending, SelectEvent{})
			} else {
				ev = SelectEvent{}
				ok = true
			}
		}
	}()

	ev, ok = e.processPointer(gtx)
	if ok {
		return ev, ok
	}
	ev, ok = e.processKey(gtx)
	if ok {
		return ev, ok
	}
	return nil, false
}

func (e *Editor) processPointer(gtx layout.Context) (EditorEvent, bool) {
	var scrollX, scrollY pointer.ScrollRange
	textDims := e.text.FullDimensions()
	visibleDims := e.text.Dimensions()

	scrollOffX := e.text.ScrollOff().X
	scrollX.Min = min(-scrollOffX, 0)
	scrollX.Max = max(0, textDims.Size.X-(scrollOffX+visibleDims.Size.X))

	scrollOffY := e.text.ScrollOff().Y
	scrollY.Min = -scrollOffY
	scrollY.Max = max(0, textDims.Size.Y-(scrollOffY+visibleDims.Size.Y))

	sbounds := e.text.ScrollBounds()
	var soff int
	var smin, smax int

	sdist := e.scroller.Update(gtx.Metric, gtx.Source, gtx.Now, gesture.Vertical, scrollX, scrollY)
	// Have to wait for the patch to be accepted by Gio dev team.
	// if e.scroller.Direction() == gesture.Horizontal {
	// 	e.text.ScrollRel(sdist, 0)
	// 	soff = e.text.ScrollOff().X
	// 	smin, smax = sbounds.Min.X, sbounds.Max.X
	// } else {
	e.text.ScrollRel(0, sdist)
	soff = e.text.ScrollOff().Y
	smin, smax = sbounds.Min.Y, sbounds.Max.Y
	//}

	for {
		evt, ok := e.clicker.Update(gtx.Source)
		if !ok {
			break
		}
		ev, ok := e.processPointerEvent(gtx, evt)
		if ok {
			return ev, ok
		}
	}
	for {
		evt, ok := e.dragger.Update(gtx.Metric, gtx.Source, gesture.Both)
		if !ok {
			break
		}
		ev, ok := e.processPointerEvent(gtx, evt)
		if ok {
			return ev, ok
		}
	}

	if (sdist > 0 && soff >= smax) || (sdist < 0 && soff <= smin) {
		e.scroller.Stop()
	}
	return nil, false
}

func (e *Editor) processPointerEvent(gtx layout.Context, ev event.Event) (EditorEvent, bool) {
	switch evt := ev.(type) {
	case gesture.ClickEvent:
		switch {
		case evt.Kind == gesture.KindPress && evt.Source == pointer.Mouse,
			evt.Kind == gesture.KindClick && evt.Source != pointer.Mouse:
			prevCaretPos, _ := e.text.Selection()
			e.blinkStart = gtx.Now
			e.text.MoveCoord(image.Point{
				X: int(math.Round(float64(evt.Position.X))),
				Y: int(math.Round(float64(evt.Position.Y))),
			})
			gtx.Execute(key.FocusCmd{Tag: e})
			if !e.readOnly {
				gtx.Execute(key.SoftKeyboardCmd{Show: true})
			}
			if e.scroller.State() != gesture.StateFlinging {
				e.scrollCaret = true
			}

			if evt.Modifiers == key.ModShift {
				start, end := e.text.Selection()
				// If they clicked closer to the end, then change the end to
				// where the caret used to be (effectively swapping start & end).
				if abs(end-start) < abs(start-prevCaretPos) {
					e.text.SetCaret(start, prevCaretPos)
				}
			} else {
				e.text.ClearSelection()
			}
			e.dragging = true

			// Process multi-clicks.
			switch {
			case evt.NumClicks == 2:
				e.text.MoveWords(-1, selectionClear)
				e.text.MoveWords(1, selectionExtend)
				e.dragging = false
			case evt.NumClicks >= 3:
				e.text.MoveLineStart(selectionClear)
				e.text.MoveLineEnd(selectionExtend)
				e.dragging = false
			}

			if e.completor != nil {
				e.completor.Cancel()
			}
		}
	case pointer.Event:
		release := false
		switch {
		case evt.Kind == pointer.Release && evt.Source == pointer.Mouse:
			release = true
			fallthrough
		case evt.Kind == pointer.Drag && evt.Source == pointer.Mouse:
			if e.dragging {
				e.blinkStart = gtx.Now
				e.text.MoveCoord(image.Point{
					X: int(math.Round(float64(evt.Position.X))),
					Y: int(math.Round(float64(evt.Position.Y))),
				})
				e.scrollCaret = true

				if release {
					e.dragging = false
				}
			}
		}
	}
	return nil, false
}

func (e *Editor) processKey(gtx layout.Context) (EditorEvent, bool) {
	if e.text.Changed() {
		return ChangeEvent{}, true
	}

	if evt := e.processEditEvents(gtx); evt != nil {
		return evt, true
	}

	if evt := e.processCommands(gtx); evt != nil {
		return evt, true
	}

	if e.text.Changed() {
		return ChangeEvent{}, true
	}

	return nil, false
}

func (e *Editor) processEditEvents(gtx layout.Context) EditorEvent {
	filters := []event.Filter{
		key.FocusFilter{Target: e},
		transfer.TargetFilter{Target: e, Type: "application/text"},
	}

	for {
		evt, ok := gtx.Event(filters...)
		if !ok {
			break
		}

		e.blinkStart = gtx.Now

		switch ke := evt.(type) {
		case key.FocusEvent:
			// Reset IME state.
			e.ime.imeState = imeState{}
			if ke.Focus && !e.readOnly {
				gtx.Execute(key.SoftKeyboardCmd{Show: true})
			}
		case key.SnippetEvent:
			e.updateSnippet(gtx, ke.Start, ke.End)
		case key.EditEvent:
			e.onTextInput(ke)
		case key.SelectionEvent:
			e.scrollCaret = true
			e.scroller.Stop()
			e.text.SetCaret(ke.Start, ke.End)

			// Complete a paste event, initiated by Shortcut-V in Editor.command().
		case transfer.DataEvent:
			if evt := e.onPasteEvent(ke); evt != nil {
				return evt
			}
		}
	}
	if e.text.Changed() {
		return ChangeEvent{}
	}

	return nil
}

// updateSnippet queues a key.SnippetCmd if the snippet content or position
// have changed. off and len are in runes.
func (e *Editor) updateSnippet(gtx layout.Context, start, end int) {
	if start > end {
		start, end = end, start
	}
	length := e.text.Len()
	if start > length {
		start = length
	}
	if end > length {
		end = length
	}
	e.ime.start = start
	e.ime.end = end
	startOff := e.text.ByteOffset(start)
	endOff := e.text.ByteOffset(end)
	n := endOff - startOff
	if n > int64(len(e.ime.scratch)) {
		e.ime.scratch = make([]byte, n)
	}
	scratch := e.ime.scratch[:n]
	read, _ := e.buffer.ReadAt(scratch, startOff)

	if read != len(scratch) {
		panic("e.rr.Read truncated data")
	}
	newSnip := key.Snippet{
		Range: key.Range{
			Start: e.ime.start,
			End:   e.ime.end,
		},
		Text: e.ime.snippet.Text,
	}
	if string(scratch) != newSnip.Text {
		newSnip.Text = string(scratch)
	}
	if newSnip == e.ime.snippet {
		return
	}
	e.ime.snippet = newSnip
	gtx.Execute(key.SnippetCmd{Tag: e, Snippet: newSnip})
}

func (e *Editor) onCopyCut(gtx layout.Context, k key.Event) EditorEvent {
	lineOp := false
	if e.text.SelectionLen() == 0 {
		lineOp = true
		e.scratch = e.text.SelectedLineText(e.scratch)
		if len(e.scratch) > 0 && e.scratch[len(e.scratch)-1] != '\n' {
			e.scratch = append(e.scratch, '\n')
		}
	} else {
		e.scratch = e.text.SelectedText(e.scratch)
	}

	if text := string(e.scratch); text != "" {
		gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(text))})
		if k.Name == "X" && !e.readOnly {
			if !lineOp {
				if e.Delete(1) != 0 {
					return ChangeEvent{}
				}
			} else {
				if e.DeleteLine() != 0 {
					return ChangeEvent{}
				}
			}
		}
	}

	return nil
}

// onTab handles tab key event. If there is no selection of lines, intert a tab character
// at position of the cursor, else indent or unindent the selected lines, depending on if
// the event contains the shift modifier.
func (e *Editor) onTab(k key.Event) EditorEvent {
	if e.readOnly {
		return nil
	}

	backward := k.Modifiers.Contain(key.ModShift)
	if (!backward && e.SelectionLen() == 0) || e.text.PartialLineSelected() {
		// expand soft tab.
		start, end := e.text.Selection()
		if e.Insert(e.text.ExpandTab(start, end, "\t")) != 0 {
			return ChangeEvent{}
		}
	}

	if e.indenter.IndentMultiLines(backward) > 0 {
		// Reset xoff.
		e.text.MoveCaret(0, 0)
		e.scrollCaret = true
		return ChangeEvent{}
	}

	return nil

}

func (e *Editor) autoCompleteBracketsAndQuotes(ke key.EditEvent) bool {
	if ke.Text == "" {
		return false
	}

	allPairs := mergeMaps(e.text.BracketPairs, e.text.QuotePairs)
	closing, ok := allPairs[[]rune(ke.Text)[0]]
	if !ok {
		return false
	}

	e.scrollCaret = true
	e.scroller.Stop()
	e.replace(ke.Range.Start, ke.Range.End, ke.Text+string(closing))
	e.text.MoveCaret(-1, -1)
	return true
}

func (e *Editor) onTextInput(ke key.EditEvent) {
	if e.readOnly {
		return
	}

	if e.autoCompleteBracketsAndQuotes(ke) {
		return
	}

	e.scrollCaret = true
	e.scroller.Stop()
	e.replace(ke.Range.Start, ke.Range.End, ke.Text)
	// Reset caret xoff.
	e.text.MoveCaret(0, 0)
	// start to auto-complete, if there is a configured Completion.
	e.updateCompletor(true)
}

func (e *Editor) updateCompletor(startNew bool) {
	if e.completor == nil {
		return
	}

	word, wordOff := e.text.ReadWord(true)
	prefix := []rune(word)[:wordOff]
	//log.Println("word, prefix and wordOff", word, string(prefix), wordOff)
	ctx := CompletionContext{
		Input: string(prefix),
	}
	ctx.Position.Line, ctx.Position.Column = e.text.CaretPos()
	// scroll off will change after we update the position, so we use doc view position instead
	// of viewport position.
	ctx.Position.Coords = e.text.CaretCoords().Round().Add(e.text.ScrollOff())

	start, end := e.text.Selection()
	ctx.Position.Start = start - len(prefix)
	ctx.Position.End = end
	ctx.New = startNew
	e.completor.OnText(ctx)
}

func (e *Editor) onPasteEvent(ke transfer.DataEvent) EditorEvent {
	if e.readOnly {
		return nil
	}

	e.scrollCaret = true
	e.scroller.Stop()
	content, err := io.ReadAll(ke.Open())
	if err != nil {
		return nil
	}

	text := string(content)
	if e.onPaste != nil {
		text = e.onPaste(text)
	}

	runes := 0
	if isSingleLine(text) {
		runes = e.InsertLine(text)
	} else {
		runes = e.Insert(text)
	}

	if runes != 0 {
		return ChangeEvent{}
	}

	return nil
}

func (e *Editor) onInsertLineBreak(ke key.Event) EditorEvent {
	if e.readOnly {
		return nil
	}

	if e.indenter.IndentOnBreak("\n") {
		return ChangeEvent{}
	}

	return nil
}
