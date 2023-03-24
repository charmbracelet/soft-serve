package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/spf13/cobra"
)

// listCommand returns a command that list file or directory at path.
func listCommand() *cobra.Command {
	listCmd := &cobra.Command{
		Use:               "list PATH",
		Aliases:           []string{"ls"},
		Short:             "List files at repository.",
		Args:              cobra.RangeArgs(0, 1),
		PersistentPreRunE: checkIfReadable,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, s := fromContext(cmd)
			rn := ""
			path := ""
			ps := []string{}
			if len(args) > 0 {
				path = filepath.Clean(args[0])
				ps = strings.Split(path, "/")
				rn = strings.TrimSuffix(ps[0], ".git")
				auth := cfg.Access.AccessLevel(rn, s.PublicKey())
				if auth < backend.ReadOnlyAccess {
					return ErrUnauthorized
				}
			}
			if path == "" || path == "." || path == "/" {
				repos, err := cfg.Backend.Repositories()
				if err != nil {
					return err
				}
				for _, r := range repos {
					if cfg.Access.AccessLevel(r.Name(), s.PublicKey()) >= backend.ReadOnlyAccess {
						cmd.Println(r.Name())
					}
				}
				return nil
			}
			rr, err := cfg.Backend.Repository(rn)
			if err != nil {
				return err
			}
			r, err := rr.Open()
			if err != nil {
				return err
			}
			head, err := r.HEAD()
			if err != nil {
				if bs, err := r.Branches(); err != nil && len(bs) == 0 {
					return fmt.Errorf("repository is empty")
				}
				return err
			}
			tree, err := r.TreePath(head, "")
			if err != nil {
				return err
			}
			subpath := strings.Join(ps[1:], "/")
			ents := git.Entries{}
			te, err := tree.TreeEntry(subpath)
			if err == git.ErrRevisionNotExist {
				return ErrFileNotFound
			}
			if err != nil {
				return err
			}
			if te.Type() == "tree" {
				tree, err = tree.SubTree(subpath)
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
			ents.Sort()
			for _, ent := range ents {
				cmd.Printf("%s\t%d\t %s\n", ent.Mode(), ent.Size(), ent.Name())
			}
			return nil
		},
	}
	return listCmd
}
