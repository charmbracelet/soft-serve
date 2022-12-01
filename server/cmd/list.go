package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/proto"
	"github.com/spf13/cobra"
)

// ListCommand returns a command that list file or directory at path.
func ListCommand() *cobra.Command {
	lsCmd := &cobra.Command{
		Use:     "ls PATH",
		Aliases: []string{"list"},
		Short:   "List file or directory at path.",
		Args:    cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, s := fromContext(cmd)
			rn := ""
			path := ""
			ps := []string{}
			if len(args) > 0 {
				path = filepath.Clean(args[0])
				ps = strings.Split(path, "/")
				rn = strings.TrimSuffix(ps[0], ".git")
				auth := cfg.AuthRepo(rn, s.PublicKey())
				if auth < proto.ReadOnlyAccess {
					return ErrUnauthorized
				}
			}
			if path == "" || path == "." || path == "/" {
				repos, err := cfg.ListRepos()
				if err != nil {
					return err
				}
				for _, r := range repos {
					if cfg.AuthRepo(r.Name(), s.PublicKey()) >= proto.ReadOnlyAccess {
						fmt.Fprintln(s, r.Name())
					}
				}
				return nil
			}
			r, err := cfg.Open(rn)
			if err != nil {
				return err
			}
			head, err := r.Repository().HEAD()
			if err != nil {
				return err
			}
			tree, err := r.Repository().TreePath(head, "")
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
				fmt.Fprintf(s, "%s\t%d\t %s\n", ent.Mode(), ent.Size(), ent.Name())
			}
			return nil
		},
	}
	return lsCmd
}
