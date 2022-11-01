package main

import (
	"github.com/kinvin/kd/cmd"
)

const Version = "0.1"

func main() {
	cmd.Version = Version
	cmd.Execute()
}
