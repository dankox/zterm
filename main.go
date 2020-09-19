package main

import (
	"log"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

func main() {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	p := widgets.NewParagraph()
	p.Title = "zMonitor"
	p.TitleStyle.Fg = ui.ColorGreen
	p.Text = "[Hello](fg:red) World"
	p.TextStyle = ui.NewStyle(ui.ColorYellow)
	p.SetRect(0, 0, 25, 5)
	p.BorderStyle.Fg = ui.ColorGreen
	// p.BorderStyle.Modifier = ui.ModifierReverse

	ui.Render(p)

	for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
			break
		}
	}
}
