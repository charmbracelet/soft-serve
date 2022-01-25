package tree

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/lexers"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/types"
	vp "github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/viewport"
	"github.com/charmbracelet/soft-serve/internal/tui/style"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/muesli/reflow/wrap"
)

type pageView int

const (
	treeView pageView = iota
	fileView
)

type item struct {
	*object.TreeEntry
	*object.File
}

func (i item) Name() string {
	return i.TreeEntry.Name
}

func (i item) Mode() filemode.FileMode {
	return i.TreeEntry.Mode
}

func (i item) FilterValue() string { return i.Name() }

type items []item

func (cl items) Len() int      { return len(cl) }
func (cl items) Swap(i, j int) { cl[i], cl[j] = cl[j], cl[i] }
func (cl items) Less(i, j int) bool {
	if cl[i].Mode() == filemode.Dir && cl[j].Mode() == filemode.Dir {
		return cl[i].Name() < cl[j].Name()
	} else if cl[i].Mode() == filemode.Dir {
		return true
	} else if cl[j].Mode() == filemode.Dir {
		return false
	} else {
		return cl[i].Name() < cl[j].Name()
	}
}

type itemDelegate struct {
	style *style.Styles
}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	name := i.Name()
	if i.Mode() == filemode.Dir {
		name = d.style.TreeFileDir.Render(name)
	}
	size := ""
	if i.File != nil {
		size = fmt.Sprintf("%d", i.File.Size)
	}
	var cs lipgloss.Style
	mode, _ := i.Mode().ToOSFileMode()
	if index == m.Index() {
		cs = d.style.LogItemActive
		fmt.Fprint(w, d.style.LogItemSelector.Render(">"))
	} else {
		cs = d.style.LogItemInactive
		fmt.Fprint(w, d.style.LogItemSelector.Render(" "))
	}
	leftMargin := d.style.LogItemSelector.GetMarginLeft() +
		d.style.LogItemSelector.GetWidth() +
		d.style.TreeFileMode.GetMarginLeft() +
		d.style.TreeFileMode.GetWidth() +
		cs.GetMarginLeft()
	rightMargin := d.style.TreeFileSize.GetMarginLeft() + len(size)
	name = types.TruncateString(name, m.Width()-leftMargin-rightMargin, "…")
	gap := m.Width() - len(name) - leftMargin - rightMargin
	if gap < 0 {
		gap = 0
	}
	fmt.Fprint(w, d.style.TreeFileMode.Render(mode.String())+
		cs.Render(name)+
		strings.Repeat(" ", gap)+
		d.style.TreeFileSize.Render(size))
}

type Bubble struct {
	repo         types.Repo
	list         list.Model
	style        *style.Styles
	width        int
	widthMargin  int
	height       int
	heightMargin int
	path         string
	pageView     pageView
	fileViewport *vp.ViewportBubble
	lastSelected []int
}

func NewBubble(repo types.Repo, style *style.Styles, width, widthMargin, height, heightMargin int) *Bubble {
	l := list.NewModel([]list.Item{}, itemDelegate{style}, width-widthMargin, height-heightMargin)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	b := &Bubble{
		fileViewport: &vp.ViewportBubble{
			Viewport: &viewport.Model{},
		},
		repo:         repo,
		style:        style,
		width:        width,
		height:       height,
		widthMargin:  widthMargin,
		heightMargin: heightMargin,
		list:         l,
		path:         "",
		pageView:     treeView,
		lastSelected: []int{},
	}
	b.SetSize(width, height)
	return b
}

func (b *Bubble) Init() tea.Cmd {
	return nil
}

func (b *Bubble) SetSize(width, height int) {
	b.width = width
	b.height = height
	b.fileViewport.Viewport.Width = width - b.widthMargin
	b.fileViewport.Viewport.Height = height - b.heightMargin
	b.list.SetSize(width-b.widthMargin, height-b.heightMargin)
}

func (b *Bubble) Help() []types.HelpEntry {
	return []types.HelpEntry{
		{"enter", "select"},
		{"esc", "back"},
	}
}

func (b *Bubble) UpdateItems() tea.Cmd {
	its := make(items, 0)
	t, err := b.repo.Tree(b.path)
	if err != nil {
		return nil
	}
	tw := object.NewTreeWalker(t, false, map[plumbing.Hash]bool{})
	defer tw.Close()
	for {
		_, e, err := tw.Next()
		if err != nil {
			break
		}
		i := item{
			TreeEntry: &e,
		}
		if e.Mode.IsFile() {
			if f, err := t.TreeEntryFile(&e); err == nil {
				i.File = f
			}
		}
		its = append(its, i)
	}
	sort.Sort(its)
	itt := make([]list.Item, len(its))
	for i, it := range its {
		itt[i] = it
	}
	cmd := b.list.SetItems(itt)
	return cmd
}

func (b *Bubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.SetSize(msg.Width, msg.Height)
		if b.pageView == fileView {
			item := b.list.SelectedItem().(item)
			content := b.currentFileView(item)
			b.fileViewport.Viewport.SetContent(content)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "T":
			b.pageView = treeView
			cmds = append(cmds, b.UpdateItems())
		case "down", "j":
			if b.pageView == treeView {
				b.list.CursorDown()
			}
		case "up", "k":
			if b.pageView == treeView {
				b.list.CursorUp()
			}
		case "enter":
			if b.pageView == treeView {
				index := b.list.Index()
				item := b.list.SelectedItem().(item)
				mode := item.Mode()
				b.path = filepath.Join(b.path, item.Name())
				if mode == filemode.Dir {
					b.lastSelected = append(b.lastSelected, index)
					cmds = append(cmds, b.UpdateItems())
				} else {
					b.lastSelected = append(b.lastSelected, index)
					b.fileViewport.Viewport.SetContent(b.currentFileView(item))
					b.fileViewport.Viewport.GotoTop()
					b.pageView = fileView
				}
				b.list.Select(0)
			}
		case "esc":
			if b.pageView == fileView {
				b.pageView = treeView
			}
			p := filepath.Dir(b.path)
			b.path = p
			cmds = append(cmds, b.UpdateItems())
			index := 0
			if len(b.lastSelected) > 0 {
				index = b.lastSelected[len(b.lastSelected)-1]
				b.lastSelected = b.lastSelected[:len(b.lastSelected)-1]
			}
			b.list.Select(index)
		}
	}
	if b.pageView == fileView {
		rv, cmd := b.fileViewport.Update(msg)
		b.fileViewport = rv.(*vp.ViewportBubble)
		cmds = append(cmds, cmd)
	}
	return b, tea.Batch(cmds...)
}

func (b *Bubble) View() string {
	switch b.pageView {
	case treeView:
		return b.list.View()
	case fileView:
		return b.fileViewport.View()
	default:
		return ""
	}
}

func (b *Bubble) currentFileView(item item) string {
	if !item.Mode().IsFile() || item.File == nil {
		return types.ErrInvalidFile.Error()
	}
	bin, err := item.File.IsBinary()
	if err != nil {
		return err.Error()
	}
	if bin {
		return types.ErrBinaryFile.Error()
	}
	c, err := item.File.Contents()
	if err != nil {
		return err.Error()
	}
	if len(strings.Split(c, "\n")) > types.MaxDiffLines {
		return types.ErrFileTooLarge.Error()
	}
	lexer := lexers.Match(b.path)
	lang := ""
	if lexer != nil && lexer.Config() != nil {
		lang = lexer.Config().Name
	}
	formatter := &gansi.CodeBlockElement{
		Code:     c,
		Language: lang,
	}
	s := strings.Builder{}
	formatter.Render(&s, types.RenderCtx)

	return wrap.String(s.String(), b.width-b.widthMargin)
}
