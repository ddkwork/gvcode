package colorpicker

import (
	"fmt"
	"image/color"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// ColorInfo 颜色信息
type ColorInfo struct {
	Color    color.NRGBA
	Range    TextRange
	Format   ColorFormat
	Original string
}

// TextRange 文本范围
type TextRange struct {
	Start int
	End   int
}

// ColorFormat 颜色格式
type ColorFormat int

const (
	ColorFormatHex3 ColorFormat = iota
	ColorFormatHex4
	ColorFormatHex6
	ColorFormatHex8
	ColorFormatRGB
	ColorFormatRGBA
	ColorFormatHSL
	ColorFormatHSLA
	ColorFormatNamed
)

// ColorDetector 颜色检测器
type ColorDetector struct {
	hex3Regex   *regexp.Regexp
	hex4Regex   *regexp.Regexp
	hex6Regex   *regexp.Regexp
	hex8Regex   *regexp.Regexp
	rgbRegex    *regexp.Regexp
	rgbaRegex   *regexp.Regexp
	hslRegex    *regexp.Regexp
	hslaRegex   *regexp.Regexp
	namedColors map[string]color.NRGBA
}

// NewColorDetector 创建新的颜色检测器
func NewColorDetector() *ColorDetector {
	return &ColorDetector{
		hex3Regex:   regexp.MustCompile(`#([0-9a-fA-F]{3})\b`),
		hex4Regex:   regexp.MustCompile(`#([0-9a-fA-F]{4})\b`),
		hex6Regex:   regexp.MustCompile(`#([0-9a-fA-F]{6})\b`),
		hex8Regex:   regexp.MustCompile(`#([0-9a-fA-F]{8})\b`),
		rgbRegex:    regexp.MustCompile(`(?i)rgb\s*\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*\)`),
		rgbaRegex:   regexp.MustCompile(`(?i)rgba\s*\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*,\s*([0-9.]+)\s*\)`),
		hslRegex:    regexp.MustCompile(`(?i)hsl\s*\(\s*(\d+)\s*,\s*(\d+)%\s*,\s*(\d+)%\s*\)`),
		hslaRegex:   regexp.MustCompile(`(?i)hsla\s*\(\s*(\d+)\s*,\s*(\d+)%\s*,\s*(\d+)%\s*,\s*([0-9.]+)\s*\)`),
		namedColors: getNamedColors(),
	}
}

// DetectColors 检测文本中的所有颜色
func (cd *ColorDetector) DetectColors(text string) []ColorInfo {
	var colors []ColorInfo

	colors = append(colors, cd.detectHexColors(text)...)
	colors = append(colors, cd.detectRGBColors(text)...)
	colors = append(colors, cd.detectHSLColors(text)...)
	colors = append(colors, cd.detectNamedColors(text)...)

	return colors
}

// detectHexColors 检测十六进制颜色
func (cd *ColorDetector) detectHexColors(text string) []ColorInfo {
	var colors []ColorInfo

	matches := cd.hex8Regex.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		hex := text[match[2]:match[3]]
		if c, err := parseHexColor(hex); err == nil {
			colors = append(colors, ColorInfo{
				Color:    c,
				Range:    TextRange{Start: match[0], End: match[1]},
				Format:   ColorFormatHex8,
				Original: text[match[0]:match[1]],
			})
		}
	}

	matches = cd.hex6Regex.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		hex := text[match[2]:match[3]]
		if c, err := parseHexColor(hex); err == nil {
			colors = append(colors, ColorInfo{
				Color:    c,
				Range:    TextRange{Start: match[0], End: match[1]},
				Format:   ColorFormatHex6,
				Original: text[match[0]:match[1]],
			})
		}
	}

	matches = cd.hex4Regex.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		hex := text[match[2]:match[3]]
		if c, err := parseHexColor(hex); err == nil {
			colors = append(colors, ColorInfo{
				Color:    c,
				Range:    TextRange{Start: match[0], End: match[1]},
				Format:   ColorFormatHex4,
				Original: text[match[0]:match[1]],
			})
		}
	}

	matches = cd.hex3Regex.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		hex := text[match[2]:match[3]]
		if c, err := parseHexColor(hex); err == nil {
			colors = append(colors, ColorInfo{
				Color:    c,
				Range:    TextRange{Start: match[0], End: match[1]},
				Format:   ColorFormatHex3,
				Original: text[match[0]:match[1]],
			})
		}
	}

	return colors
}

// detectRGBColors 检测 RGB 颜色
func (cd *ColorDetector) detectRGBColors(text string) []ColorInfo {
	var colors []ColorInfo

	matches := cd.rgbaRegex.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		r, _ := strconv.Atoi(text[match[2]:match[3]])
		g, _ := strconv.Atoi(text[match[4]:match[5]])
		b, _ := strconv.Atoi(text[match[6]:match[7]])
		a, _ := strconv.ParseFloat(text[match[8]:match[9]], 64)

		colors = append(colors, ColorInfo{
			Color: color.NRGBA{
				R: uint8(r),
				G: uint8(g),
				B: uint8(b),
				A: uint8(a * 255),
			},
			Range:    TextRange{Start: match[0], End: match[1]},
			Format:   ColorFormatRGBA,
			Original: text[match[0]:match[1]],
		})
	}

	matches = cd.rgbRegex.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		r, _ := strconv.Atoi(text[match[2]:match[3]])
		g, _ := strconv.Atoi(text[match[4]:match[5]])
		b, _ := strconv.Atoi(text[match[6]:match[7]])

		colors = append(colors, ColorInfo{
			Color: color.NRGBA{
				R: uint8(r),
				G: uint8(g),
				B: uint8(b),
				A: 255,
			},
			Range:    TextRange{Start: match[0], End: match[1]},
			Format:   ColorFormatRGB,
			Original: text[match[0]:match[1]],
		})
	}

	return colors
}

// detectHSLColors 检测 HSL 颜色
func (cd *ColorDetector) detectHSLColors(text string) []ColorInfo {
	var colors []ColorInfo

	matches := cd.hslaRegex.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		h, _ := strconv.Atoi(text[match[2]:match[3]])
		s, _ := strconv.Atoi(text[match[4]:match[5]])
		l, _ := strconv.Atoi(text[match[6]:match[7]])
		a, _ := strconv.ParseFloat(text[match[8]:match[9]], 64)

		c := hslToNRGBA(float64(h)/360.0, float64(s)/100.0, float64(l)/100.0)
		c.A = uint8(a * 255)

		colors = append(colors, ColorInfo{
			Color:    c,
			Range:    TextRange{Start: match[0], End: match[1]},
			Format:   ColorFormatHSLA,
			Original: text[match[0]:match[1]],
		})
	}

	matches = cd.hslRegex.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		h, _ := strconv.Atoi(text[match[2]:match[3]])
		s, _ := strconv.Atoi(text[match[4]:match[5]])
		l, _ := strconv.Atoi(text[match[6]:match[7]])

		c := hslToNRGBA(float64(h)/360.0, float64(s)/100.0, float64(l)/100.0)

		colors = append(colors, ColorInfo{
			Color:    c,
			Range:    TextRange{Start: match[0], End: match[1]},
			Format:   ColorFormatHSL,
			Original: text[match[0]:match[1]],
		})
	}

	return colors
}

// detectNamedColors 检测命名颜色
func (cd *ColorDetector) detectNamedColors(text string) []ColorInfo {
	var colors []ColorInfo

	for name, c := range cd.namedColors {
		index := 0
		for {
			pos := strings.Index(text[index:], name)
			if pos == -1 {
				break
			}
			pos += index

			start := pos
			end := pos + len(name)

			if (start == 0 || !isAlphaNum(text[start-1])) &&
				(end == len(text) || !isAlphaNum(text[end])) {
				colors = append(colors, ColorInfo{
					Color:    c,
					Range:    TextRange{Start: start, End: end},
					Format:   ColorFormatNamed,
					Original: text[start:end],
				})
			}

			index = end
		}
	}

	return colors
}

// parseHexColor 解析十六进制颜色
func parseHexColor(hex string) (color.NRGBA, error) {
	var r, g, b, a uint8
	a = 255

	switch len(hex) {
	case 3:
		r = parseHexDigit(hex[0]) * 17
		g = parseHexDigit(hex[1]) * 17
		b = parseHexDigit(hex[2]) * 17
	case 4:
		r = parseHexDigit(hex[0]) * 17
		g = parseHexDigit(hex[1]) * 17
		b = parseHexDigit(hex[2]) * 17
		a = parseHexDigit(hex[3]) * 17
	case 6:
		r = parseHexDigit(hex[0])<<4 | parseHexDigit(hex[1])
		g = parseHexDigit(hex[2])<<4 | parseHexDigit(hex[3])
		b = parseHexDigit(hex[4])<<4 | parseHexDigit(hex[5])
	case 8:
		r = parseHexDigit(hex[0])<<4 | parseHexDigit(hex[1])
		g = parseHexDigit(hex[2])<<4 | parseHexDigit(hex[3])
		b = parseHexDigit(hex[4])<<4 | parseHexDigit(hex[5])
		a = parseHexDigit(hex[6])<<4 | parseHexDigit(hex[7])
	default:
		return color.NRGBA{}, nil
	}

	return color.NRGBA{R: r, G: g, B: b, A: a}, nil
}

// parseHexDigit 解析十六进制数字
func parseHexDigit(c byte) uint8 {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	default:
		return 0
	}
}

// hslToNRGBA 将 HSL 转换为 NRGBA
func hslToNRGBA(h, s, l float64) color.NRGBA {
	var r, g, b float64

	if s == 0 {
		r = l
		g = l
		b = l
	} else {
		var hue2rgb func(p, q, t float64) float64
		hue2rgb = func(p, q, t float64) float64 {
			if t < 0 {
				t += 1
			}
			if t > 1 {
				t -= 1
			}
			if t < 1.0/6.0 {
				return p + (q-p)*6*t
			}
			if t < 1.0/2.0 {
				return q
			}
			if t < 2.0/3.0 {
				return p + (q-p)*(2.0/3.0-t)*6
			}
			return p
		}

		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q

		r = hue2rgb(p, q, h+1.0/3.0)
		g = hue2rgb(p, q, h)
		b = hue2rgb(p, q, h-1.0/3.0)
	}

	return color.NRGBA{
		R: uint8(r * 255),
		G: uint8(g * 255),
		B: uint8(b * 255),
		A: 255,
	}
}

// isAlphaNum 检查字符是否为字母或数字
func isAlphaNum(c byte) bool {
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9')
}

// getNamedColors 获取命名颜色映射
func getNamedColors() map[string]color.NRGBA {
	return map[string]color.NRGBA{
		"black":   {R: 0, G: 0, B: 0, A: 255},
		"white":   {R: 255, G: 255, B: 255, A: 255},
		"red":     {R: 255, G: 0, B: 0, A: 255},
		"green":   {R: 0, G: 128, B: 0, A: 255},
		"blue":    {R: 0, G: 0, B: 255, A: 255},
		"yellow":  {R: 255, G: 255, B: 0, A: 255},
		"cyan":    {R: 0, G: 255, B: 255, A: 255},
		"magenta": {R: 255, G: 0, B: 255, A: 255},
		"gray":    {R: 128, G: 128, B: 128, A: 255},
		"grey":    {R: 128, G: 128, B: 128, A: 255},
		"orange":  {R: 255, G: 165, B: 0, A: 255},
		"pink":    {R: 255, G: 192, B: 203, A: 255},
		"purple":  {R: 128, G: 0, B: 128, A: 255},
		"brown":   {R: 165, G: 42, B: 42, A: 255},
		"lime":    {R: 0, G: 255, B: 0, A: 255},
		"navy":    {R: 0, G: 0, B: 128, A: 255},
		"teal":    {R: 0, G: 128, B: 128, A: 255},
		"olive":   {R: 128, G: 128, B: 0, A: 255},
		"maroon":  {R: 128, G: 0, B: 0, A: 255},
		"aqua":    {R: 0, G: 255, B: 255, A: 255},
		"fuchsia": {R: 255, G: 0, B: 255, A: 255},
		"silver":  {R: 192, G: 192, B: 192, A: 255},
	}
}

// FormatColorToString 将颜色格式化为字符串
func (ci *ColorInfo) FormatColorToString() string {
	switch ci.Format {
	case ColorFormatHex3, ColorFormatHex4, ColorFormatHex6, ColorFormatHex8:
		return rgbToHex(ci.Color.R, ci.Color.G, ci.Color.B)
	case ColorFormatRGB:
		return fmt.Sprintf("rgb(%d, %d, %d)", ci.Color.R, ci.Color.G, ci.Color.B)
	case ColorFormatRGBA:
		return fmt.Sprintf("rgba(%d, %d, %d, %.2f)", ci.Color.R, ci.Color.G, ci.Color.B, float64(ci.Color.A)/255.0)
	case ColorFormatHSL:
		h, s, l := nrgbaToHSL(ci.Color)
		return fmt.Sprintf("hsl(%d, %d%%, %d%%)", int(h*360), int(s*100), int(l*100))
	case ColorFormatHSLA:
		h, s, l := nrgbaToHSL(ci.Color)
		return fmt.Sprintf("hsla(%d, %d%%, %d%%, %.2f)", int(h*360), int(s*100), int(l*100), float64(ci.Color.A)/255.0)
	case ColorFormatNamed:
		return ci.Original
	default:
		return rgbToHex(ci.Color.R, ci.Color.G, ci.Color.B)
	}
}

// nrgbaToHSL 将 NRGBA 转换为 HSL
func nrgbaToHSL(c color.NRGBA) (h, s, l float64) {
	r := float64(c.R) / 255.0
	g := float64(c.G) / 255.0
	b := float64(c.B) / 255.0

	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	delta := max - min

	l = (max + min) / 2.0

	if delta == 0 {
		h = 0
		s = 0
	} else {
		if l < 0.5 {
			s = delta / (max + min)
		} else {
			s = delta / (2.0 - max - min)
		}

		switch max {
		case r:
			h = (g - b) / delta
			if g < b {
				h += 6
			}
		case g:
			h = (b-r)/delta + 2
		case b:
			h = (r-g)/delta + 4
		}
		h /= 6.0
	}

	return h, s, l
}
