package ui

import (
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// RenderDiff produces a colorized line-based diff between oldContent and
// newContent. Output looks like:
//
//   --- main.go
//   +++ main.go
//     unchanged line
//   - removed line
//   + added line
//     unchanged line
//
// The "+++"/"---" header rows use DiffHeader; added lines use DiffAdd
// (green), removed lines use DiffRemove (red), unchanged lines use the
// default text color so the eye is drawn to the changes.
//
// We use sergi/go-diff in line-mode (DiffLinesToChars + DiffCharsToLines)
// instead of raw character-mode. Line-mode avoids the visual noise of
// per-character diffing — a renamed variable shows as one removed line and
// one added line, not eight inline segments.
func RenderDiff(styles *Styles, filename, oldContent, newContent string) string {
	var b strings.Builder
	b.WriteString(styles.DiffHeader.Render("--- "+filename) + "\n")
	b.WriteString(styles.DiffHeader.Render("+++ "+filename) + "\n")

	if oldContent == newContent {
		// No changes — header only. Avoids running the diff algorithm at
		// all when callers (e.g. edit_file showing a no-op) would otherwise
		// pay for it on a large file.
		return b.String()
	}

	dmp := diffmatchpatch.New()
	charsA, charsB, lineArray := dmp.DiffLinesToChars(oldContent, newContent)
	diffs := dmp.DiffMain(charsA, charsB, false)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)

	for _, d := range diffs {
		// d.Text contains one-or-more lines for Equal segments, but for
		// Insert/Delete it's typically one line at a time after CharsToLines.
		// Splitting on \n covers all cases uniformly.
		lines := strings.Split(d.Text, "\n")
		// DiffLinesToChars terminates each line with \n, so the split
		// produces a trailing empty string we should ignore.
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		for _, line := range lines {
			switch d.Type {
			case diffmatchpatch.DiffInsert:
				b.WriteString(styles.DiffAdd.Render("+ "+line) + "\n")
			case diffmatchpatch.DiffDelete:
				b.WriteString(styles.DiffRemove.Render("- "+line) + "\n")
			case diffmatchpatch.DiffEqual:
				b.WriteString("  " + line + "\n")
			}
		}
	}
	return b.String()
}
