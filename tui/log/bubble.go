package log

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/internal/tui/style"
	"github.com/charmbracelet/soft-serve/tui/common"
	"github.com/charmbracelet/soft-serve/tui/refs"
	vp "github.com/charmbracelet/soft-serve/tui/viewport"
)

var (
	diffChroma = &gansi.CodeBlockElement{
		Code:     "",
		Language: "diff",
	}
	waitBeforeLoading = time.Millisecond * 300
)

type itemsMsg struct{}

type commitMsg *git.Commit

type countMsg int64

type sessionState int

const (
	logState sessionState = iota
	commitState
	errorState
)

type item struct {
	*git.Commit
}

func (i item) Title() string {
	if i.Commit != nil {
		return strings.Split(i.Commit.Message, "\n")[0]
	}
	return ""
}

func (i item) FilterValue() string { return i.Title() }

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
	if i.Commit == nil {
		return
	}

	hash := i.ID.String()
	leftMargin := d.style.LogItemSelector.GetMarginLeft() +
		d.style.LogItemSelector.GetWidth() +
		d.style.LogItemHash.GetMarginLeft() +
		d.style.LogItemHash.GetWidth() +
		d.style.LogItemInactive.GetMarginLeft()
	title := common.TruncateString(i.Title(), m.Width()-leftMargin, "…")
	if index == m.Index() {
		fmt.Fprint(w, d.style.LogItemSelector.Render(">")+
			d.style.LogItemHash.Bold(true).Render(hash[:7])+
			d.style.LogItemActive.Render(title))
	} else {
		fmt.Fprint(w, d.style.LogItemSelector.Render(" ")+
			d.style.LogItemHash.Render(hash[:7])+
			d.style.LogItemInactive.Render(title))
	}
}

type Bubble struct {
	repo           common.GitRepo
	count          int64
	list           list.Model
	state          sessionState
	commitViewport *vp.ViewportBubble
	ref            *git.Reference
	style          *style.Styles
	width          int
	widthMargin    int
	height         int
	heightMargin   int
	error          common.ErrMsg
	spinner        spinner.Model
	loading        bool
	loadingStart   time.Time
	selectedCommit *git.Commit
	nextPage       int
}

func NewBubble(repo common.GitRepo, styles *style.Styles, width, widthMargin, height, heightMargin int) *Bubble {
	l := list.New([]list.Item{}, itemDelegate{styles}, width-widthMargin, height-heightMargin)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowPagination(true)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()
	l.KeyMap.NextPage = common.NextPage
	l.KeyMap.PrevPage = common.PrevPage
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner
	b := &Bubble{
		commitViewport: &vp.ViewportBubble{
			Viewport: &viewport.Model{},
		},
		repo:         repo,
		style:        styles,
		state:        logState,
		width:        width,
		widthMargin:  widthMargin,
		height:       height,
		heightMargin: heightMargin,
		list:         l,
		spinner:      s,
	}
	b.SetSize(width, height)
	return b
}

func (b *Bubble) countCommits() tea.Msg {
	if b.ref == nil {
		ref, err := b.repo.HEAD()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		b.ref = ref
	}
	count, err := b.repo.CountCommits(b.ref)
	if err != nil {
		return common.ErrMsg{Err: err}
	}
	return countMsg(count)
}

func (b *Bubble) updateItems() tea.Msg {
	if b.count == 0 {
		b.count = int64(b.countCommits().(countMsg))
	}
	count := b.count
	items := make([]list.Item, count)
	page := b.nextPage
	limit := b.list.Paginator.PerPage
	skip := page * limit
	// CommitsByPage pages start at 1
	cc, err := b.repo.CommitsByPage(b.ref, page+1, limit)
	if err != nil {
		return common.ErrMsg{Err: err}
	}
	for i, c := range cc {
		idx := i + skip
		if int64(idx) >= count {
			break
		}
		items[idx] = item{c}
	}
	b.list.SetItems(items)
	b.SetSize(b.width, b.height)
	return itemsMsg{}
}

func (b *Bubble) Help() []common.HelpEntry {
	return nil
}

func (b *Bubble) GotoTop() {
	b.commitViewport.Viewport.GotoTop()
}

func (b *Bubble) Init() tea.Cmd {
	return nil
}

func (b *Bubble) SetSize(width, height int) {
	b.width = width
	b.height = height
	b.commitViewport.Viewport.Width = width - b.widthMargin
	b.commitViewport.Viewport.Height = height - b.heightMargin
	b.list.SetSize(width-b.widthMargin, height-b.heightMargin)
	b.list.Styles.PaginationStyle = b.style.LogPaginator.Copy().Width(width - b.widthMargin)
}

func (b *Bubble) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.SetSize(msg.Width, msg.Height)
		cmds = append(cmds, b.updateItems)

	case tea.KeyMsg:
		switch msg.String() {
		case "C":
			b.count = 0
			b.loading = true
			b.loadingStart = time.Now().Add(-waitBeforeLoading) // always show spinner
			b.list.Select(0)
			b.nextPage = 0
			return b, tea.Batch(b.updateItems, b.spinner.Tick)
		case "enter", "right", "l":
			if b.state == logState {
				i := b.list.SelectedItem()
				if i != nil {
					c, ok := i.(item)
					if ok {
						b.selectedCommit = c.Commit
					}
				}
				cmds = append(cmds, b.loadCommit, b.spinner.Tick)
			}
		case "esc", "left", "h":
			if b.state != logState {
				b.state = logState
				b.selectedCommit = nil
			}
		}
		switch b.state {
		case logState:
			curPage := b.list.Paginator.Page
			m, cmd := b.list.Update(msg)
			b.list = m
			if m.Paginator.Page != curPage {
				b.loading = true
				b.loadingStart = time.Now()
				b.list.Paginator.Page = curPage
				b.nextPage = m.Paginator.Page
				cmds = append(cmds, b.updateItems, b.spinner.Tick)
			}
			cmds = append(cmds, cmd)
		case commitState:
			rv, cmd := b.commitViewport.Update(msg)
			b.commitViewport = rv.(*vp.ViewportBubble)
			cmds = append(cmds, cmd)
		}
		return b, tea.Batch(cmds...)
	case itemsMsg:
		b.loading = false
		b.list.Paginator.Page = b.nextPage
		if b.state != commitState {
			b.state = logState
		}
	case countMsg:
		b.count = int64(msg)
	case common.ErrMsg:
		b.error = msg
		b.state = errorState
		b.loading = false
		return b, nil
	case commitMsg:
		b.loading = false
		b.state = commitState
	case refs.RefMsg:
		b.ref = msg
		b.count = 0
		cmds = append(cmds, b.countCommits)
	case spinner.TickMsg:
		if b.loading {
			s, cmd := b.spinner.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			b.spinner = s
		}
	}

	return b, tea.Batch(cmds...)
}

func (b *Bubble) loadPatch(c *git.Commit) error {
	var patch strings.Builder
	style := b.style.LogCommit.Copy().Width(b.width - b.widthMargin - b.style.LogCommit.GetHorizontalFrameSize())
	p, err := b.repo.Diff(c)
	if err != nil {
		return err
	}
	stats := strings.Split(p.Stats().String(), "\n")
	for i, l := range stats {
		ch := strings.Split(l, "|")
		if len(ch) > 1 {
			adddel := ch[len(ch)-1]
			adddel = strings.ReplaceAll(adddel, "+", b.style.LogCommitStatsAdd.Render("+"))
			adddel = strings.ReplaceAll(adddel, "-", b.style.LogCommitStatsDel.Render("-"))
			stats[i] = strings.Join(ch[:len(ch)-1], "|") + "|" + adddel
		}
	}
	patch.WriteString(b.renderCommit(c))
	fpl := len(p.Files)
	if fpl > common.MaxDiffFiles {
		patch.WriteString("\n" + common.ErrDiffFilesTooLong.Error())
	} else {
		patch.WriteString("\n" + strings.Join(stats, "\n"))
	}
	if fpl <= common.MaxDiffFiles {
		ps := ""
		if len(strings.Split(ps, "\n")) > common.MaxDiffLines {
			patch.WriteString("\n" + common.ErrDiffTooLong.Error())
		} else {
			patch.WriteString("\n" + b.renderDiff(p))
		}
	}
	content := style.Render(patch.String())
	b.commitViewport.Viewport.SetContent(content)
	b.GotoTop()
	return nil
}

func (b *Bubble) loadCommit() tea.Msg {
	b.loading = true
	b.loadingStart = time.Now()
	c := b.selectedCommit
	if err := b.loadPatch(c); err != nil {
		return common.ErrMsg{Err: err}
	}
	return commitMsg(c)
}

func (b *Bubble) renderCommit(c *git.Commit) string {
	s := strings.Builder{}
	// FIXME: lipgloss prints empty lines when CRLF is used
	// sanitize commit message from CRLF
	msg := strings.ReplaceAll(c.Message, "\r\n", "\n")
	s.WriteString(fmt.Sprintf("%s\n%s\n%s\n%s\n",
		b.style.LogCommitHash.Render("commit "+c.ID.String()),
		b.style.LogCommitAuthor.Render(fmt.Sprintf("Author: %s <%s>", c.Author.Name, c.Author.Email)),
		b.style.LogCommitDate.Render("Date:   "+c.Committer.When.Format(time.UnixDate)),
		b.style.LogCommitBody.Render(msg),
	))
	return s.String()
}

func (b *Bubble) renderDiff(diff *git.Diff) string {
	var s strings.Builder
	var pr strings.Builder
	diffChroma.Code = diff.Patch()
	err := diffChroma.Render(&pr, common.RenderCtx)
	if err != nil {
		s.WriteString(fmt.Sprintf("\n%s", err.Error()))
	} else {
		s.WriteString(fmt.Sprintf("\n%s", pr.String()))
	}
	return s.String()
}

func (b *Bubble) View() string {
	if b.loading && b.loadingStart.Add(waitBeforeLoading).Before(time.Now()) {
		msg := fmt.Sprintf("%s loading commit", b.spinner.View())
		if b.selectedCommit == nil {
			msg += "s"
		}
		msg += "…"
		return msg
	}
	switch b.state {
	case logState:
		return b.list.View()
	case errorState:
		return b.error.ViewWithPrefix(b.style, "Error")
	case commitState:
		return b.commitViewport.View()
	default:
		return ""
	}
}
