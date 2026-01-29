module github.com/yourorg/arc-tmux

go 1.23

require (
	github.com/mattn/go-isatty v0.0.20
	github.com/spf13/cobra v1.8.1
	github.com/yourorg/arc-sdk v0.1.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/sys v0.22.0 // indirect
)

replace github.com/yourorg/arc-sdk => ../arc-sdk
