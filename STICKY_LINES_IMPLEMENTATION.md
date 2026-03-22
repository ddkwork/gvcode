# Sticky Lines 功能实现文档

## 功能概述

Sticky Lines（粘性行）是类似 JetBrains GoLand 的功能，当滚动代码时，当前可见代码的上下文（如函数签名、类型定义等）会固定在编辑器顶部显示，方便理解代码结构。

## 实现文件

### 1. 核心提供者 (`gutter/providers/stickylines.go`)

```go
// 主要结构
type StickyLinesProvider struct {
    enabled        bool
    maxStickyLines int
    stickyLines    []StickyLineInfo
    allLines       []string
    structureCache []StickyLineInfo
    clicker        gesture.Click
    pending        []StickyLineEvent
    // ... 其他字段
}

// 关键方法
- analyzeStructure()      // 分析代码结构，识别函数、类型等
- calculateStickyLines()  // 根据滚动位置计算应显示的粘性行
- Layout()                // 布局计算
```

**代码结构识别**：
- 函数定义：`^\s*(func|func\s+\(\s*\w+\s*\*?\s*\w+\s*\))\s+(\w+)\s*\(`
- 类型定义：`^\s*type\s+(\w+)\s+(struct|interface|map|chan|func)`
- 常量/变量块：`^\s*(const|var)\s+\(`
- 导入块：`^\s*import\s*\(`

### 2. 编辑器集成 (`editor.go`)

**新增字段**：
```go
type Editor struct {
    // ... 现有字段
    stickyLinesProvider interface{}
    stickyLineClicker   gesture.Click
}
```

**核心方法** `renderStickyLines`：
1. 从 provider 获取当前应显示的粘性行
2. 处理点击事件，计算点击位置对应的行
3. 注册点击区域
4. 渲染背景和文本
5. 使用缓存优化性能

**跳转功能** `moveToLine`：
```go
func (e *Editor) moveToLine(lineNum int) {
    layouter := e.text.TextLayout()
    para := layouter.Paragraphs[lineNum]
    e.text.ScrollRel(0, para.StartY-e.text.ScrollOff().Y)
}
```

### 3. 配置选项 (`option.go`)

```go
func WithStickyLines() EditorOption {
    return func(e *Editor) {
        provider := providers.NewStickyLinesProvider()
        e.gutterManager.AddProvider(provider)
        e.stickyLinesProvider = provider
    }
}
```

### 4.  gutter 集成 (`gutter.go`)

添加 `feedLineContentsToStickyLinesProvider` 方法，将所有行内容提供给粘性行提供者进行结构分析。

## 关键问题及解决方案

### 问题 1: 初始 Panic
**现象**: 运行时报错，clip 操作未正确配对
**原因**: `clip.Rect().Push()` 后没有正确调用 `Pop()`
**解决**: 确保每个 Push 都有对应的 Pop

### 问题 2: 无显示效果
**现象**: 粘性行没有显示
**原因**: Provider 返回 width 为 0，导致 gutter 不渲染
**解决**: 改为在 `editor.go` 的 `renderStickyLines` 中直接渲染，不依赖 gutter 系统

### 问题 3: 颜色类型冲突
**现象**: 编译错误，颜色类型不匹配
**原因**: 同时使用了 `image/color` 和 `github.com/oligo/gvcode/color`
**解决**: 导入 `image/color` 时使用别名 `stdcolor`

### 问题 4: 点击位置不准确
**现象**: 点击函数头部没有跳转，点击行间才触发
**原因**: 坐标计算和事件处理顺序问题
**解决**:
- 使用 `float32(evt.Position.Y) / float32(lineHeight)` 计算行索引
- 确保 clip 区域覆盖整个粘性行区域

### 问题 5: 性能问题
**现象**: 行数多时卡顿，点击无响应
**原因**: 每帧重新计算文本布局
**解决**:
- 添加缓存机制 `stickyLineCache`
- 只在行数或内容变化时重建缓存
- 缓存预计算的字形数据

## 使用方式

```go
// 在创建 Editor 时启用
codeEditor := gvcode.NewEditor(
    gvcode.WithStickyLines(),
    // ... 其他选项
)
```

## 配置参数

```go
provider.SetMaxStickyLines(5)  // 最多显示 5 行
provider.SetEnabled(true)       // 启用/禁用
```

## 渲染流程

1. **分析阶段**: `analyzeStructure()` 解析代码结构
2. **计算阶段**: `calculateStickyLines()` 根据滚动位置确定显示哪些行
3. **事件阶段**: 处理点击事件，计算点击位置
4. **注册阶段**: 注册点击区域和光标
5. **渲染阶段**: 绘制背景和文本

## 点击处理流程

1. 调用 `clicker.Update()` 获取点击事件
2. 计算点击位置 `clickY / lineHeight` 得到行索引
3. 根据索引找到对应的目标行号
4. 调用 `moveToLine()` 滚动到目标位置

## 注意事项

1. 缓存是全局的，多编辑器实例时可能需要改进
2. 代码结构识别使用正则表达式，可能对复杂情况识别不准确
3. 点击事件在 Gio 中是异步的，需要正确处理事件循环
