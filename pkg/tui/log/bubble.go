package log

import (
	"fmt"
	"io"
	"log"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/soft-serve/internal/tui/style"
	"github.com/charmbracelet/soft-serve/pkg/git"
	"github.com/charmbracelet/soft-serve/pkg/tui/refs"
	"github.com/charmbracelet/soft-serve/pkg/tui/utils"
	vp "github.com/charmbracelet/soft-serve/pkg/tui/viewport"
	"github.com/dustin/go-humanize/english"
)

var (
	diffChroma = &gansi.CodeBlockElement{
		Code:     "",
		Language: "diff",
	}
	waitBeforeLoading = time.Millisecond * 300
)

type commitMsg *git.Commit

type sessionState int

const (
	logState sessionState = iota
	commitState
	loadingState
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
	title := utils.TruncateString(i.Title(), m.Width()-leftMargin, "…")
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
	repo           utils.GitRepo
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
	error          utils.ErrMsg
	spinner        spinner.Model
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

func (b *Bubble) reset() tea.Cmd {
	b.state = logState
	b.list.Select(0)
	b.SetSize(b.width, b.height)
	cmd := b.updateItems()
	return cmd
}

func (b *Bubble) updateItems() tea.Cmd {
	count := b.count
	page := b.list.Paginator.Page
	limit := b.list.Paginator.PerPage
	skip := page * limit
	items := make([]list.Item, count)
	cc, err := b.repo.CommitsByPage(b.ref, page, limit)
	if err != nil {
		return func() tea.Msg { return utils.ErrMsg{Err: err} }
	}
	for i, c := range cc {
		idx := i + skip
		if idx >= int(count) {
			break
		}
		items[idx] = item{c}
	}
	cmd := b.list.SetItems(items)
	log.Printf("page %d/%d %d/%d", page, b.list.Paginator.TotalPages, skip, limit)
	return cmd
}

func (b *Bubble) Help() []utils.HelpEntry {
	return nil
}

func (b *Bubble) GotoTop() {
	b.commitViewport.Viewport.GotoTop()
}

func (b *Bubble) Init() tea.Cmd {
	errMsg := func(err error) tea.Cmd {
		return func() tea.Msg { return utils.ErrMsg{Err: err} }
	}
	ref, err := b.repo.HEAD()
	if err != nil {
		return errMsg(err)
	}
	b.ref = ref
	count, err := b.repo.CountCommits(ref)
	if err != nil {
		return errMsg(err)
	}
	b.count = count
	return func() tea.Msg { return refs.RefMsg(ref) }
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

	case tea.KeyMsg:
		switch msg.String() {
		case "C":
			return b, b.reset()
		case "enter", "right", "l":
			if b.state == logState {
				cmds = append(cmds, b.loadCommit())
			}
		case "esc", "left", "h":
			if b.state != logState {
				b.state = logState
			}
		}
		switch b.state {
		case logState:
			curPage := b.list.Paginator.Page
			m, cmd := b.list.Update(msg)
			b.list = m
			cmds = append(cmds, cmd)
			if m.Paginator.Page != curPage {
				cmds = append(cmds, b.updateItems())
			}
		case commitState:
			rv, cmd := b.commitViewport.Update(msg)
			b.commitViewport = rv.(*vp.ViewportBubble)
			cmds = append(cmds, cmd)
		}
		return b, tea.Batch(cmds...)
	case utils.ErrMsg:
		b.error = msg
		b.state = errorState
		return b, nil
	case commitMsg:
		if b.state == loadingState {
			cmds = append(cmds, b.spinner.Tick)
		}
	case refs.RefMsg:
		b.ref = msg
	case spinner.TickMsg:
		if b.state == loadingState {
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
	// ctx, cancel := context.WithTimeout(context.TODO(), types.MaxPatchWait)
	// defer cancel()
	p, err := b.repo.Diff(c)
	if err != nil {
		return err
	}
	patch.WriteString(b.renderCommit(c))
	fpl := len(p.Files)
	if fpl > utils.MaxDiffFiles {
		patch.WriteString("\n" + utils.ErrDiffFilesTooLong.Error())
	} else {
		patch.WriteString("\n" + b.renderStats(p))
	}
	if fpl <= utils.MaxDiffFiles {
		ps := ""
		if len(strings.Split(ps, "\n")) > utils.MaxDiffLines {
			patch.WriteString("\n" + utils.ErrDiffTooLong.Error())
		} else {
			patch.WriteString("\n" + b.renderDiff(p))
		}
	}
	content := style.Render(patch.String())
	b.commitViewport.Viewport.SetContent(content)
	b.GotoTop()
	return nil
}

func (b *Bubble) loadCommit() tea.Cmd {
	var err error
	done := make(chan struct{}, 1)
	i := b.list.SelectedItem()
	if i == nil {
		return nil
	}
	c, ok := i.(item)
	if !ok {
		return nil
	}
	go func() {
		err = b.loadPatch(c.Commit)
		done <- struct{}{}
		b.state = commitState
	}()
	return func() tea.Msg {
		select {
		case <-done:
		case <-time.After(waitBeforeLoading):
			b.state = loadingState
		}
		if err != nil {
			return utils.ErrMsg{Err: err}
		}
		return commitMsg(c.Commit)
	}
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

func (b *Bubble) renderStats(diff *git.Diff) string {
	padLength := float64(len(" "))
	newlineLength := float64(len("\n"))
	separatorLength := float64(len("|"))
	// Soft line length limit. The text length calculation below excludes
	// length of the change number. Adding that would take it closer to 80,
	// but probably not more than 80, until it's a huge number.
	lineLength := 72.0

	// Get the longest filename and longest total change.
	var longestLength float64
	var longestTotalChange float64
	for _, fs := range diff.Files {
		if int(longestLength) < len(fs.Name) {
			longestLength = float64(len(fs.Name))
		}
		totalChange := fs.NumAdditions() + fs.NumDeletions()
		if int(longestTotalChange) < totalChange {
			longestTotalChange = float64(totalChange)
		}
	}

	// Parts of the output:
	// <pad><filename><pad>|<pad><changeNumber><pad><+++/---><newline>
	// example: " main.go | 10 +++++++--- "

	// <pad><filename><pad>
	leftTextLength := padLength + longestLength + padLength

	// <pad><number><pad><+++++/-----><newline>
	// Excluding number length here.
	rightTextLength := padLength + padLength + newlineLength

	totalTextArea := leftTextLength + separatorLength + rightTextLength
	heightOfHistogram := lineLength - totalTextArea

	// Scale the histogram.
	var scaleFactor float64
	if longestTotalChange > heightOfHistogram {
		// Scale down to heightOfHistogram.
		scaleFactor = longestTotalChange / heightOfHistogram
	} else {
		scaleFactor = 1.0
	}

	taddc := 0
	tdelc := 0
	output := strings.Builder{}
	for _, fs := range diff.Files {
		taddc += fs.NumAdditions()
		tdelc += fs.NumDeletions()
		addn := float64(fs.NumAdditions())
		deln := float64(fs.NumDeletions())
		addc := int(math.Floor(addn / scaleFactor))
		delc := int(math.Floor(deln / scaleFactor))
		if addc < 0 {
			addc = 0
		}
		if delc < 0 {
			delc = 0
		}
		adds := strings.Repeat("+", addc)
		dels := strings.Repeat("-", delc)
		diffLines := fmt.Sprint(fs.NumAdditions() + fs.NumDeletions())
		totalDiffLines := fmt.Sprint(int(longestTotalChange))
		fmt.Fprintf(&output, "%s | %s %s%s\n",
			fs.Name+strings.Repeat(" ", int(longestLength)-len(fs.Name)),
			strings.Repeat(" ", len(totalDiffLines)-len(diffLines))+diffLines,
			b.style.LogCommitStatsAdd.Render(adds),
			b.style.LogCommitStatsDel.Render(dels))
	}
	files := diff.NumFiles()
	fc := fmt.Sprintf("%s changed", english.Plural(files, "file", ""))
	ins := fmt.Sprintf("%s(+)", english.Plural(taddc, "insertion", ""))
	dels := fmt.Sprintf("%s(-)", english.Plural(tdelc, "deletion", ""))
	fmt.Fprint(&output, fc)
	if taddc > 0 {
		fmt.Fprintf(&output, ", %s", ins)
	}
	if tdelc > 0 {
		fmt.Fprintf(&output, ", %s", dels)
	}
	fmt.Fprint(&output, "\n")

	return output.String()
}

func (b *Bubble) renderDiff(diff *git.Diff) string {
	var s strings.Builder
	pr := strings.Builder{}
	diffChroma.Code = ""
	err := diffChroma.Render(&pr, utils.RenderCtx)
	if err != nil {
		s.WriteString(fmt.Sprintf("\n%s", err.Error()))
	} else {
		s.WriteString(fmt.Sprintf("\n%s", pr.String()))
	}
	return s.String()
}

func (b *Bubble) View() string {
	switch b.state {
	case logState:
		return b.list.View()
	case loadingState:
		return fmt.Sprintf("%s loading commit…", b.spinner.View())
	case errorState:
		return b.error.ViewWithPrefix(b.style, "Error")
	case commitState:
		return b.commitViewport.View()
	default:
		return ""
	}
}
