//go:build mango
// +build mango

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/muesli/mango"
	"github.com/muesli/mango/mflag"
	"github.com/muesli/roff"
)

func init() {
	manPage := mango.NewManPage(1, "soft", "A self-hostable Git server for the command line").
		WithLongDescription("Soft Serve is a self-hostable Git server for the command line.").
		WithSection("Copyright", "(C) 2021-2022 Charmbracelet, Inc.\n"+
			"Released under MIT license.")

	flag.VisitAll(mflag.FlagVisitor(manPage))
	fmt.Println(manPage.Build(roff.NewDocument()))
	os.Exit(0)
}
