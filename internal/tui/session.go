package tui

import (
	"fmt"
	"io/fs"
	"log"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft/internal/config"
	"github.com/fsnotify/fsnotify"
	"github.com/gliderlabs/ssh"
)

func SessionHandler(cfg *config.Config) func(ssh.Session) (tea.Model, []tea.ProgramOption) {
	go reloadOnChange(cfg)
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		cmd := s.Command()
		scfg := &SessionConfig{Session: s}
		switch len(cmd) {
		case 0:
			scfg.InitialRepo = ""
		case 1:
			scfg.InitialRepo = cmd[0]
		default:
			return nil, nil
		}
		pty, _, active := s.Pty()
		if !active {
			fmt.Println("not active")
			return nil, nil
		}
		scfg.Width = pty.Window.Width
		scfg.Height = pty.Window.Height
		if cfg.Cfg.Callbacks != nil {
			cfg.Cfg.Callbacks.Tui("view")
		}
		return NewBubble(cfg, scfg), []tea.ProgramOption{tea.WithAltScreen()}
	}
}

func reloadOnChange(cfg *config.Config) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	if err = filepath.WalkDir(filepath.Join(cfg.Cfg.RepoPath, "config"), func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return watcher.Add(path)
		}

		return nil
	}); err != nil {
		log.Printf("watcher error: %s", err)
		return
	}

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					cfg.Reload()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					log.Printf("watcher error: %s", err)
					return
				}
			}
		}
	}()

	<-done
}
