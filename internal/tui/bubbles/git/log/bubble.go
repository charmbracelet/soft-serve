package log

import (
	"context"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	gansi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/refs"
	"github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/types"
	vp "github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/viewport"
	"github.com/charmbracelet/soft-serve/internal/tui/style"
	"github.com/dustin/go-humanize/english"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var (
	diffChroma = &gansi.CodeBlockElement{
		Code:     "",
		Language: "diff",
	}
)

type commitMsg struct {
	commit     *object.Commit
	parent     *object.Commit
	tree       *object.Tree
	parentTree *object.Tree
	patch      *object.Patch
}

type sessionState int

const (
	logState sessionState = iota
	commitState
	errorState
)

type item struct {
	*object.Commit
}

func (i item) Title() string {
	lines := strings.Split(i.Message, "\n")
	if len(lines) > 0 {
		return lines[0]
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

	leftMargin := d.style.LogItemSelector.GetMarginLeft() +
		d.style.LogItemSelector.GetWidth() +
		d.style.LogItemHash.GetMarginLeft() +
		d.style.LogItemHash.GetWidth() +
		d.style.LogItemInactive.GetMarginLeft()
	title := types.TruncateString(i.Title(), m.Width()-leftMargin, "…")
	if index == m.Index() {
		fmt.Fprint(w, d.style.LogItemSelector.Render(">")+
			d.style.LogItemHash.Bold(true).Render(i.Hash.String()[:7])+
			d.style.LogItemActive.Render(title))
	} else {
		fmt.Fprint(w, d.style.LogItemSelector.Render(" ")+
			d.style.LogItemHash.Render(i.Hash.String()[:7])+
			d.style.LogItemInactive.Render(title))
	}
}

type Bubble struct {
	repo           types.Repo
	list           list.Model
	state          sessionState
	commitViewport *vp.ViewportBubble
	ref            *plumbing.Reference
	style          *style.Styles
	width          int
	widthMargin    int
	height         int
	heightMargin   int
	error          types.ErrMsg
}

func NewBubble(repo types.Repo, styles *style.Styles, width, widthMargin, height, heightMargin int) *Bubble {
	l := list.New([]list.Item{}, itemDelegate{styles}, width-widthMargin, height-heightMargin)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()
	l.KeyMap.NextPage = types.NextPage
	l.KeyMap.PrevPage = types.PrevPage
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
		ref:          repo.GetHEAD(),
	}
	b.SetSize(width, height)
	return b
}

func (b *Bubble) reset() tea.Cmd {
	b.state = logState
	b.list.Select(0)
	return b.updateItems()
}

func (b *Bubble) updateItems() tea.Cmd {
	items := make([]list.Item, 0)
	cc, err := b.repo.GetCommits(b.ref)
	if err != nil {
		return func() tea.Msg { return types.ErrMsg{Err: err} }
	}
	for _, c := range cc {
		items = append(items, item{c})
	}
	return b.list.SetItems(items)
}

func (b *Bubble) Help() []types.HelpEntry {
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
	case types.ErrMsg:
		b.error = msg
		b.state = errorState
		return b, nil
	case commitMsg:
		content := b.renderCommit(msg)
		b.state = commitState
		b.commitViewport.Viewport.SetContent(content)
		b.GotoTop()
	case refs.RefMsg:
		b.ref = msg
	}

	switch b.state {
	case commitState:
		rv, cmd := b.commitViewport.Update(msg)
		b.commitViewport = rv.(*vp.ViewportBubble)
		cmds = append(cmds, cmd)
	case logState:
		l, cmd := b.list.Update(msg)
		b.list = l
		cmds = append(cmds, cmd)
	}

	return b, tea.Batch(cmds...)
}

func (b *Bubble) loadCommit() tea.Cmd {
	return func() tea.Msg {
		i := b.list.SelectedItem()
		if i == nil {
			return nil
		}
		c, ok := i.(item)
		if !ok {
			return nil
		}
		// Using commit trees fixes the issue when generating diff for the first commit
		// https://github.com/go-git/go-git/issues/281
		tree, err := c.Tree()
		if err != nil {
			return types.ErrMsg{Err: err}
		}
		var parent *object.Commit
		parentTree := &object.Tree{}
		if c.NumParents() > 0 {
			parent, err = c.Parents().Next()
			if err != nil {
				return types.ErrMsg{Err: err}
			}
			parentTree, err = parent.Tree()
			if err != nil {
				return types.ErrMsg{Err: err}
			}
		}
		ctx, cancel := context.WithTimeout(context.TODO(), types.MaxPatchWait)
		defer cancel()
		patch, err := parentTree.PatchContext(ctx, tree)
		if err != nil {
			return types.ErrMsg{Err: err}
		}
		return commitMsg{
			commit:     c.Commit,
			tree:       tree,
			parent:     parent,
			parentTree: parentTree,
			patch:      patch,
		}
	}
}

func (b *Bubble) renderCommit(m commitMsg) string {
	s := strings.Builder{}
	st := b.style
	c := m.commit
	// FIXME: lipgloss prints empty lines when CRLF is used
	// sanitize commit message from CRLF
	msg := strings.ReplaceAll(c.Message, "\r\n", "\n")
	s.WriteString(fmt.Sprintf("%s\n%s\n%s\n%s\n",
		st.LogCommitHash.Render("commit "+c.Hash.String()),
		st.LogCommitAuthor.Render("Author: "+c.Author.String()),
		st.LogCommitDate.Render("Date:   "+c.Committer.When.Format(time.UnixDate)),
		st.LogCommitBody.Render(msg),
	))
	stats := m.patch.Stats()
	if len(stats) > types.MaxDiffFiles {
		s.WriteString("\n" + types.ErrDiffFilesTooLong.Error())
	} else {
		s.WriteString("\n" + b.renderStats(stats))
	}
	ps := m.patch.String()
	if len(strings.Split(ps, "\n")) > types.MaxDiffLines {
		s.WriteString("\n" + types.ErrDiffTooLong.Error())
	} else {
		p := strings.Builder{}
		diffChroma.Code = ps
		err := diffChroma.Render(&p, types.RenderCtx)
		if err != nil {
			s.WriteString(fmt.Sprintf("\n%s", err.Error()))
		} else {
			s.WriteString(fmt.Sprintf("\n%s", p.String()))
		}
	}
	return st.LogCommit.Copy().Width(b.width - b.widthMargin - st.LogCommit.GetHorizontalFrameSize()).Render(s.String())
}

func (b *Bubble) renderStats(fileStats object.FileStats) string {
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
	for _, fs := range fileStats {
		if int(longestLength) < len(fs.Name) {
			longestLength = float64(len(fs.Name))
		}
		totalChange := fs.Addition + fs.Deletion
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
	for _, fs := range fileStats {
		taddc += fs.Addition
		tdelc += fs.Deletion
		addn := float64(fs.Addition)
		deln := float64(fs.Deletion)
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
		diffLines := fmt.Sprint(fs.Addition + fs.Deletion)
		totalDiffLines := fmt.Sprint(int(longestTotalChange))
		fmt.Fprintf(&output, "%s | %s %s%s\n",
			fs.Name+strings.Repeat(" ", int(longestLength)-len(fs.Name)),
			strings.Repeat(" ", len(totalDiffLines)-len(diffLines))+diffLines,
			b.style.LogCommitStatsAdd.Render(adds),
			b.style.LogCommitStatsDel.Render(dels))
	}
	files := len(fileStats)
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

func (b *Bubble) View() string {
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
