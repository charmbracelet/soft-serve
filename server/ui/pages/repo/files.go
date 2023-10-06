package repo

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/alecthomas/chroma/lexers"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/proto"
	"github.com/charmbracelet/soft-serve/server/ui/common"
	"github.com/charmbracelet/soft-serve/server/ui/components/code"
	"github.com/charmbracelet/soft-serve/server/ui/components/selector"
)

type filesView int

const (
	filesViewLoading filesView = iota
	filesViewFiles
	filesViewContent
)

var (
	errNoFileSelected = errors.New("no file selected")
	errBinaryFile     = errors.New("binary file")
	errInvalidFile    = errors.New("invalid file")
)

var (
	lineNo = key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "toggle line numbers"),
	)
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
	ref            *git.Reference
	activeView     filesView
	repo           proto.Repository
	code           *code.Code
	path           string
	currentItem    *FileItem
	currentContent FileContentMsg
	lastSelected   []int
	lineNumber     bool
	spinner        spinner.Model
	cursor         int
}

// NewFiles creates a new files model.
func NewFiles(common common.Common) *Files {
	f := &Files{
		common:       common,
		code:         code.New(common, "", ""),
		activeView:   filesViewLoading,
		lastSelected: make([]int, 0),
		lineNumber:   true,
	}
	selector := selector.New(common, []selector.IdentifiableItem{}, FileItemDelegate{&common})
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
	f.code.SetShowLineNumber(f.lineNumber)
	s := spinner.New(spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(common.Styles.Spinner))
	f.spinner = s
	return f
}

// TabName returns the tab name.
func (f *Files) TabName() string {
	return "Files"
}

// SetSize implements common.Component.
func (f *Files) SetSize(width, height int) {
	f.common.SetSize(width, height)
	f.selector.SetSize(width, height)
	f.code.SetSize(width, height)
}

// ShortHelp implements help.KeyMap.
func (f *Files) ShortHelp() []key.Binding {
	k := f.selector.KeyMap
	switch f.activeView {
	case filesViewFiles:
		copyKey := f.common.KeyMap.Copy
		copyKey.SetHelp("c", "copy name")
		return []key.Binding{
			f.common.KeyMap.SelectItem,
			f.common.KeyMap.BackItem,
			k.CursorUp,
			k.CursorDown,
			copyKey,
		}
	case filesViewContent:
		copyKey := f.common.KeyMap.Copy
		copyKey.SetHelp("c", "copy content")
		b := []key.Binding{
			f.common.KeyMap.UpDown,
			f.common.KeyMap.BackItem,
			copyKey,
		}
		lexer := lexers.Match(f.currentContent.ext)
		lang := ""
		if lexer != nil && lexer.Config() != nil {
			lang = lexer.Config().Name
		}
		if lang != "markdown" {
			b = append(b, lineNo)
		}
		return b
	default:
		return []key.Binding{}
	}
}

// FullHelp implements help.KeyMap.
func (f *Files) FullHelp() [][]key.Binding {
	b := make([][]key.Binding, 0)
	copyKey := f.common.KeyMap.Copy
	switch f.activeView {
	case filesViewFiles:
		copyKey.SetHelp("c", "copy name")
		k := f.selector.KeyMap
		b = append(b, []key.Binding{
			f.common.KeyMap.SelectItem,
			f.common.KeyMap.BackItem,
		})
		b = append(b, [][]key.Binding{
			{
				k.CursorUp,
				k.CursorDown,
				k.NextPage,
				k.PrevPage,
			},
			{
				k.GoToStart,
				k.GoToEnd,
				copyKey,
			},
		}...)
	case filesViewContent:
		copyKey.SetHelp("c", "copy content")
		k := f.code.KeyMap
		b = append(b, []key.Binding{
			f.common.KeyMap.BackItem,
		})
		b = append(b, [][]key.Binding{
			{
				k.PageDown,
				k.PageUp,
				k.HalfPageDown,
				k.HalfPageUp,
			},
		}...)
		lc := []key.Binding{
			k.Down,
			k.Up,
			f.common.KeyMap.GotoTop,
			f.common.KeyMap.GotoBottom,
			copyKey,
		}
		lexer := lexers.Match(f.currentContent.ext)
		lang := ""
		if lexer != nil && lexer.Config() != nil {
			lang = lexer.Config().Name
		}
		if lang != "markdown" {
			lc = append(lc, lineNo)
		}
		b = append(b, lc)
	}
	return b
}

// Init implements tea.Model.
func (f *Files) Init() tea.Cmd {
	f.path = ""
	f.currentItem = nil
	f.activeView = filesViewLoading
	f.lastSelected = make([]int, 0)
	return tea.Batch(f.spinner.Tick, f.updateFilesCmd)
}

// Update implements tea.Model.
func (f *Files) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case RepoMsg:
		f.repo = msg
	case RefMsg:
		f.ref = msg
		f.selector.Select(0)
		cmds = append(cmds, f.Init())
	case FileItemsMsg:
		cmds = append(cmds,
			f.selector.SetItems(msg),
		)
		f.activeView = filesViewFiles
		if f.cursor >= 0 {
			f.selector.Select(f.cursor)
			f.cursor = -1
		}
	case FileContentMsg:
		f.activeView = filesViewContent
		f.currentContent = msg
		cmds = append(cmds,
			f.code.SetContent(msg.content, msg.ext),
		)
		f.code.GotoTop()
	case selector.SelectMsg:
		switch sel := msg.IdentifiableItem.(type) {
		case FileItem:
			f.currentItem = &sel
			f.path = filepath.Join(f.path, sel.entry.Name())
			if sel.entry.IsTree() {
				cmds = append(cmds, f.selectTreeCmd)
			} else {
				cmds = append(cmds, f.selectFileCmd)
			}
		}
	case tea.KeyMsg:
		switch f.activeView {
		case filesViewFiles:
			switch {
			case key.Matches(msg, f.common.KeyMap.SelectItem):
				cmds = append(cmds, f.selector.SelectItemCmd)
			case key.Matches(msg, f.common.KeyMap.BackItem):
				cmds = append(cmds, f.deselectItemCmd())
			}
		case filesViewContent:
			switch {
			case key.Matches(msg, f.common.KeyMap.BackItem):
				cmds = append(cmds, f.deselectItemCmd())
			case key.Matches(msg, f.common.KeyMap.Copy):
				cmds = append(cmds, copyCmd(f.currentContent.content, "File contents copied to clipboard"))
			case key.Matches(msg, lineNo):
				f.lineNumber = !f.lineNumber
				f.code.SetShowLineNumber(f.lineNumber)
				cmds = append(cmds, f.code.SetContent(f.currentContent.content, f.currentContent.ext))
			}
		}
	case tea.WindowSizeMsg:
		f.SetSize(msg.Width, msg.Height)
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
	case EmptyRepoMsg:
		f.ref = nil
		f.path = ""
		f.currentItem = nil
		f.activeView = filesViewFiles
		f.lastSelected = make([]int, 0)
		f.selector.Select(0)
		cmds = append(cmds, f.setItems([]selector.IdentifiableItem{}))
	case spinner.TickMsg:
		if f.activeView == filesViewLoading && f.spinner.ID() == msg.ID {
			s, cmd := f.spinner.Update(msg)
			f.spinner = s
			if cmd != nil {
				cmds = append(cmds, cmd)
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
	case filesViewLoading:
		return renderLoading(f.common, f.spinner)
	case filesViewFiles:
		return f.selector.View()
	case filesViewContent:
		return f.code.View()
	default:
		return ""
	}
}

// SpinnerID implements common.TabComponent.
func (f *Files) SpinnerID() int {
	return f.spinner.ID()
}

// StatusBarValue returns the status bar value.
func (f *Files) StatusBarValue() string {
	p := f.path
	if p == "." || p == "" {
		return " "
	}
	return p
}

// StatusBarInfo returns the status bar info.
func (f *Files) StatusBarInfo() string {
	switch f.activeView {
	case filesViewFiles:
		return fmt.Sprintf("# %d/%d", f.selector.Index()+1, len(f.selector.VisibleItems()))
	case filesViewContent:
		return fmt.Sprintf("☰ %.f%%", f.code.ScrollPercent()*100)
	default:
		return ""
	}
}

func (f *Files) updateFilesCmd() tea.Msg {
	files := make([]selector.IdentifiableItem, 0)
	dirs := make([]selector.IdentifiableItem, 0)
	if f.ref == nil {
		return nil
	}
	r, err := f.repo.Open()
	if err != nil {
		return common.ErrorCmd(err)
	}
	path := f.path
	ref := f.ref
	t, err := r.TreePath(ref, path)
	if err != nil {
		return common.ErrorCmd(err)
	}
	ents, err := t.Entries()
	if err != nil {
		return common.ErrorCmd(err)
	}
	ents.Sort()
	for _, e := range ents {
		if e.IsTree() {
			dirs = append(dirs, FileItem{entry: e})
		} else {
			files = append(files, FileItem{entry: e})
		}
	}
	return FileItemsMsg(append(dirs, files...))
}

func (f *Files) selectTreeCmd() tea.Msg {
	if f.currentItem != nil && f.currentItem.entry.IsTree() {
		f.lastSelected = append(f.lastSelected, f.selector.Index())
		f.cursor = 0
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

		var err error
		var bin bool

		r, err := f.repo.Open()
		if err == nil {
			attrs, err := r.CheckAttributes(f.ref, fi.Path())
			if err == nil {
				for _, attr := range attrs {
					if (attr.Name == "binary" && attr.Value == "set") ||
						(attr.Name == "text" && attr.Value == "unset") {
						bin = true
						break
					}
				}
			}
		}

		if !bin {
			bin, err = fi.IsBinary()
			if err != nil {
				f.path = filepath.Dir(f.path)
				return common.ErrorMsg(err)
			}
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

func (f *Files) deselectItemCmd() tea.Cmd {
	f.path = filepath.Dir(f.path)
	index := 0
	if len(f.lastSelected) > 0 {
		index = f.lastSelected[len(f.lastSelected)-1]
		f.lastSelected = f.lastSelected[:len(f.lastSelected)-1]
	}
	f.cursor = index
	f.activeView = filesViewFiles
	return f.updateFilesCmd
}

func (f *Files) setItems(items []selector.IdentifiableItem) tea.Cmd {
	return func() tea.Msg {
		return FileItemsMsg(items)
	}
}
