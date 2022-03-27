package tree

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/soft-serve/internal/tui/style"
	"github.com/charmbracelet/soft-serve/pkg/git"
	"github.com/charmbracelet/soft-serve/pkg/tui/refs"
	"github.com/charmbracelet/soft-serve/pkg/tui/utils"
	vp "github.com/charmbracelet/soft-serve/pkg/tui/viewport"
	"github.com/dustin/go-humanize"
)

type fileMsg struct {
	content string
}

type sessionState int

const (
	treeState sessionState = iota
	fileState
	errorState
)

type item struct {
	entry *git.TreeEntry
}

func (i item) Name() string {
	return i.entry.Name()
}

func (i item) Mode() fs.FileMode {
	return i.entry.Mode()
}

func (i item) FilterValue() string { return i.Name() }

type items []item

func (cl items) Len() int      { return len(cl) }
func (cl items) Swap(i, j int) { cl[i], cl[j] = cl[j], cl[i] }
func (cl items) Less(i, j int) bool {
	if cl[i].entry.IsTree() && cl[j].entry.IsTree() {
		return cl[i].Name() < cl[j].Name()
	} else if cl[i].entry.IsTree() {
		return true
	} else if cl[j].entry.IsTree() {
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
	s := d.style
	i, ok := listItem.(item)
	if !ok {
		return
	}

	name := i.Name()
	size := humanize.Bytes(uint64(i.entry.Size()))
	if i.entry.IsTree() {
		size = ""
		name = s.TreeFileDir.Render(name)
	}
	var cs lipgloss.Style
	mode := i.Mode()
	if index == m.Index() {
		cs = s.TreeItemActive
		fmt.Fprint(w, s.TreeItemSelector.Render(">"))
	} else {
		cs = s.TreeItemInactive
		fmt.Fprint(w, s.TreeItemSelector.Render(" "))
	}
	leftMargin := s.TreeItemSelector.GetMarginLeft() +
		s.TreeItemSelector.GetWidth() +
		s.TreeFileMode.GetMarginLeft() +
		s.TreeFileMode.GetWidth() +
		cs.GetMarginLeft()
	rightMargin := s.TreeFileSize.GetMarginLeft() + lipgloss.Width(size)
	name = utils.TruncateString(name, m.Width()-leftMargin-rightMargin, "…")
	sizeStyle := s.TreeFileSize.Copy().
		Width(m.Width() -
			leftMargin -
			s.TreeFileSize.GetMarginLeft() -
			lipgloss.Width(name)).
		Align(lipgloss.Right)
	fmt.Fprint(w, s.TreeFileMode.Render(mode.String())+
		cs.Render(name)+
		sizeStyle.Render(size))
}

type Bubble struct {
	repo         utils.GitRepo
	list         list.Model
	style        *style.Styles
	width        int
	widthMargin  int
	height       int
	heightMargin int
	path         string
	state        sessionState
	error        utils.ErrMsg
	fileViewport *vp.ViewportBubble
	lastSelected []int
	ref          *git.Reference
}

func NewBubble(repo utils.GitRepo, styles *style.Styles, width, widthMargin, height, heightMargin int) *Bubble {
	l := list.New([]list.Item{}, itemDelegate{styles}, width-widthMargin, height-heightMargin)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowPagination(true)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()
	l.KeyMap.NextPage = utils.NextPage
	l.KeyMap.PrevPage = utils.PrevPage
	l.Styles.NoItems = styles.TreeNoItems
	b := &Bubble{
		fileViewport: &vp.ViewportBubble{
			Viewport: &viewport.Model{},
		},
		repo:         repo,
		style:        styles,
		width:        width,
		height:       height,
		widthMargin:  widthMargin,
		heightMargin: heightMargin,
		list:         l,
		state:        treeState,
	}
	b.SetSize(width, height)
	return b
}

func (b *Bubble) reset() tea.Cmd {
	b.path = ""
	b.state = treeState
	b.lastSelected = make([]int, 0)
	cmd := b.updateItems()
	return cmd
}

func (b *Bubble) Init() tea.Cmd {
	head, err := b.repo.HEAD()
	if err != nil {
		return func() tea.Msg {
			return utils.ErrMsg{Err: err}
		}
	}
	b.ref = head
	return nil
}

func (b *Bubble) SetSize(width, height int) {
	b.width = width
	b.height = height
	b.fileViewport.Viewport.Width = width - b.widthMargin
	b.fileViewport.Viewport.Height = height - b.heightMargin
	b.list.SetSize(width-b.widthMargin, height-b.heightMargin)
	b.list.Styles.PaginationStyle = b.style.LogPaginator.Copy().Width(width - b.widthMargin)
}

func (b *Bubble) Help() []utils.HelpEntry {
	return nil
}

func (b *Bubble) updateItems() tea.Cmd {
	files := make([]list.Item, 0)
	dirs := make([]list.Item, 0)
	t, err := b.repo.Tree(b.ref, b.path)
	if err != nil {
		return func() tea.Msg { return utils.ErrMsg{Err: err} }
	}
	ents, err := t.Entries()
	if err != nil {
		return func() tea.Msg { return utils.ErrMsg{Err: err} }
	}
	ents.Sort()
	for _, e := range ents {
		if e.IsTree() {
			dirs = append(dirs, item{e})
		} else {
			files = append(files, item{e})
		}
	}
	cmd := b.list.SetItems(append(dirs, files...))
	b.list.Select(0)
	b.SetSize(b.width, b.height)
	return cmd
}

func (b *Bubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		if b.state == errorState {
			ref, _ := b.repo.HEAD()
			b.ref = ref
			return b, tea.Batch(b.reset(), func() tea.Msg {
				return ref
			})
		}

		switch msg.String() {
		case "F":
			return b, b.reset()
		case "enter", "right", "l":
			if len(b.list.Items()) > 0 && b.state == treeState {
				index := b.list.Index()
				item := b.list.SelectedItem().(item)
				mode := item.Mode()
				b.path = filepath.Join(b.path, item.Name())
				if mode.IsDir() {
					b.lastSelected = append(b.lastSelected, index)
					cmds = append(cmds, b.updateItems())
				} else {
					b.lastSelected = append(b.lastSelected, index)
					cmds = append(cmds, b.loadFile(item))
				}
			}
		case "esc", "left", "h":
			if b.state != treeState {
				b.state = treeState
			}
			p := filepath.Dir(b.path)
			b.path = p
			cmds = append(cmds, b.updateItems())
			index := 0
			if len(b.lastSelected) > 0 {
				index = b.lastSelected[len(b.lastSelected)-1]
				b.lastSelected = b.lastSelected[:len(b.lastSelected)-1]
			}
			b.list.Select(index)
		}

	case refs.RefMsg:
		b.ref = msg
		return b, b.reset()

	case utils.ErrMsg:
		b.error = msg
		b.state = errorState
		return b, nil

	case fileMsg:
		content := b.renderFile(msg)
		b.fileViewport.Viewport.SetContent(content)
		b.fileViewport.Viewport.GotoTop()
		b.state = fileState
	}

	switch b.state {
	case fileState:
		rv, cmd := b.fileViewport.Update(msg)
		b.fileViewport = rv.(*vp.ViewportBubble)
		cmds = append(cmds, cmd)
	case treeState:
		l, cmd := b.list.Update(msg)
		b.list = l
		cmds = append(cmds, cmd)
	}

	return b, tea.Batch(cmds...)
}

func (b *Bubble) View() string {
	switch b.state {
	case treeState:
		return b.list.View()
	case errorState:
		return b.error.ViewWithPrefix(b.style, "Error")
	case fileState:
		return b.fileViewport.View()
	default:
		return ""
	}
}

func (b *Bubble) loadFile(i item) tea.Cmd {
	return func() tea.Msg {
		f := i.entry.File()
		if i.Mode().IsDir() || f == nil {
			return utils.ErrMsg{Err: utils.ErrInvalidFile}
		}
		bin, err := f.IsBinary()
		if err != nil {
			return utils.ErrMsg{Err: err}
		}
		if bin {
			return utils.ErrMsg{Err: utils.ErrBinaryFile}
		}
		c, err := f.Bytes()
		if err != nil {
			return utils.ErrMsg{Err: err}
		}
		return fileMsg{
			content: string(c),
		}
	}
}

func (b *Bubble) renderFile(m fileMsg) string {
	s := strings.Builder{}
	c := m.content
	if len(strings.Split(c, "\n")) > utils.MaxDiffLines {
		s.WriteString(b.style.TreeNoItems.Render(utils.ErrFileTooLarge.Error()))
	} else {
		w := b.width - b.widthMargin - b.style.RepoBody.GetHorizontalFrameSize()
		f, err := utils.RenderFile(b.path, m.content, w)
		if err != nil {
			s.WriteString(err.Error())
		} else {
			s.WriteString(f)
		}
	}
	return b.style.TreeFileContent.Copy().Width(b.width - b.widthMargin).Render(s.String())
}
