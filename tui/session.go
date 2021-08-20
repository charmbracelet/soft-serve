package tui

import (
	"encoding/json"
	"fmt"
	"log"
	"smoothie/git"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gliderlabs/ssh"
)

func SessionHandler(reposPath string, repoPoll time.Duration) func(ssh.Session) (tea.Model, []tea.ProgramOption) {
	rs := git.NewRepoSource(reposPath)
	err := createDefaultConfigRepo(rs)
	if err != nil {
		if err != nil {
			log.Fatalf("cannot create config repo: %s", err)
		}
	}
	appCfg, err := loadConfig(rs)
	if err != nil {
		if err != nil {
			log.Printf("cannot load config: %s", err)
		}
	}
	go func() {
		for {
			time.Sleep(repoPoll)
			err := rs.LoadRepos()
			if err != nil {
				log.Printf("cannot load repos: %s", err)
				continue
			}
			cfg, err := loadConfig(rs)
			if err != nil {
				if err != nil {
					log.Printf("cannot load config: %s", err)
					continue
				}
			}
			appCfg = cfg
		}
	}()

	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		cmd := s.Command()
		cfg := &SessionConfig{}
		switch len(cmd) {
		case 0:
			cfg.InitialRepo = ""
		case 1:
			cfg.InitialRepo = cmd[0]
		default:
			return nil, nil
		}
		pty, _, active := s.Pty()
		if !active {
			fmt.Println("not active")
			return nil, nil
		}
		cfg.Width = pty.Window.Width
		cfg.Height = pty.Window.Height
		return NewBubble(appCfg, cfg), []tea.ProgramOption{tea.WithAltScreen()}
	}
}

func loadConfig(rs *git.RepoSource) (*Config, error) {
	cfg := &Config{}
	cfg.RepoSource = rs
	cr, err := rs.GetRepo("config")
	if err != nil {
		return nil, fmt.Errorf("cannot load config repo: %s", err)
	}
	cs, err := cr.LatestFile("config.json")
	if err != nil {
		return nil, fmt.Errorf("cannot load config.json: %s", err)
	}
	err = json.Unmarshal([]byte(cs), cfg)
	if err != nil {
		return nil, fmt.Errorf("bad json in config.json: %s", err)
	}
	return cfg, nil
}
