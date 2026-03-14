package colorpicker

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// ColorPreview 颜色预览组件
type ColorPreview struct {
	ColorInfo  ColorInfo
	Clickable  widget.Clickable
	IsHovered  bool
	IsSelected bool
}

// NewColorPreview 创建新的颜色预览
func NewColorPreview(colorInfo ColorInfo) *ColorPreview {
	return &ColorPreview{
		ColorInfo: colorInfo,
	}
}

// Layout 渲染颜色预览
func (cp *ColorPreview) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	cp.Clickable.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		size := gtx.Constraints.Min

		if size.X == 0 {
			size.X = gtx.Dp(unit.Dp(20))
		}
		if size.Y == 0 {
			size.Y = gtx.Dp(unit.Dp(20))
		}

		rect := clip.Rect(image.Rectangle{
			Max: size,
		})

		paint.FillShape(gtx.Ops, cp.ColorInfo.Color, rect.Op())

		if cp.IsHovered || cp.IsSelected {
			borderColor := color.NRGBA{R: 0, G: 120, B: 215, A: 255}
			borderRect := clip.Rect(image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: size.X, Y: size.Y},
			})
			paint.FillShape(gtx.Ops, borderColor, borderRect.Op())
		}

		return layout.Dimensions{Size: size}
	})

	return layout.Dimensions{Size: gtx.Constraints.Min}
}

// ColorPreviewManager 颜色预览管理器
type ColorPreviewManager struct {
	detector      *ColorDetector
	previews      []*ColorPreview
	selectedIndex int
	isOpen        bool
	colorPicker   *ColorPicker
}

// NewColorPreviewManager 创建新的颜色预览管理器
func NewColorPreviewManager() *ColorPreviewManager {
	return &ColorPreviewManager{
		detector:      NewColorDetector(),
		previews:      make([]*ColorPreview, 0),
		selectedIndex: -1,
		isOpen:        false,
	}
}

// UpdatePreviews 更新颜色预览
func (cpm *ColorPreviewManager) UpdatePreviews(text string) {
	colors := cpm.detector.DetectColors(text)
	cpm.previews = make([]*ColorPreview, 0, len(colors))

	for _, colorInfo := range colors {
		cpm.previews = append(cpm.previews, NewColorPreview(colorInfo))
	}
}

// GetPreviews 获取所有颜色预览
func (cpm *ColorPreviewManager) GetPreviews() []*ColorPreview {
	return cpm.previews
}

// HandleClick 处理点击事件
func (cpm *ColorPreviewManager) HandleClick(index int) {
	if index >= 0 && index < len(cpm.previews) {
		cpm.selectedIndex = index
		cpm.isOpen = true

		if cpm.colorPicker == nil {
			cpm.colorPicker = NewColorPicker(cpm.previews[index].ColorInfo.Color)
		} else {
			cpm.colorPicker.State.SelectedColor = cpm.previews[index].ColorInfo.Color
			cpm.colorPicker.State.SetRGB(
				cpm.previews[index].ColorInfo.Color.R,
				cpm.previews[index].ColorInfo.Color.G,
				cpm.previews[index].ColorInfo.Color.B,
			)
		}
	}
}

// ClosePicker 关闭颜色选择器
func (cpm *ColorPreviewManager) ClosePicker() {
	cpm.isOpen = false
	cpm.selectedIndex = -1
}

// IsPickerOpen 检查颜色选择器是否打开
func (cpm *ColorPreviewManager) IsPickerOpen() bool {
	return cpm.isOpen
}

// GetColorPicker 获取颜色选择器
func (cpm *ColorPreviewManager) GetColorPicker() *ColorPicker {
	return cpm.colorPicker
}

// GetSelectedColorInfo 获取选中的颜色信息
func (cpm *ColorPreviewManager) GetSelectedColorInfo() *ColorInfo {
	if cpm.selectedIndex >= 0 && cpm.selectedIndex < len(cpm.previews) {
		return &cpm.previews[cpm.selectedIndex].ColorInfo
	}
	return nil
}

// UpdateHover 更新悬停状态
func (cpm *ColorPreviewManager) UpdateHover(index int, hovered bool) {
	if index >= 0 && index < len(cpm.previews) {
		cpm.previews[index].IsHovered = hovered
	}
}

// LayoutPreviews 渲染所有颜色预览
func (cpm *ColorPreviewManager) LayoutPreviews(gtx layout.Context, th *material.Theme) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(cpm.previews)*2)

	for _, preview := range cpm.previews {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return preview.Layout(gtx, th)
		}))
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Width: unit.Dp(4)}.Layout(gtx)
		}))
	}

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
}

// LayoutPicker 渲染颜色选择器
func (cpm *ColorPreviewManager) LayoutPicker(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if !cpm.isOpen || cpm.colorPicker == nil {
		return layout.Dimensions{}
	}

	return cpm.colorPicker.Layout(gtx, th)
}

// Update 更新颜色选择器状态
func (cpm *ColorPreviewManager) Update(gtx layout.Context) {
	if cpm.colorPicker != nil {
		cpm.colorPicker.Update(gtx)
	}
}

// GetUpdatedColorString 获取更新后的颜色字符串
func (cpm *ColorPreviewManager) GetUpdatedColorString() string {
	if cpm.colorPicker == nil {
		return ""
	}

	selectedInfo := cpm.GetSelectedColorInfo()
	if selectedInfo == nil {
		return ""
	}

	return cpm.colorPicker.State.FormatColorToString(selectedInfo.Format, selectedInfo.Original)
}
