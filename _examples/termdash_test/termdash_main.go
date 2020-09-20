// Doesn't work on Windows terminal (I guess specifically MinTTY)
// spaces are not populated (not sure why)
// everything is put together, any space is left out

package main

import (
	"context"
	"log"

	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/terminal/terminalapi"

	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/termbox"
	"github.com/mum4k/termdash/widgets/text"
)

func main() {
	tb, err := termbox.New()
	if err != nil {
		log.Fatalf("failed to initialized termbox: %v", err)
	}
	defer tb.Close()

	ctx, cancel := context.WithCancel(context.Background())
	txt, err := text.New()
	if err != nil {
		panic(err)
	}
	if err := txt.Write("Hello World"); err != nil {
		panic(err)
	}

	c, err := container.New(tb, container.Border(linestyle.Double), container.BorderTitle("zMonitor"), container.PlaceWidget(txt))
	if err != nil {
		panic(err)
	}

	quitter := func(k *terminalapi.Keyboard) {
		if k.Key == 'q' || k.Key == 'Q' {
			cancel()
		}
	}

	if err := termdash.Run(ctx, tb, c, termdash.KeyboardSubscriber(quitter)); err != nil {
		panic(err)
	}
}
