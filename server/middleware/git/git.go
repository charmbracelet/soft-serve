package git

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"smoothie/server/middleware"
	"strings"

	"github.com/gliderlabs/ssh"
)

func Middleware(repoDir, authorizedKeys, authorizedKeysFile string) middleware.Middleware {
	authedKeys := make([]ssh.PublicKey, 0)
	hasAuth, err := fileExists(authorizedKeysFile)
	if err != nil {
		log.Fatal(err)
	}
	if hasAuth || authorizedKeys != "" {
		var scanner *bufio.Scanner
		if authorizedKeys == "" {
			log.Printf("Importing authorized keys from file: %s", authorizedKeysFile)
			f, err := os.Open(authorizedKeysFile)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			scanner = bufio.NewScanner(f)
		} else {
			log.Printf("Importing authorized keys from environment")
			scanner = bufio.NewScanner(strings.NewReader(authorizedKeys))
		}
		for scanner.Scan() {
			pt := scanner.Text()
			log.Printf("Adding authorized key: %s", pt)
			pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pt))
			if err != nil {
				log.Fatal(err)
			}
			authedKeys = append(authedKeys, pk)
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			cmd := s.Command()
			if len(cmd) == 2 {
				switch cmd[0] {
				case "git-upload-pack", "git-upload-archive", "git-receive-pack":
					if hasAuth && cmd[0] == "git-receive-pack" {
						authed := false
						for _, pk := range authedKeys {
							if ssh.KeysEqual(pk, s.PublicKey()) {
								authed = true
							}
						}
						if !authed {
							fatalGit(s, fmt.Errorf("you are not authorized to do this"))
							break
						}
					}
					r := cmd[1]
					rp := fmt.Sprintf("%s%s", repoDir, r)
					ctx := s.Context()
					err := ensureRepo(ctx, repoDir, r)
					if err != nil {
						fatalGit(s, err)
						break
					}
					c := exec.CommandContext(ctx, cmd[0], rp)
					c.Dir = "./"
					c.Stdout = s
					c.Stdin = s
					err = c.Run()
					if err != nil {
						fatalGit(s, err)
						break
					}
				}
			}
			sh(s)
		}
	}
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func fatalGit(s ssh.Session, err error) {
	// hex length includes 4 byte length prefix and ending newline
	msg := err.Error()
	pktLine := fmt.Sprintf("%04x%s\n", len(msg)+5, msg)
	_, _ = s.Write([]byte(pktLine))
	s.Exit(1)
}

func ensureRepo(ctx context.Context, dir string, repo string) error {
	exists, err := fileExists(dir)
	if err != nil {
		return err
	}
	if !exists {
		err = os.MkdirAll(dir, os.ModeDir|os.FileMode(0700))
		if err != nil {
			return err
		}
	}
	rp := fmt.Sprintf("%s%s", dir, repo)
	exists, err = fileExists(rp)
	if err != nil {
		return err
	}
	if !exists {
		c := exec.CommandContext(ctx, "git", "init", "--bare", rp)
		err = c.Run()
		if err != nil {
			return err
		}
	}
	return nil
}
