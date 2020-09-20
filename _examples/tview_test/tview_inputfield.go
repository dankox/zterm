// Works in a weird way!
// Directly running inside of the library, it works `go run demos/inputfields/main.go`
// Here, it doesn't display stuff properly... not sure why

package main

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()
	inputField := tview.NewInputField().
		SetLabel("Enter a number: ").
		SetPlaceholder("E.g. 1234").
		SetFieldWidth(10).
		SetAcceptanceFunc(tview.InputFieldInteger).
		SetDoneFunc(func(key tcell.Key) {
			app.Stop()
		})
	if err := app.SetRoot(inputField, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
