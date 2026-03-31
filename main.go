package main

import "github.com/dev-sre-toolset/dev-assist/cmd"

// version is injected at build time via -ldflags "-X main.version=x.y.z"
var version = "dev"

func main() {
	cmd.Execute(version)
}
