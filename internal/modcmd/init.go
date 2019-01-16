// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// go mod init

package modcmd

import (
	"github.com/sirkon/goproxy/internal/base"
	"github.com/sirkon/goproxy/internal/modload"
	"os"
)

var cmdInit = &base.Command{
	UsageLine: "go mod init [module]",
	Short:     "initialize new module in current directory",
	Long: `
Init initializes and writes a new go.mod to the current directory,
in effect creating a new module rooted at the current directory.
The file go.mod must not already exist.
If possible, init will guess the module path from import comments
(see 'go help importpath') or from version control configuration.
To override this guess, supply the module path as an argument.
	`,
	Run: runInit,
}

func runInit(cmd *base.Command, args []string) {
	modload.CmdModInit = true
	if len(args) > 1 {
		base.Fatalf("go mod init: too many arguments")
	}
	if len(args) == 1 {
		modload.CmdModModule = args[0]
	}
	if _, err := os.Stat("go.mod"); err == nil {
		base.Fatalf("go mod init: go.mod already exists")
	}
	modload.InitMod() // does all the hard work
}
