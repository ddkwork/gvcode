# 代码折叠功能实现文档

## 功能概述

代码折叠功能允许用户折叠和展开代码块（函数、类型、导入块、常量块等），类似于 GoLand 的代码折叠功能。

## 实现文件

### 1. 核心折叠管理器 (`internal/folding/folding.go`)

```go
// 主要结构
type Manager struct {
    foldRanges     []FoldRange      // 所有检测到的折叠区域
    collapsedLines map[int]bool     // 被折叠的行
    lineCache      []string         // 上次分析的行缓存
}

// 支持的折叠类型
const (
    FoldTypeFunction  // 函数/方法
    FoldTypeType      // 类型定义
    FoldTypeComment   // 多行注释
    FoldTypeImport    // 导入块
    FoldTypeConst     // 常量块
    FoldTypeVar       // 变量块
    FoldTypeRegion    // 用户定义区域
)
```

**关键方法：**
- `AnalyzeLines()` - 分析代码结构，检测可折叠区域
- `detectFolds()` - 使用正则表达式识别代码块
- `ToggleFold()` - 切换折叠状态
- `CollapseAll()` / `ExpandAll()` - 全部折叠/展开
- `IsLineVisible()` - 检查行是否可见（未被折叠）

**代码结构识别：**
- 函数定义：`^func\s+(?:\([^)]+\)\s+)?(\w+)`
- 类型定义：`^type\s+(\w+)`
- 导入/常量/变量块：`import (`, `const (`, `var (`
- 多行注释：`/* ... */`
- 用户区域：`//region Name`

### 2. 折叠按钮提供者 (`gutter/providers/foldbutton.go`)

在 gutter 中显示折叠/展开按钮：
- 折叠状态显示减号 (-)
- 展开状态显示加号 (+)
- 支持点击切换折叠状态
- 悬停显示提示信息

### 3. 布局系统集成 (`internal/layout/text_layout.go`)

修改 `trackLines()` 方法：
- 在生成 Paragraphs 时检查行是否被折叠
- 被折叠的行不会添加到可见段落列表
- 保留段落索引映射关系

### 4. TextView 集成 (`textview/text.go`)

添加折叠管理器支持：
```go
func (e *TextView) SetFoldManager(fm *folding.Manager)
func (e *TextView) FoldManager() *folding.Manager
func (e *TextView) Invalidate()  // 强制重新布局
```

### 5. 编辑器集成 (`editor.go`, `commands.go`, `option.go`)

**配置选项：**
```go
gvcode.WithCodeFolding()  // 启用代码折叠
```

**快捷键：**
- `Ctrl+[` - 折叠当前行的代码块
- `Ctrl+]` - 展开当前行的代码块
- `Ctrl+Shift+[` - 折叠所有代码块
- `Ctrl+Shift+]` - 展开所有代码块

**编辑器方法：**
```go
func (e *Editor) Invalidate()           // 强制重新布局
func (e *Editor) toggleFoldAtCaret()    // 切换当前行的折叠状态
```

### 6. Gutter 集成 (`gutter.go`)

添加 `feedLineContentsToFoldButtonProvider()` 方法：
- 将所有行内容传递给折叠按钮提供者
- 提供者分析代码结构并显示折叠按钮

## 使用方式

```go
// 创建编辑器时启用代码折叠
editor := gvcode.NewEditor(
    gvcode.WithCodeFolding(),
    gvcode.WithDefaultGutters(),
    // ... 其他选项
)
```

## 渲染流程

1. **分析阶段**: `folding.Manager.AnalyzeLines()` 解析代码结构
2. **按钮渲染**: `FoldButtonProvider.Layout()` 在 gutter 显示折叠按钮
3. **布局阶段**: `TextLayout.trackLines()` 跳过被折叠的行
4. **文本渲染**: `TextView` 只渲染可见段落
5. **事件处理**: 点击按钮或快捷键触发折叠/展开

## 折叠状态管理

```go
// 获取折叠管理器
fm := editor.FoldManager()

// 操作折叠状态
fm.ToggleFold(lineNumber)     // 切换指定行的折叠状态
fm.CollapseFold(lineNumber)   // 折叠指定行
fm.ExpandFold(lineNumber)     // 展开指定行
fm.CollapseAll()              // 折叠所有
fm.ExpandAll()                // 展开所有

// 查询折叠信息
fold := fm.GetFoldAtLine(lineNumber)      // 获取指定行的折叠信息
folds := fm.GetFoldRanges()               // 获取所有折叠区域
visible := fm.IsLineVisible(lineNumber)   // 检查行是否可见
```

## 与粘性行的集成

代码折叠与粘性行功能完美配合：
- 粘性行显示当前可见代码的上下文
- 折叠的代码块不会出现在粘性行中
- 点击粘性行可以跳转到对应位置（考虑折叠状态）

## 性能优化

1. **缓存机制**: 折叠管理器缓存上次分析的行，避免重复分析
2. **增量更新**: 只在行数或内容变化时重新分析
3. **可见性过滤**: 布局时只处理可见段落

## 注意事项

1. 代码结构识别使用正则表达式，可能对复杂语法识别不准确
2. 折叠状态不会持久化保存（重启后丢失）
3. 多编辑器实例共享全局缓存可能需要改进
4. 大文件折叠操作可能需要优化性能
