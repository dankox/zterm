package zterm

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/awesome-gocui/gocui"
	"github.com/muesli/termenv"
	"golang.org/x/image/colornames"
)

var (
	// TUI coloring
	cFgColor, cFgColorStr     = gocui.ColorDefault, termenv.ForegroundColor()
	cBgColor, cBgColorStr     = gocui.ColorDefault, termenv.BackgroundColor()
	cFrame, cFrameStr         = AttributeAnsi(gocui.ColorGreen)
	cFrameSel, cFrameSelStr   = AttributeAnsi(gocui.ColorYellow)
	cConsole, cConsoleStr     = AttributeAnsi(gocui.ColorCyan)
	cError, cErrorStr         = AttributeAnsi(gocui.ColorRed)
	cHighlight, cHighlightStr = AttributeAnsi(gocui.ColorMagenta)

	// For coloring, if set (thru theme color-space:basic) colors are converted to `3x;1m` for normal or `3x;2m` for bright
	colorBasic = false
)

// BasicColor is for termenv to be able to convert to `\033[3x;1m` or `\033[3x;2m`
type BasicColor int

// Sequence return string with ANSI code
func (c BasicColor) Sequence(bg bool) string {
	col := int(c)
	bgMod := func(c int) int {
		if bg {
			return c + 10
		}
		return c
	}

	// NOTE: Maybe it's only on Windows 7 without ANSI support?
	// This is how termbox is dealing with colors, gocui.AttrBold is actually `\1xb[2m` (512 -> x0200 -> x02 -> 2m)
	if col < 8 {
		// this is normal, the same as \x1b[1m -> bold (who knows why?)
		return fmt.Sprintf("%d", bgMod(col)+30)
	}
	// \x1b[2m -> it's faint but termbox took it as AttrBold
	return fmt.Sprintf("%d;2", bgMod(col-8)+30)
}

// Theme configuration
type Theme struct {
	ColorSpace string `mapstructure:"color-space"`
	FgColor    string `mapstructure:"fgcolor"`
	BgColor    string `mapstructure:"bgcolor"`
	Frame      string `mapstructure:"frame"`
	FrameSel   string `mapstructure:"frame-select"`
	Console    string `mapstructure:"console"`
	Error      string `mapstructure:"error"`
	Highlight  string `mapstructure:"highlight"`
}

// LoadTheme loads theme specified in config file.
func LoadTheme() {
	if a, c, e := StringAttributeAnsi(config.Theme.FgColor); e == nil {
		cFgColor, cFgColorStr = a, c
	}
	if a, c, e := StringAttributeAnsi(config.Theme.BgColor); e == nil {
		cBgColor, cBgColorStr = a, c
	}
	if a, c, e := StringAttributeAnsi(config.Theme.Frame); e == nil {
		cFrame, cFrameStr = a, c
	}
	if a, c, e := StringAttributeAnsi(config.Theme.FrameSel); e == nil {
		cFrameSel, cFrameSelStr = a, c
	}
	if a, c, e := StringAttributeAnsi(config.Theme.Console); e == nil {
		cConsole, cConsoleStr = a, c
	}
	if a, c, e := StringAttributeAnsi(config.Theme.Error); e == nil {
		cError, cErrorStr = a, c
	}
	if a, c, e := StringAttributeAnsi(config.Theme.Highlight); e == nil {
		cHighlight, cHighlightStr = a, c
	}
}

// AttributeAnsi converts gocui.Attribute to ANSI color and returns both of them
func AttributeAnsi(col gocui.Attribute) (gocui.Attribute, termenv.Color) {
	return col, termenv.ANSI.Color(strconv.Itoa(int(col) - 1))
}

// StringAttributeAnsi converts string to gocui.Attribute and returns both of them
func StringAttributeAnsi(col string) (gocui.Attribute, termenv.Color, error) {
	if len(col) == 0 {
		return 0, nil, errors.New("no color specified")
	}
	p := ColorProfile()
	if p == termenv.Ascii {
		return 0, nil, errors.New("ascii profile, nothing to convert")
	}
	pcol := p.Color(col)
	if pcol == nil {
		// look for color names
		if c, ok := colornames.Map[col]; ok {
			col = fmt.Sprintf("#%.2x%.2x%.2x", c.R, c.G, c.B)
			pcol = p.Color(col)
		} else {
			return 0, nil, errors.New("cannot find color name") // this will keep the default
		}
	}

	switch v := pcol.(type) {
	case termenv.ANSIColor:
		if colorBasic {
			bv := BasicColor(v)
			if bv > 7 {
				return gocui.Attribute(v-7) | gocui.AttrBold, bv, nil
			}
			return gocui.Attribute(v + 1), bv, nil
		}
		return gocui.Attribute(v + 1), v, nil

	case termenv.ANSI256Color:
		return gocui.Attribute(v + 1), v, nil

	case termenv.RGBColor:
		if npcol := termenv.ANSI256.Color(col); npcol != nil {
			switch nv := npcol.(type) {
			case termenv.ANSIColor:
				return gocui.Attribute(nv + 1), v, nil
			case termenv.ANSI256Color:
				return gocui.Attribute(nv + 1), v, nil
			}
		}
	}

	return 0, nil, errors.New("cannot convert color") // this will keep the default
}

// ColorProfile returns color profile based on configuration
func ColorProfile() termenv.Profile {
	p := termenv.ColorProfile()
	switch config.Theme.ColorSpace {
	case "basic":
		p = termenv.ANSI
		colorBasic = true
	case "ansi256":
		p = termenv.ANSI256
	case "truecolor":
		p = termenv.TrueColor
	}
	return p
}

// func colorText(text string, color string) string {
func colorText(text string, color termenv.Color) string {
	return termenv.String(text).Foreground(color).String()
}
