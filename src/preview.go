package fzf

import (
	"fmt"
	"math"
	"strings"

	"github.com/rivo/uniseg"

	"github.com/junegunn/fzf/src/tui"
)

// --- Preview capability queries ---

func mayTriggerPreview(opts *Options) bool {
	if opts.ListenAddr != nil {
		return true
	}
	for _, actions := range opts.Keymap {
		for _, action := range actions {
			switch action.t {
			case actPreview, actChangePreview, actTransform, actBgTransform:
				return true
			}
		}
	}
	return false
}

func (t *Terminal) hasPreviewer() bool {
	return t.previewBox != nil
}

func (t *Terminal) needPreviewWindow() bool {
	return t.hasPreviewer() && len(t.previewOpts.command) > 0 && t.activePreviewOpts.Visible()
}

// Check if previewer is currently in action (invisible previewer with size 0 or visible previewer)
func (t *Terminal) canPreview() bool {
	return t.hasPreviewer() && (!t.activePreviewOpts.Visible() && !t.activePreviewOpts.hidden || t.hasPreviewWindow())
}

func (t *Terminal) hasPreviewWindow() bool {
	return t.pwindow != nil
}

func (t *Terminal) hasPreviewWindowOnRight() bool {
	return t.hasPreviewWindow() && t.activePreviewOpts.position == posRight
}

// --- Preview environment and sizing ---

func (t *Terminal) environForPreview() []string {
	return t.environImpl(true)
}

func (t *Terminal) minPreviewSize(opts *previewOpts) (int, int) {
	border := opts.Border(t.layout)
	minPreviewWidth := 1 + borderColumns(border, t.borderWidth)
	minPreviewHeight := 1 + borderLines(border)

	switch opts.position {
	case posLeft, posRight:
		if len(t.scrollbar) > 0 && !border.HasRight() {
			// Need a column to show scrollbar
			minPreviewWidth++
		}
	}

	return minPreviewWidth, minPreviewHeight
}

func (t *Terminal) pwindowSize() tui.TermSize {
	if t.pwindow == nil {
		return tui.TermSize{}
	}
	size := tui.TermSize{Lines: t.pwindow.Height(), Columns: t.pwindow.Width()}

	if t.termSize.PxWidth > 0 {
		size.PxWidth = size.Columns * t.termSize.PxWidth / t.termSize.Columns
		size.PxHeight = size.Lines * t.termSize.PxHeight / t.termSize.Lines
	}
	return size
}

// --- Preview process management ---

func (t *Terminal) killPreview() {
	select {
	case t.killChan <- true:
		<-t.killedChan
	default:
	}
}

func (t *Terminal) cancelPreview() {
	select {
	case t.killChan <- false:
	default:
	}
}

// --- Preview template parsing ---

func hasPreviewFlags(template string) (slot bool, plus bool, asterisk bool, forceUpdate bool) {
	for _, match := range placeholder.FindAllString(template, -1) {
		escaped, _, flags := parsePlaceholder(match)
		if escaped {
			continue
		}
		slot = true
		plus = plus || flags.plus
		asterisk = asterisk || flags.asterisk
		forceUpdate = forceUpdate || flags.forceUpdate
	}
	return
}

// --- Preview scroll offset calculation ---

// followOffset computes the correct content-line offset for follow mode,
// accounting for line wrapping in the preview window.
func (t *Terminal) followOffset() int {
	lines := t.previewer.lines
	headerLines := t.activePreviewOpts.headerLines
	height := t.pwindow.Height() - headerLines
	if height <= 0 || len(lines) <= headerLines {
		return headerLines
	}

	body := lines[headerLines:]
	if !t.activePreviewOpts.wrap {
		return max(t.previewer.offset, headerLines+len(body)-height)
	}

	maxWidth := t.pwindow.Width()
	visualLines := 0
	for i := len(body) - 1; i >= 0; i-- {
		h := t.previewLineHeight(body[i], maxWidth)
		if visualLines+h > height {
			return min(len(lines)-1, headerLines+i+1)
		}
		visualLines += h
	}
	return headerLines
}

// previewLineHeight estimates the number of visual lines a preview content line
// occupies when wrapping is enabled.
func (t *Terminal) previewLineHeight(line string, maxWidth int) int {
	if maxWidth <= 0 {
		return 1
	}

	// For word-wrap mode, count the sub-lines produced by word wrapping.
	// Each sub-line may still char-wrap if it contains a word longer than the width.
	if t.activePreviewOpts.wrapWord {
		subLines := t.wordWrapAnsiLine(line, maxWidth, t.previewWrapSignWidth)
		total := 0
		for i, sub := range subLines {
			prefixWidth := 0
			cols := maxWidth
			if i > 0 {
				prefixWidth = t.previewWrapSignWidth
				cols -= t.previewWrapSignWidth
			}
			w := t.ansiLineWidth(sub, prefixWidth)
			if cols <= 0 {
				cols = 1
			}
			total += max(1, (w+cols-1)/cols)
		}
		return total
	}

	// For char-wrap, compute visible width and divide by available width.
	w := t.ansiLineWidth(line, 0)
	if w <= maxWidth {
		return 1
	}
	remaining := w - maxWidth
	contWidth := max(1, maxWidth-t.previewWrapSignWidth)
	return 1 + (remaining+contWidth-1)/contWidth
}

// ansiLineWidth computes the display width of a string, skipping ANSI escape sequences.
// prefixWidth is the visual offset where the content starts (e.g. wrap sign width for
// continuation lines), used for correct tab stop alignment.
func (t *Terminal) ansiLineWidth(line string, prefixWidth int) int {
	line = strings.TrimSuffix(line, "\n")
	trimmed, _, _ := extractColor(line, nil, nil)
	_, width := t.processTabsStr(trimmed, prefixWidth)
	return width - prefixWidth
}

func (t *Terminal) wordWrapAnsiLine(line string, maxWidth int, wrapSignWidth int) []string {
	if maxWidth <= 0 {
		return []string{line}
	}

	var result []string
	lineStart := 0
	width := 0
	lastSpaceStart := -1
	lastSpaceEnd := -1
	widthBeforeLastSpace := 0
	lastSpaceWidth := 0
	max := maxWidth
	pos := 0

	for pos < len(line) {
		// Find next ANSI escape sequence
		start, end := nextAnsiEscapeSequence(line[pos:])

		// Determine the end of printable text before the next escape
		var printableEnd int
		if start < 0 {
			printableEnd = len(line)
		} else {
			printableEnd = pos + start
		}

		// Process printable characters using grapheme clusters
		gr := uniseg.NewGraphemes(line[pos:printableEnd])
		for gr.Next() {
			gStart, gEnd := gr.Positions()
			w := gr.Width()
			str := gr.Str()

			if str == "\t" {
				w = t.tabstop - (width % t.tabstop)
			}

			if str == " " || str == "\t" {
				lastSpaceStart = pos + gStart
				lastSpaceEnd = pos + gEnd
				widthBeforeLastSpace = width
				lastSpaceWidth = w
			}

			width += w

			if width > max && lastSpaceEnd > lineStart {
				result = append(result, line[lineStart:lastSpaceStart])
				lineStart = lastSpaceEnd
				width -= widthBeforeLastSpace + lastSpaceWidth
				lastSpaceStart = -1
				lastSpaceEnd = -1
				widthBeforeLastSpace = 0
				max = maxWidth - wrapSignWidth
			}
		}
		pos = printableEnd

		// Skip the ANSI escape sequence
		if start >= 0 {
			pos += end - start
		}
	}

	result = append(result, line[lineStart:])
	return result
}

// --- Preview rendering ---

func (t *Terminal) renderPreviewSpinner() {
	numLines := len(t.previewer.lines)
	spin := t.previewer.spinner
	if len(spin) > 0 || t.previewer.scrollable {
		maxWidth := t.pwindow.Width()
		if !t.previewer.scrollable || !t.activePreviewOpts.info {
			if maxWidth > 0 {
				t.pwindow.Move(0, maxWidth-1)
				t.pwindow.CPrint(tui.ColPreviewSpinner, spin)
			}
		} else {
			offsetString := fmt.Sprintf("%d/%d", t.previewer.offset+1, numLines)
			if len(spin) > 0 {
				spin += " "
				maxWidth -= 2
			}
			offsetRunes, _ := t.trimRight([]rune(offsetString), maxWidth)
			pos := maxWidth - t.displayWidth(offsetRunes)
			t.pwindow.Move(0, pos)
			if maxWidth > 0 {
				t.pwindow.CPrint(tui.ColPreviewSpinner, spin)
				t.pwindow.CPrint(tui.ColInfo.WithAttr(tui.Reverse), string(offsetRunes))
			}
		}
	}
}

func (t *Terminal) renderPreviewArea(unchanged bool) {
	if t.previewed.wipe && t.previewed.version != t.previewer.version {
		t.previewed.wipe = false
		t.pwindow.Erase()
	} else if unchanged {
		t.pwindow.MoveAndClear(0, 0) // Clear scroll offset display
	} else {
		t.previewed.filled = false
		// We don't erase the window here to avoid flickering during scroll.
		// However, tcell renderer uses double-buffering technique and there's no
		// flickering. So we just erase the window and make the rest of the code
		// simpler.
		if !t.pwindow.EraseMaybe() {
			t.pwindow.DrawBorder()
			t.pwindow.Move(0, 0)
		}
	}

	height := t.pwindow.Height()
	body := t.previewer.lines
	headerLines := t.activePreviewOpts.headerLines
	// Do not enable preview header lines if it's value is too large
	if headerLines > 0 && headerLines < min(len(body), height) {
		header := t.previewer.lines[0:headerLines]
		body = t.previewer.lines[headerLines:]
		// Always redraw header
		t.renderPreviewText(height, header, 0, false)
		t.pwindow.MoveAndClear(t.pwindow.Y(), 0)
	}
	t.renderPreviewText(height, body, -t.previewer.offset+headerLines, unchanged)

	if !unchanged {
		t.pwindow.FinishFill()
	}

	if len(t.scrollbar) == 0 {
		return
	}

	effectiveHeight := height - headerLines
	barLength, barStart := getScrollbar(1, len(body), effectiveHeight, min(len(body)-effectiveHeight, t.previewer.offset-headerLines))
	t.renderPreviewScrollbar(headerLines, barLength, barStart)
}

func (t *Terminal) renderPreviewText(height int, lines []string, lineNo int, unchanged bool) {
	maxWidth := t.pwindow.Width()
	var ansi *ansiState
	spinnerRedraw := t.pwindow.Y() == 0
	wiped := false
	image := false
	wireframe := false
	var index int
	var line string
Loop:
	for index, line = range lines {
		var lbg tui.Color = -1
		if ansi != nil {
			ansi.lbg = -1
		}

		passThroughs, line := extractPassThroughs(line)
		line = strings.TrimLeft(strings.TrimRight(line, "\r\n"), "\r")

		if lineNo >= height || t.pwindow.Y() == height-1 && t.pwindow.X() > 0 {
			t.previewed.filled = true
			t.previewer.scrollable = true
			break
		} else if lineNo >= 0 {
			x := t.pwindow.X()
			y := t.pwindow.Y()
			if spinnerRedraw && lineNo > 0 {
				spinnerRedraw = false
				t.renderPreviewSpinner()
				t.pwindow.Move(y, x)
			}
			for idx, passThrough := range passThroughs {
				// Handling Sixel/iTerm image
				requiredLines := 0
				isSixel := strings.HasPrefix(passThrough, "\x1bP")
				isItermImage := strings.HasPrefix(passThrough, "\x1b]1337;")
				isImage := isSixel || isItermImage
				if isImage {
					t.previewed.wipe = true
					// NOTE: We don't have a good way to get the height of an iTerm image,
					// so we assume that it requires the full height of the preview
					// window.
					requiredLines = height

					if isSixel && t.termSize.PxHeight > 0 {
						rows := strings.Count(passThrough, "-")
						requiredLines = int(math.Ceil(float64(rows*6*t.termSize.Lines) / float64(t.termSize.PxHeight)))
					}
				}

				// Render wireframe when the image cannot be displayed entirely
				if requiredLines > 0 && y+requiredLines > height {
					top := true
					for ; y < height; y++ {
						t.pwindow.MoveAndClear(y, 0)
						t.pwindow.CFill(tui.ColPreview.Fg(), tui.ColPreview.Bg(), -1, tui.AttrRegular, t.makeImageBorder(maxWidth, top))
						top = false
					}
					wireframe = true
					t.previewed.filled = true
					t.previewer.scrollable = true
					break Loop
				}

				// Clear previous wireframe or any other text
				if (t.previewed.wireframe || isImage && !t.previewed.image) && !wiped {
					wiped = true
					for i := y + 1; i < height; i++ {
						t.pwindow.MoveAndClear(i, 0)
					}
				}
				image = image || isImage
				if idx == 0 {
					t.pwindow.MoveAndClear(y, x)
				} else {
					t.pwindow.Move(y, x)
				}
				t.tui.PassThrough(passThrough)

				if requiredLines > 0 {
					if y+requiredLines == height {
						t.pwindow.Move(height-1, maxWidth-1)
						t.previewed.filled = true
						break Loop
					}
					t.pwindow.MoveAndClear(y+requiredLines, 0)
				}
			}

			if len(passThroughs) > 0 && len(line) == 0 {
				continue
			}

			// Pre-split line into sub-lines for word wrapping
			var subLines []string
			if t.activePreviewOpts.wrapWord {
				subLines = t.wordWrapAnsiLine(line, maxWidth, t.previewWrapSignWidth)
			} else {
				subLines = []string{line}
			}

			var fillRet tui.FillReturn
			wrap := t.activePreviewOpts.wrap
			printWrapSign := func() {
				if t.pwindow.CFill(tui.ColPreview.Fg(), tui.ColPreview.Bg(), -1, tui.Dim, t.previewWrapSign) == tui.FillNextLine {
					t.pwindow.Move(t.pwindow.Y()-1, t.pwindow.Width())
				}
				fillRet = tui.FillContinue
			}
			for subIdx, subLine := range subLines {
				// Render wrap sign for continuation sub-lines
				if subIdx > 0 {
					if fillRet == tui.FillContinue {
						fillRet = t.pwindow.Fill("\n")
						if fillRet == tui.FillSuspend {
							t.previewed.filled = true
							break Loop
						}
					}
					printWrapSign()
				}

				prefixWidth := t.pwindow.X()
				var url *url
				_, _, ansi = extractColor(subLine, ansi, func(str string, ansi *ansiState) bool {
					if len(str) > 0 && fillRet == tui.FillNextLine {
						printWrapSign()
						prefixWidth = t.pwindow.X()
					}
					trimmed := []rune(str)
					isTrimmed := false
					if !wrap {
						trimmed, isTrimmed = t.trimRight(trimmed, maxWidth-t.pwindow.X())
					}
					if url == nil && ansi != nil && ansi.url != nil {
						url = ansi.url
						t.pwindow.LinkBegin(url.uri, url.params)
					}
					if url != nil && (ansi == nil || ansi.url == nil) {
						url = nil
						t.pwindow.LinkEnd()
					}
					if ansi != nil {
						lbg = ansi.lbg
					} else {
						lbg = -1
					}
					str, width := t.processTabs(trimmed, prefixWidth)
					if width > prefixWidth {
						prefixWidth = width
						colored := ansi != nil && ansi.colored()
						if t.theme.Colored && colored {
							fillRet = t.pwindow.CFill(ansi.fg, ansi.bg, ansi.ul, ansi.attr, str)
						} else {
							attr := tui.AttrRegular
							if colored {
								attr = ansi.attr
							}
							fillRet = t.pwindow.CFill(tui.ColPreview.Fg(), tui.ColPreview.Bg(), -1, attr, str)
						}
					}
					return !isTrimmed &&
						(fillRet == tui.FillContinue || wrap && fillRet == tui.FillNextLine)
				})
				if url != nil {
					t.pwindow.LinkEnd()
				}

				if fillRet == tui.FillSuspend {
					t.previewed.filled = true
					break Loop
				}
			}

			t.previewer.scrollable = t.previewer.scrollable || t.pwindow.Y() == height-1 && t.pwindow.X() == t.pwindow.Width() || t.previewed.filled
			if fillRet == tui.FillNextLine {
				continue
			} else if fillRet == tui.FillSuspend {
				t.previewed.filled = true
				break
			}
			if unchanged && lineNo == 0 {
				break
			}
			if t.theme.Colored && lbg >= 0 {
				fillRet = t.pwindow.CFill(-1, lbg, -1, tui.AttrRegular,
					strings.Repeat(" ", t.pwindow.Width()-t.pwindow.X())+"\n")
			} else {
				fillRet = t.pwindow.Fill("\n")
			}
			if fillRet == tui.FillSuspend {
				t.previewed.filled = true
				break
			}
		}
		lineNo++
	}
	t.previewer.scrollable = t.previewer.scrollable || t.previewed.filled || index < len(lines)-1
	t.previewed.image = image
	t.previewed.wireframe = wireframe
}

func (t *Terminal) renderPreviewScrollbar(yoff int, barLength int, barStart int) {
	height := t.pwindow.Height()
	w := t.pborder.Width()
	xw := [2]int{t.pwindow.Left(), t.pwindow.Width()}
	redraw := false
	if len(t.previewer.bar) != height || t.previewer.xw != xw || t.previewed.version != t.previewer.version {
		redraw = true
		t.previewer.bar = make([]bool, height)
		t.previewer.xw = xw
	}
	xshift := -1 - t.borderWidth
	if !t.activePreviewOpts.Border(t.layout).HasRight() {
		xshift = -1
	}
	yshift := 1
	if !t.activePreviewOpts.Border(t.layout).HasTop() {
		yshift = 0
	}
	for i := yoff; i < height; i++ {
		x := w + xshift
		y := i + yshift

		// Avoid unnecessary redraws
		bar := i >= yoff+barStart && i < yoff+barStart+barLength
		if !redraw && bar == t.previewer.bar[i] && !t.tui.NeedScrollbarRedraw() {
			continue
		}

		t.previewer.bar[i] = bar
		t.pborder.Move(y, x)
		if i >= yoff+barStart && i < yoff+barStart+barLength {
			t.pborder.CPrint(tui.ColPreviewScrollbar, t.previewScrollbar)
		} else {
			t.pborder.CPrint(tui.ColPreviewScrollbar, " ")
		}
	}
}

// --- Preview entry points ---

func (t *Terminal) printPreview() {
	if !t.hasPreviewWindow() || t.pwindow.Height() == 0 {
		return
	}
	numLines := len(t.previewer.lines)
	height := t.pwindow.Height()
	unchanged := (t.previewed.filled || numLines == t.previewed.numLines) &&
		t.previewer.version == t.previewed.version &&
		t.previewer.offset == t.previewed.offset
	t.previewer.scrollable = t.previewer.offset > t.activePreviewOpts.headerLines || numLines > height
	t.renderPreviewArea(unchanged)
	t.renderPreviewSpinner()
	t.previewed.numLines = numLines
	t.previewed.version = t.previewer.version
	t.previewed.offset = t.previewer.offset
}

func (t *Terminal) printPreviewDelayed() {
	if !t.hasPreviewWindow() || len(t.previewer.lines) > 0 && t.previewed.version == t.previewer.version {
		return
	}

	t.previewer.scrollable = false
	t.renderPreviewArea(true)

	message := t.trimMessage("Loading ..", t.pwindow.Width())
	pos := t.pwindow.Width() - len(message)
	t.pwindow.Move(0, pos)
	t.pwindow.CPrint(tui.ColInfo.WithAttr(tui.Reverse), message)
}
