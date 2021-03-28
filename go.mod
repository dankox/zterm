module github.com/dankox/zterm

go 1.15

require (
	github.com/alecthomas/chroma v0.8.1
	github.com/awesome-gocui/gocui v1.0.0-beta-3
	github.com/melbahja/goph v1.1.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/muesli/termenv v0.7.4
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.1
	golang.org/x/crypto v0.0.0-20201117144127-c1f2f97bffc9
	golang.org/x/image v0.0.0-20190802002840-cff245a6509b
)

// replace github.com/awesome-gocui/gocui => github.com/dankox/gocui v0.6.1-0.20201110211249-ab1c2311e43d
// replace github.com/awesome-gocui/gocui => ../gocui
