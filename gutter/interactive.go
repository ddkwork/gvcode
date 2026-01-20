package gutter

import (
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
)

// InteractiveGutter extends GutterProvider with interaction capabilities.
// Providers implementing this interface can respond to mouse clicks and hovers.
type InteractiveGutter interface {
	GutterProvider

	// HandleClick is called when the user clicks on this provider's area.
	// It receives the line number that was clicked, the pointer source (mouse/touch),
	// the number of clicks, and active key modifiers.
	// Returns true if the event was handled.
	HandleClick(line int, source pointer.Source, numClicks int, modifiers key.Modifiers) bool

	// HandleHover is called when the user hovers over this provider's area.
	// It receives the line number being hovered and returns hover information,
	// or nil if no hover effect should be shown.
	HandleHover(line int) *HoverInfo
}

// HoverInfo contains information about a hover effect to display.
type HoverInfo struct {
	// Text is a simple text description to show in a tooltip.
	Text string

	// Widget is an optional custom widget to render for the hover effect.
	// If provided, it takes precedence over Text.
	Widget layout.Widget
}

// GutterEvent is the base interface for all gutter-related events.
type GutterEvent interface {
	isGutterEvent()
}

// GutterClickEvent is emitted when a user clicks on a gutter area.
type GutterClickEvent struct {
	// ProviderID is the ID of the provider that was clicked.
	ProviderID string

	// Line is the 0-based line number that was clicked.
	Line int

	// Source indicates the pointer source (mouse, touch, etc.).
	Source pointer.Source

	// NumClicks is the number of successive clicks.
	NumClicks int

	// Modifiers indicates which modifier keys were held.
	Modifiers key.Modifiers
}

func (GutterClickEvent) isGutterEvent() {}

// GutterHoverEvent is emitted when a user hovers over a gutter area.
type GutterHoverEvent struct {
	// ProviderID is the ID of the provider being hovered.
	ProviderID string

	// Line is the 0-based line number being hovered.
	Line int

	// Info contains the hover information provided by the provider.
	Info *HoverInfo
}

func (GutterHoverEvent) isGutterEvent() {}
