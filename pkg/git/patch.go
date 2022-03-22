package git

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/dustin/go-humanize/english"
	"github.com/gogs/git-module"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffSection is a wrapper to git.DiffSection with helper methods.
type DiffSection struct {
	*git.DiffSection

	initOnce sync.Once
	dmp      *diffmatchpatch.DiffMatchPatch
}

// diffFor computes inline diff for the given line.
func (s *DiffSection) diffFor(line *git.DiffLine) string {
	fallback := line.Content

	// Find equivalent diff line, ignore when not found.
	var diff1, diff2 string
	switch line.Type {
	case git.DiffLineAdd:
		compareLine := s.Line(git.DiffLineDelete, line.RightLine)
		if compareLine == nil {
			return fallback
		}

		diff1 = compareLine.Content
		diff2 = line.Content

	case git.DiffLineDelete:
		compareLine := s.Line(git.DiffLineAdd, line.LeftLine)
		if compareLine == nil {
			return fallback
		}

		diff1 = line.Content
		diff2 = compareLine.Content

	default:
		return fallback
	}

	s.initOnce.Do(func() {
		s.dmp = diffmatchpatch.New()
		s.dmp.DiffEditCost = 100
	})

	diffs := s.dmp.DiffMain(diff1[1:], diff2[1:], true)
	diffs = s.dmp.DiffCleanupEfficiency(diffs)

	return diffsToString(diffs, line.Type)
}

func diffsToString(diffs []diffmatchpatch.Diff, lineType git.DiffLineType) string {
	buf := bytes.NewBuffer(nil)

	// Reproduce signs which are cutted for inline diff before.
	switch lineType {
	case git.DiffLineAdd:
		buf.WriteByte('+')
	case git.DiffLineDelete:
		buf.WriteByte('-')
	}

	const (
		addedCodePrefix   = ``
		removedCodePrefix = ``
		codeTagSuffix     = ``
	)

	for i := range diffs {
		switch {
		case diffs[i].Type == diffmatchpatch.DiffInsert && lineType == git.DiffLineAdd:
			buf.WriteString(addedCodePrefix)
			buf.WriteString(diffs[i].Text)
			buf.WriteString(codeTagSuffix)
		case diffs[i].Type == diffmatchpatch.DiffDelete && lineType == git.DiffLineDelete:
			buf.WriteString(removedCodePrefix)
			buf.WriteString(diffs[i].Text)
			buf.WriteString(codeTagSuffix)
		case diffs[i].Type == diffmatchpatch.DiffEqual:
			buf.WriteString(diffs[i].Text)
		}
	}

	return string(buf.Bytes())
}

// DiffFile is a wrapper to git.DiffFile with helper methods.
type DiffFile struct {
	*git.DiffFile
	Sections []*DiffSection
}

// FileStats
type FileStats []*DiffFile

// String returns a string representation of file stats.
func (fs FileStats) String() string {
	return printStats(fs)
}

func printStats(stats FileStats) string {
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
	for _, fs := range stats {
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
	for _, fs := range stats {
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
			adds,
			dels)
	}
	files := len(stats)
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

// Diff is a wrapper around git.Diff with helper methods.
type Diff struct {
	*git.Diff
	Files []*DiffFile
}

// FileStats returns the diff file stats.
func (d *Diff) Stats() FileStats {
	return d.Files
}

// Patch returns the diff as a patch.
func (d *Diff) Patch() string {
	var p strings.Builder
	for _, f := range d.Files {
		for _, s := range f.Sections {
			for _, l := range s.Lines {
				p.WriteString(s.diffFor(l))
				p.WriteString("\n")
			}
		}
	}
	return p.String()
}
