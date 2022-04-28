package repo

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	ggit "github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/ui/common"
	"github.com/charmbracelet/soft-serve/ui/components/code"
	"github.com/charmbracelet/soft-serve/ui/components/selector"
	"github.com/charmbracelet/soft-serve/ui/git"
)

type filesView int

const (
	filesViewFiles filesView = iota
	filesViewContent
)

var (
	errNoFileSelected = errors.New("no file selected")
	errBinaryFile     = errors.New("binary file")
	errFileTooLarge   = errors.New("file is too large")
	errInvalidFile    = errors.New("invalid file")
)

// FileItemsMsg is a message that contains a list of files.
type FileItemsMsg []selector.IdentifiableItem

// FileContentMsg is a message that contains the content of a file.
type FileContentMsg struct {
	content string
	ext     string
}

// Files is the model for the files view.
type Files struct {
	common         common.Common
	selector       *selector.Selector
	ref            *ggit.Reference
	activeView     filesView
	repo           git.GitRepo
	code           *code.Code
	path           string
	currentItem    *FileItem
	currentContent FileContentMsg
	lastSelected   []int
}

// NewFiles creates a new files model.
func NewFiles(common common.Common) *Files {
	f := &Files{
		common:       common,
		code:         code.New(common, "", ""),
		activeView:   filesViewFiles,
		lastSelected: make([]int, 0),
	}
	selector := selector.New(common, []selector.IdentifiableItem{}, FileItemDelegate{common.Styles})
	selector.SetShowFilter(false)
	selector.SetShowHelp(false)
	selector.SetShowPagination(false)
	selector.SetShowStatusBar(false)
	selector.SetShowTitle(false)
	selector.SetFilteringEnabled(false)
	selector.DisableQuitKeybindings()
	selector.KeyMap.NextPage = common.KeyMap.NextPage
	selector.KeyMap.PrevPage = common.KeyMap.PrevPage
	f.selector = selector
	return f
}

// SetSize implements common.Component.
func (f *Files) SetSize(width, height int) {
	f.common.SetSize(width, height)
	f.selector.SetSize(width, height)
	f.code.SetSize(width, height)
}

// Init implements tea.Model.
func (f *Files) Init() tea.Cmd {
	f.path = ""
	f.currentItem = nil
	f.activeView = filesViewFiles
	f.lastSelected = make([]int, 0)
	return f.updateFilesCmd
}

// Update implements tea.Model.
func (f *Files) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case RepoMsg:
		f.selector.Select(0)
		f.repo = git.GitRepo(msg)
		cmds = append(cmds, f.Init())
	case RefMsg:
		f.ref = msg
		cmds = append(cmds, f.Init())
	case FileItemsMsg:
		cmds = append(cmds,
			f.selector.SetItems(msg),
			updateStatusBarCmd,
		)
	case FileContentMsg:
		f.activeView = filesViewContent
		f.currentContent = msg
		f.code.SetContent(msg.content, msg.ext)
		f.code.GotoTop()
		cmds = append(cmds, updateStatusBarCmd)
	case selector.SelectMsg:
		switch sel := msg.IdentifiableItem.(type) {
		case FileItem:
			f.currentItem = &sel
			f.path = filepath.Join(f.path, sel.entry.Name())
			log.Printf("selected index %d", f.selector.Index())
			if sel.entry.IsTree() {
				cmds = append(cmds, f.selectTreeCmd)
			} else {
				cmds = append(cmds, f.selectFileCmd)
			}
		}
	case tea.KeyMsg:
		switch f.activeView {
		case filesViewFiles:
			switch msg.String() {
			case "l", "right":
				cmds = append(cmds, f.selector.SelectItem)
			case "h", "left":
				cmds = append(cmds, f.deselectItemCmd)
			}
		case filesViewContent:
			switch msg.String() {
			case "h", "left":
				cmds = append(cmds, f.deselectItemCmd)
			}
		}
	case tea.WindowSizeMsg:
		switch f.activeView {
		case filesViewFiles:
			if f.repo != nil {
				cmds = append(cmds, f.updateFilesCmd)
			}
		case filesViewContent:
			if f.currentContent.content != "" {
				m, cmd := f.code.Update(msg)
				f.code = m.(*code.Code)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
	}
	switch f.activeView {
	case filesViewFiles:
		m, cmd := f.selector.Update(msg)
		f.selector = m.(*selector.Selector)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case filesViewContent:
		m, cmd := f.code.Update(msg)
		f.code = m.(*code.Code)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return f, tea.Batch(cmds...)
}

// View implements tea.Model.
func (f *Files) View() string {
	switch f.activeView {
	case filesViewFiles:
		return f.selector.View()
	case filesViewContent:
		return f.code.View()
	default:
		return ""
	}
}

// StatusBarValue returns the status bar value.
func (f *Files) StatusBarValue() string {
	p := f.path
	if p == "." {
		return ""
	}
	return p
}

// StatusBarInfo returns the status bar info.
func (f *Files) StatusBarInfo() string {
	switch f.activeView {
	case filesViewFiles:
		return fmt.Sprintf("%d/%d", f.selector.Index()+1, len(f.selector.VisibleItems()))
	case filesViewContent:
		return fmt.Sprintf("%.f%%", f.code.ScrollPercent()*100)
	default:
		return ""
	}
}

func (f *Files) updateFilesCmd() tea.Msg {
	files := make([]selector.IdentifiableItem, 0)
	dirs := make([]selector.IdentifiableItem, 0)
	t, err := f.repo.Tree(f.ref, f.path)
	if err != nil {
		return common.ErrorMsg(err)
	}
	ents, err := t.Entries()
	if err != nil {
		return common.ErrorMsg(err)
	}
	ents.Sort()
	for _, e := range ents {
		if e.IsTree() {
			dirs = append(dirs, FileItem{e})
		} else {
			files = append(files, FileItem{e})
		}
	}
	return FileItemsMsg(append(dirs, files...))
}

func (f *Files) selectTreeCmd() tea.Msg {
	if f.currentItem != nil && f.currentItem.entry.IsTree() {
		f.lastSelected = append(f.lastSelected, f.selector.Index())
		f.selector.Select(0)
		return f.updateFilesCmd()
	}
	return common.ErrorMsg(errNoFileSelected)
}

func (f *Files) selectFileCmd() tea.Msg {
	i := f.currentItem
	if i != nil && !i.entry.IsTree() {
		fi := i.entry.File()
		if i.Mode().IsDir() || f == nil {
			return common.ErrorMsg(errInvalidFile)
		}
		bin, err := fi.IsBinary()
		if err != nil {
			f.path = filepath.Dir(f.path)
			return common.ErrorMsg(err)
		}
		if bin {
			f.path = filepath.Dir(f.path)
			return common.ErrorMsg(errBinaryFile)
		}
		c, err := fi.Bytes()
		if err != nil {
			f.path = filepath.Dir(f.path)
			return common.ErrorMsg(err)
		}
		f.lastSelected = append(f.lastSelected, f.selector.Index())
		return FileContentMsg{string(c), i.entry.Name()}
	}
	return common.ErrorMsg(errNoFileSelected)
}

func (f *Files) deselectItemCmd() tea.Msg {
	f.path = filepath.Dir(f.path)
	f.activeView = filesViewFiles
	msg := f.updateFilesCmd()
	index := 0
	if len(f.lastSelected) > 0 {
		index = f.lastSelected[len(f.lastSelected)-1]
		f.lastSelected = f.lastSelected[:len(f.lastSelected)-1]
	}
	log.Printf("deselect %d", index)
	f.selector.Select(index)
	return msg
}
