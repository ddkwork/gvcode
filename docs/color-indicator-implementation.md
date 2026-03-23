# 颜色指示器实现文档

## 概述

颜色指示器是一个代码编辑器功能，用于在代码中检测颜色值（如 `#FF0000`、`rgb(255,0,0)` 等）并在其旁边显示一个可视化的颜色指示器。用户可以点击颜色指示器打开颜色选择器，方便地修改颜色值。

## 架构设计

### 主要组件

1. **ColorIndicatorProvider** - 颜色指示器提供者
   - 位置：`gutter/providers/colorindicator.go`
   - 负责检测代码中的颜色值并渲染颜色指示器

2. **EditorColorPicker** - 编辑器颜色选择器
   - 位置：`gutter/providers/editorcolorpicker.go`
   - 提供颜色选择器界面和交互逻辑

3. **ColorDetector** - 颜色检测器
   - 位置：`colorpicker/detector.go`
   - 使用正则表达式检测代码中的颜色值

4. **文本布局集成** - 文本布局系统集成
   - 位置：`internal/layout/line.go`、`internal/layout/text_layout.go`
   - 在文本布局阶段为颜色指示器预留空间

## 接口设计

### ColorPickerProvider 接口

```go
type ColorPickerProvider interface {
    GutterProvider
    ShowColorPicker() bool
    GetEditorColorPicker() ColorPickerLayout
    RenderInTextArea(gtx layout.Context, ctx GutterContext, gutterWidth int)
    GetColorOffsets() map[int][]int
    GetIndicatorWidth(gtx layout.Context) int
}
```

### 关键方法说明

- `GetColorOffsets()` - 返回字符偏移量，用于指示在哪些位置插入颜色指示器
- `GetIndicatorWidth()` - 返回颜色指示器的宽度（像素）
- `RenderInTextArea()` - 在文本区域渲染颜色指示器

## 实现细节

### 1. 颜色检测

颜色检测器使用正则表达式匹配以下格式的颜色值：

```go
var colorPatterns = []string{
    `#([0-9a-fA-F]{3,4})`,           // #RGB, #RGBA
    `#([0-9a-fA-F]{6,8})`,           // #RRGGBB, #RRGGBBAA
    `rgba?\(\s*(\d{1,3}%?)\s*,\s*(\d{1,3}%?)\s*,\s*(\d{1,3}%?)\s*(?:,\s*([01]?\.?\d*)\s*)?\)`, // rgb(), rgba()
    `hsla?\(\s*(\d{1,3}(?:deg)?)\s*,\s*(\d{1,3}%?)\s*,\s*(\d{1,3}%?)\s*(?:,\s*([01]?\.?\d*)\s*)?\)`, // hsl(), hsla()
}
```

### 2. 颜色指示器渲染

颜色指示器在文本区域渲染，而不是在 gutter 中，这样可以更精确地定位到颜色值的位置。

```go
func (p *ColorIndicatorProvider) RenderInTextArea(gtx layout.Context, ctx gutter.GutterContext, gutterWidth int) {
    indicatorSizePx := gtx.Dp(unit.Dp(indicatorSize))

    for _, para := range ctx.Paragraphs {
        colors, hasColors := p.colorInfos[para.Index]
        if !hasColors || len(colors) == 0 {
            continue
        }

        // 计算颜色指示器的位置
        for i, colorInfo := range colors {
            xPos := gutterWidth + gapPx + i*(indicatorSizePx+gapPx)
            yPos := para.StartY + (para.EndY-para.StartY-indicatorSizePx)/2

            // 渲染颜色指示器
            p.renderColorIndicator(gtx, colorInfo.Color, xPos, yPos, indicatorSizePx)
        }
    }
}
```

### 3. 文本布局集成

为了确保颜色指示器不与代码重叠，我们在文本布局阶段为颜色指示器预留空间。

#### 修改 Line.recompute 方法

```go
func (li *Line) recompute(alignOff fixed.Int26_6, runeOff int, colorOffsets map[int]int) {
    // 跟踪当前字符位置
    charPos := 0
    // 跟踪累积的颜色指示器偏移量
    colorOffset := fixed.I(0)

    for j := runStart; j < i; j++ {
        // 检查是否需要在此字形之前添加颜色指示器偏移量
        if offset, hasOffset := colorOffsets[charPos]; hasOffset {
            colorOffset += fixed.I(offset)
        }

        li.Glyphs[j].X = cursor + colorOffset
        cursor += li.Glyphs[j].Advance

        // 更新字符位置
        charPos += int(li.Glyphs[j].Runes)
    }

    // 更新行宽度以包含颜色指示器偏移量
    li.Width += colorOffset
}
```

#### 修改 TextLayout

```go
type TextLayout struct {
    // ... 其他字段

    // colorOffsets 映射行号到字符位置，指示在哪里插入颜色指示器
    colorOffsets map[int]map[int]int
}

func (tl *TextLayout) SetColorOffsets(offsets map[int]map[int]int) {
    tl.colorOffsets = offsets
}

func (tl *TextLayout) calculateXOffsets() {
    runeOff := 0
    for i, line := range tl.Lines {
        alignOff := tl.params.Alignment.Align(tl.params.Locale.Direction, line.Width, tl.params.MaxWidth)

        // 获取此行的颜色偏移量
        var lineColorOffsets map[int]int
        if tl.colorOffsets != nil {
            lineColorOffsets = tl.colorOffsets[i]
        }

        tl.Lines[i].recompute(alignOff, runeOff, lineColorOffsets)
        runeOff += line.Runes
    }
}
```

### 4. 编辑器集成

编辑器在文本布局之前设置颜色偏移信息。

```go
func (e *Editor) setColorOffsets(gtx layout.Context) {
    if e.gutterManager == nil {
        return
    }

    // 查找颜色指示器提供者
    var colorPickerProvider gutter.ColorPickerProvider
    providers := e.gutterManager.Providers()
    for _, p := range providers {
        if p.ID() == "colorindicator" {
            if ci, ok := p.(gutter.ColorPickerProvider); ok {
                colorPickerProvider = ci
                break
            }
        }
    }

    if colorPickerProvider == nil {
        return
    }

    // 从提供者获取颜色偏移量
    colorOffsets := colorPickerProvider.GetColorOffsets()
    if len(colorOffsets) == 0 {
        return
    }

    // 将颜色偏移量转换为文本布局期望的格式
    indicatorWidth := colorPickerProvider.GetIndicatorWidth(gtx)
    layoutOffsets := make(map[int]map[int]int)

    for line, offsets := range colorOffsets {
        lineOffsets := make(map[int]int)
        for _, offset := range offsets {
            lineOffsets[offset] = indicatorWidth
        }
        layoutOffsets[line] = lineOffsets
    }

    // 在文本布局中设置颜色偏移量
    e.text.SetColorOffsets(layoutOffsets)
}
```

## 工作流程

### 初始化阶段

1. 创建 `ColorIndicatorProvider` 实例
2. 将提供者添加到 gutter 管理器
3. 设置编辑器颜色选择器

### 渲染阶段

1. **颜色检测**
   - 编辑器调用 `SetLineContents` 方法传递可见行的内容
   - `ColorIndicatorProvider` 使用 `ColorDetector` 检测颜色值
   - 将检测到的颜色信息按行分组存储

2. **文本布局**
   - 编辑器调用 `setColorOffsets` 获取颜色偏移信息
   - 将偏移信息传递给 `TextView`
   - `TextView` 将偏移信息传递给 `TextLayout`
   - `TextLayout` 在计算字形位置时插入额外的空间

3. **渲染颜色指示器**
   - 编辑器调用 `RenderInTextArea` 渲染颜色指示器
   - 在预留的空间位置绘制颜色指示器

### 交互阶段

1. **点击颜色指示器**
   - 用户点击颜色指示器
   - `ColorIndicatorProvider` 处理点击事件
   - 打开颜色选择器

2. **修改颜色**
   - 用户在颜色选择器中选择新颜色
   - `EditorColorPicker` 更新颜色值
   - 编辑器更新代码中的颜色值

3. **关闭颜色选择器**
   - 用户点击其他位置或按 ESC 键
   - 颜色选择器关闭

## 关键代码文件

| 文件 | 说明 |
|------|------|
| `gutter/providers/colorindicator.go` | 颜色指示器提供者实现 |
| `gutter/providers/editorcolorpicker.go` | 编辑器颜色选择器实现 |
| `colorpicker/detector.go` | 颜色检测器实现 |
| `gutter/gutter.go` | Gutter 接口定义 |
| `internal/layout/line.go` | 文本行布局实现 |
| `internal/layout/text_layout.go` | 文本布局实现 |
| `textview/text.go` | 文本视图实现 |
| `editor.go` | 编辑器实现 |

## 配置选项

### 颜色指示器大小

```go
const indicatorSize = 16 // 颜色指示器大小（像素）
```

### 间距

```go
const gapPx = 2 // 颜色指示器之间的间距（像素）
```

### Gutter 优先级

```go
func (p *ColorIndicatorProvider) Priority() int {
    return 140 // 在行号（100）和运行按钮（150）之间
}
```

## 使用方法

### 1. 创建颜色指示器提供者

```go
colorIndicator := providers.NewColorIndicatorProvider(editorApp.state)
```

### 2. 添加到 gutter 管理器

```go
gutterManager := gutter.NewManager()
gutterManager.AddProvider(colorIndicator)
```

### 3. 设置到编辑器

```go
editor.SetGutterManager(gutterManager)
```

## 性能优化

1. **颜色检测缓存** - 颜色检测结果按行缓存，避免重复检测
2. **增量更新** - 只在文本内容变化时重新检测颜色
3. **可见行检测** - 只检测可见行中的颜色值
4. **延迟渲染** - 颜色指示器在文本布局之后渲染

## 已知限制

1. **多颜色指示器** - 当一行中有多个颜色值时，所有颜色指示器会显示在相同位置
2. **颜色格式** - 只支持常见的颜色格式，不支持自定义颜色变量
3. **性能** - 在大文件中检测颜色可能会有性能影响

## 未来改进

1. **多颜色指示器支持** - 为每个颜色值显示独立的指示器
2. **颜色变量支持** - 支持检测和编辑 CSS 变量等颜色变量
3. **性能优化** - 使用更高效的颜色检测算法
4. **自定义样式** - 允许用户自定义颜色指示器的样式

## 参考资料

- [Gio 文本布局](https://gioui.org/widget/text)
- [颜色格式规范](https://www.w3.org/TR/css-color-3/)
- [正则表达式教程](https://regexone.com/)
