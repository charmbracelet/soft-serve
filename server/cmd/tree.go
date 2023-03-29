package cmd

import (
	"fmt"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

// treeCommand returns a command that list file or directory at path.
func treeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "tree REPOSITORY [REFERENCE] [PATH]",
		Short:             "Print repository tree at path",
		Args:              cobra.RangeArgs(1, 3),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := fromContext(cmd)
			rn := args[0]
			path := ""
			ref := ""
			switch len(args) {
			case 2:
				path = args[1]
			case 3:
				ref = args[1]
				path = args[2]
			}
			rr, err := cfg.Backend.Repository(rn)
			if err != nil {
				return err
			}

			r, err := rr.Open()
			if err != nil {
				return err
			}

			if ref == "" {
				head, err := r.HEAD()
				if err != nil {
					if bs, err := r.Branches(); err != nil && len(bs) == 0 {
						return fmt.Errorf("repository is empty")
					}
					return err
				}

				ref = head.Hash.String()
			}

			tree, err := r.LsTree(ref)
			if err != nil {
				return err
			}

			ents := git.Entries{}
			if path != "" && path != "/" {
				te, err := tree.TreeEntry(path)
				if err == git.ErrRevisionNotExist {
					return ErrFileNotFound
				}
				if err != nil {
					return err
				}
				if te.Type() == "tree" {
					tree, err = tree.SubTree(path)
					if err != nil {
						return err
					}
					ents, err = tree.Entries()
					if err != nil {
						return err
					}
				} else {
					ents = append(ents, te)
				}
			} else {
				ents, err = tree.Entries()
				if err != nil {
					return err
				}
			}
			ents.Sort()
			for _, ent := range ents {
				size := ent.Size()
				ssize := ""
				if size == 0 {
					ssize = "-"
				} else {
					ssize = humanize.Bytes(uint64(size))
				}
				cmd.Printf("%s\t%s\t %s\n", ent.Mode(), ssize, ent.Name())
			}
			return nil
		},
	}
	return cmd
}
