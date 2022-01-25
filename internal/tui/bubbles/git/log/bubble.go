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
	"github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/types"
	vp "github.com/charmbracelet/soft-serve/internal/tui/bubbles/git/viewport"
	"github.com/charmbracelet/soft-serve/internal/tui/style"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/muesli/reflow/wrap"
)

var (
	diffChroma = &gansi.CodeBlockElement{
		Code:     "",
		Language: "diff",
	}
)

type pageView int

const (
	logView pageView = iota
	commitView
)

type item struct {
	*types.Commit
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
	pageView       pageView
	commitViewport *vp.ViewportBubble
	style          *style.Styles
	width          int
	widthMargin    int
	height         int
	heightMargin   int
	rctx           gansi.RenderContext
}

func NewBubble(repo types.Repo, style *style.Styles, width, widthMargin, height, heightMargin int) *Bubble {
	l := list.NewModel([]list.Item{}, itemDelegate{style}, width-widthMargin, height-heightMargin)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	b := &Bubble{
		commitViewport: &vp.ViewportBubble{
			Viewport: &viewport.Model{},
		},
		repo:         repo,
		style:        style,
		pageView:     logView,
		width:        width,
		widthMargin:  widthMargin,
		height:       height,
		heightMargin: heightMargin,
		rctx:         types.RenderCtx,
		list:         l,
	}
	b.SetSize(width, height)
	return b
}

func (b *Bubble) UpdateItems() tea.Cmd {
	items := make([]list.Item, 0)
	for _, c := range b.repo.GetCommits(0) {
		items = append(items, item{c})
	}
	return b.list.SetItems(items)
}

func (b *Bubble) Help() []types.HelpEntry {
	switch b.pageView {
	case logView:
		return []types.HelpEntry{
			{"enter", "select"},
		}
	case commitView:
		return []types.HelpEntry{
			{"esc", "back"},
		}
	default:
		return []types.HelpEntry{}
	}
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
		if b.pageView == commitView {
			b.commitViewport.Viewport.SetContent(b.commitView())
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "L":
			b.pageView = logView
			b.list.Select(0)
			cmds = append(cmds, b.UpdateItems())
		case "down", "j":
			if b.pageView == logView {
				b.list.CursorDown()
			}
		case "up", "k":
			if b.pageView == logView {
				b.list.CursorUp()
			}
		case "enter":
			if b.pageView == logView {
				b.pageView = commitView
				b.commitViewport.Viewport.SetContent(b.commitView())
				b.GotoTop()
			}
		case "esc":
			if b.pageView == commitView {
				b.pageView = logView
			}
		}
	}
	if b.pageView == commitView {
		rv, cmd := b.commitViewport.Update(msg)
		b.commitViewport = rv.(*vp.ViewportBubble)
		cmds = append(cmds, cmd)
	}
	return b, tea.Batch(cmds...)
}

func (b *Bubble) writePatch(commitTree *object.Tree, parentTree *object.Tree, s io.StringWriter) {
	ctx, cancel := context.WithTimeout(context.TODO(), types.MaxPatchWait)
	defer cancel()
	patch, err := parentTree.PatchContext(ctx, commitTree)
	if err != nil {
		s.WriteString(err.Error())
	} else {
		stats := patch.Stats()
		if len(stats) > types.MaxDiffFiles {
			s.WriteString("\n" + types.ErrDiffFilesTooLong.Error())
			return
		}
		s.WriteString("\n" + b.renderStats(stats))
		p := strings.Builder{}
		ps := patch.String()
		if len(strings.Split(ps, "\n")) > types.MaxDiffLines {
			s.WriteString("\n" + types.ErrDiffTooLong.Error())
			return
		}
		diffChroma.Code = ps
		err = diffChroma.Render(&p, b.rctx)
		if err != nil {
			s.WriteString(err.Error())
		} else {
			s.WriteString(fmt.Sprintf("\n%s", p.String()))
		}
	}
}

func (b *Bubble) commitView() string {
	s := strings.Builder{}
	commit := b.list.SelectedItem().(item)
	// FIXME: lipgloss prints empty line when CRLF is used
	// sanitize commit message from CRLF
	msg := strings.ReplaceAll(commit.Message, "\r\n", "\n")
	s.WriteString(fmt.Sprintf("%s\n%s\n%s\n%s\n",
		b.style.LogCommitHash.Render("commit "+commit.Hash.String()),
		b.style.LogCommitAuthor.Render("Author: "+commit.Author.String()),
		b.style.LogCommitDate.Render("Date:   "+commit.Committer.When.Format(time.UnixDate)),
		b.style.LogCommitBody.Render(msg),
	))
	// Using commit trees fixes the issue when generating diff for the first commit
	// https://github.com/go-git/go-git/issues/281
	commitTree, err := commit.Tree()
	if err != nil {
		s.WriteString(err.Error())
	} else {
		parentTree := &object.Tree{}
		if commit.NumParents() != 0 {
			parent, err := commit.Parents().Next()
			if err != nil {
				s.WriteString(err.Error())
			} else {
				parentTree, err = parent.Tree()
				if err != nil {
					s.WriteString(err.Error())
				} else {
					b.writePatch(commitTree, parentTree, &s)
				}
			}
		} else {
			b.writePatch(commitTree, parentTree, &s)
		}
	}
	return b.style.LogCommit.Render(wrap.String(s.String(), b.width-b.widthMargin-b.style.LogCommit.GetHorizontalFrameSize()))
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
	fmt.Fprintf(&output, "%d file", files)
	if files != 1 {
		fmt.Fprintf(&output, "s")
	}
	fmt.Fprint(&output, " changed")
	if taddc > 0 {
		fmt.Fprintf(&output, ", %d insertion", taddc)
		if taddc != 1 {
			fmt.Fprintf(&output, "s")
		}
		fmt.Fprint(&output, "(+)")
	}
	if tdelc > 0 {
		fmt.Fprintf(&output, ", %d deletion", tdelc)
		if tdelc != 1 {
			fmt.Fprintf(&output, "s")
		}
		fmt.Fprint(&output, "(-)")
	}
	fmt.Fprint(&output, "\n")

	return output.String()
}

func (b *Bubble) View() string {
	switch b.pageView {
	case logView:
		return b.list.View()
	case commitView:
		return b.commitViewport.View()
	default:
		return ""
	}
}
