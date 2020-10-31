package monitor

import (
	"strconv"

	"github.com/awesome-gocui/gocui"
	"github.com/muesli/termenv"
)

var (
	// TUI coloring
	cFrame, cFrameStr         = AttributeAnsi(gocui.ColorGreen)
	cFrameSel, cFrameSelStr   = AttributeAnsi(gocui.ColorYellow)
	cConsole, cConsoleStr     = AttributeAnsi(gocui.ColorCyan)
	cError, cErrorStr         = AttributeAnsi(gocui.ColorRed)
	cHighlight, cHighlightStr = AttributeAnsi(gocui.ColorMagenta)
)

// AttributeAnsi converts gocui.Attribute to ANSI color and returns both of them
func AttributeAnsi(col gocui.Attribute) (gocui.Attribute, string) {
	return col, strconv.Itoa(int(col) - 1)
}

func colorText(text string, color string) string {
	// outputStr := "\033[38;5;"
	p := termenv.ColorProfile()
	return termenv.String(text).Foreground(p.Color(color)).String()
	// outputStr := "\033[3"
	// outbuf.Write(strconv.AppendUint(intbuf, uint64(a-1), 10))
	// attr := strings.Split(color, ",")
	// bold := false
	// col := attr[0]
	// if attr[0] == "bold" {
	// 	bold = true
	// } else if (len(attr) > 1 && attr[1] == "bold") {
	// 	bold = true
	// }
	// switch color {
	// 	case ""
	// }
	// outputStr += "m"
	// return outputStr
}
