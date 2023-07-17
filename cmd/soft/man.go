package main

import (
	"fmt"

	mcobra "github.com/muesli/mango-cobra"
	"github.com/muesli/roff"
	"github.com/spf13/cobra"
)

var manCmd = &cobra.Command{
	Use:    "man",
	Short:  "Generate man pages",
	Args:   cobra.NoArgs,
	Hidden: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		manPage, err := mcobra.NewManPage(1, rootCmd) //.
		if err != nil {
			return err
		}

		manPage = manPage.WithSection("Copyright", "(C) 2021-2023 Charmbracelet, Inc.\n"+
			"Released under MIT license.")
		fmt.Println(manPage.Build(roff.NewDocument()))
		return nil
	},
}
