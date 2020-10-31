module github.com/dankox/zterm

go 1.15

require (
	github.com/awesome-gocui/gocui v0.6.1-0.20200808231733-d0eae9ef0497
	github.com/mitchellh/go-homedir v1.1.0
	github.com/muesli/termenv v0.7.4
	github.com/nsf/termbox-go v0.0.0-20200418040025-38ba6e5628f1 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.1
	golang.org/x/crypto v0.0.0-20190605123033-f99c8df09eb5
	golang.org/x/image v0.0.0-20190802002840-cff245a6509b
)

replace github.com/awesome-gocui/gocui => github.com/dankox/gocui v0.6.1-0.20201028144745-cb1cfb0a01a1
