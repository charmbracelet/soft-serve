//go:build mango
// +build mango

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/muesli/mango"
	mcobra "github.com/muesli/mango-cobra"
	"github.com/muesli/roff"
)

func init() {
	manPage := mcobra.NewManPage(1, rootCmd).
		WithSection("Copyright", "(C) 2021-2022 Charmbracelet, Inc.\n"+
			"Released under MIT license.")

	fmt.Println(manPage.Build(roff.NewDocument()))
	os.Exit(0)
}
