package colorpicker

import (
	"gioui.org/layout"
	"gioui.org/widget/material"
)

// EditorColorPicker 编辑器颜色拾色器集成
type EditorColorPicker struct {
	previewManager *ColorPreviewManager
	editorText     string
	onColorChange  func(oldText, newText string, start, end int)
	onClose        func()
}

// NewEditorColorPicker 创建新的编辑器颜色拾色器
func NewEditorColorPicker() *EditorColorPicker {
	return &EditorColorPicker{
		previewManager: NewColorPreviewManager(),
		editorText:     "",
		onColorChange:  nil,
	}
}

// SetEditorText 设置编辑器文本
func (ecp *EditorColorPicker) SetEditorText(text string) {
	ecp.editorText = text
	ecp.previewManager.UpdatePreviews(text)
}

// GetEditorText 获取编辑器文本
func (ecp *EditorColorPicker) GetEditorText() string {
	return ecp.editorText
}

// SetOnColorChange 设置颜色变化回调
func (ecp *EditorColorPicker) SetOnColorChange(callback func(oldText, newText string, start, end int)) {
	ecp.onColorChange = callback
}

// SetOnClose sets the callback for when the color picker is closed.
func (ecp *EditorColorPicker) SetOnClose(callback func()) {
	ecp.onClose = callback
}

// HandleClick 处理颜色预览点击
func (ecp *EditorColorPicker) HandleClick(index int) {
	// Find the preview that matches the selected color info
	previews := ecp.previewManager.GetPreviews()
	if len(previews) == 0 {
		return
	}

	// Use the first preview (or find the correct one based on index)
	ecp.previewManager.HandleClick(0)
	// Set confirm callback to apply color and close picker
	if colorPicker := ecp.previewManager.GetColorPicker(); colorPicker != nil {
		colorPicker.OnConfirm = func() {
			ecp.HandleConfirm()
		}
	}
}

// HandleConfirm 处理颜色选择确认
func (ecp *EditorColorPicker) HandleConfirm() {
	if !ecp.previewManager.IsPickerOpen() {
		return
	}

	selectedInfo := ecp.previewManager.GetSelectedColorInfo()
	if selectedInfo == nil {
		return
	}

	oldText := selectedInfo.Original
	newText := ecp.previewManager.GetUpdatedColorString()
	start := selectedInfo.Range.Start
	end := selectedInfo.Range.End

	if ecp.onColorChange != nil && oldText != newText {
		ecp.onColorChange(oldText, newText, start, end)
	}

	if colorPicker := ecp.previewManager.GetColorPicker(); colorPicker != nil {
		colorPicker.AddToHistory(colorPicker.State.SelectedColor)
	}

	ecp.previewManager.ClosePicker()
	ecp.previewManager.UpdatePreviews(ecp.editorText)

	// Call the close callback
	if ecp.onClose != nil {
		ecp.onClose()
	}
}

// HandleCancel 处理颜色选择取消
func (ecp *EditorColorPicker) HandleCancel() {
	ecp.previewManager.ClosePicker()
}

// IsPickerOpen 检查颜色选择器是否打开
func (ecp *EditorColorPicker) IsPickerOpen() bool {
	return ecp.previewManager.IsPickerOpen()
}

// Update 更新状态
func (ecp *EditorColorPicker) Update(gtx layout.Context) {
	ecp.previewManager.Update(gtx)
}

// LayoutPreviews 渲染颜色预览
func (ecp *EditorColorPicker) LayoutPreviews(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return ecp.previewManager.LayoutPreviews(gtx, th)
}

// LayoutPicker 渲染颜色选择器
func (ecp *EditorColorPicker) LayoutPicker(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return ecp.previewManager.LayoutPicker(gtx, th)
}

// Layout 渲染颜色选择器（实现 ColorPickerLayout 接口）
func (ecp *EditorColorPicker) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return ecp.previewManager.LayoutPicker(gtx, th)
}

// GetPreviews 获取所有颜色预览
func (ecp *EditorColorPicker) GetPreviews() []*ColorPreview {
	return ecp.previewManager.GetPreviews()
}

// UpdateHover 更新悬停状态
func (ecp *EditorColorPicker) UpdateHover(index int, hovered bool) {
	ecp.previewManager.UpdateHover(index, hovered)
}

// ApplyColorChange 应用颜色变化到编辑器文本
func (ecp *EditorColorPicker) ApplyColorChange(oldText, newText string, start, end int) {
	if start < 0 || end > len(ecp.editorText) {
		return
	}

	newEditorText := ecp.editorText[:start] + newText + ecp.editorText[end:]
	ecp.editorText = newEditorText
	ecp.previewManager.UpdatePreviews(ecp.editorText)
}
