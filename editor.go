package main

import (
	"bufio"
	"slices"
	"strings"
	"unicode"
)

type Line = []rune

type Editor struct {
	lines  []Line
	line   int
	column int
	top    int
	left   int
	mark   struct {
		line   int
		column int
	}
	markActive bool
}

// https://stackoverflow.com/a/61938973
func splitLines(s string) []string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}

func CreateEditor(text string) *Editor {
	var lines []Line
	for _, line := range splitLines(text) {
		lines = append(lines, Line(line))
	}
	return &Editor{
		lines:  lines,
		line:   0,
		column: 0,
		top:    0,
		left:   0,
	}
}

func (e *Editor) GetLine(index int) Line {
	if index < len(e.lines) {
		return e.lines[index]
	} else {
		return nil
	}
}

func (e *Editor) GetLineLength(index int) int {
	if index < len(e.lines) {
		return len(e.lines[index])
	} else {
		return 0
	}
}

func (e *Editor) CurrentRune() rune {
	currentLine := e.CurrentLine()
	if currentLine == nil {
		return 0x85 // NEL (NExtLine)
	}
	if e.column == len(currentLine) {
		return 0x85 // NEL (NExtLine)
	}
	return currentLine[e.column]
}

func (e *Editor) CurrentLine() Line {
	return e.GetLine(e.line)
}

func (e *Editor) CurrentLineLength() int {
	return e.GetLineLength(e.line)
}

func (e *Editor) AtBOF() bool {
	return e.line == 0 && e.column == 0
}

func (e *Editor) AtEOF() bool {
	return e.line == len(e.lines)
}

func (e *Editor) AtBOL() bool {
	return e.column == 0
}

func (e *Editor) AtEOL() bool {
	return e.column == e.CurrentLineLength()
}

func (e *Editor) AdvanceLine(amount int) {
	e.line += amount
	if e.line < 0 {
		e.line = 0
	} else if e.line > len(e.lines) {
		e.line = len(e.lines)
	}
	if e.column > e.CurrentLineLength() {
		e.column = e.CurrentLineLength()
	}
}

func (e *Editor) AdvanceColumn(amount int) {
	e.column += amount
	if e.column < 0 {
		if e.line > 0 {
			e.AdvanceLine(-1)
			e.column = e.CurrentLineLength()
		} else {
			e.column = 0
		}
	} else if e.column > e.CurrentLineLength() {
		if e.line < len(e.lines) {
			e.AdvanceLine(1)
			e.column = 0
		} else {
			e.column = 0
		}
	}
}

func (e *Editor) MoveToBOL() {
	e.column = 0
}

func (e *Editor) MoveToEOL() {
	e.column = e.CurrentLineLength()
}

func (e *Editor) WordLeft() {
	if !unicode.IsSpace(e.CurrentRune()) {
		e.AdvanceColumn(-1)
		if !unicode.IsSpace(e.CurrentRune()) {
			for !e.AtBOF() && !unicode.IsSpace(e.CurrentRune()) {
				e.AdvanceColumn(-1)
			}
			return
		}
	}
	for !e.AtBOF() && unicode.IsSpace(e.CurrentRune()) {
		e.AdvanceColumn(-1)
	}
	for !e.AtBOF() && !unicode.IsSpace(e.CurrentRune()) {
		e.AdvanceColumn(-1)
	}
	if unicode.IsSpace(e.CurrentRune()) {
		e.AdvanceColumn(1)
	}
}

func (e *Editor) WordRight() {
	for !e.AtEOF() && unicode.IsSpace(e.CurrentRune()) {
		e.AdvanceColumn(1)
	}
	for !e.AtEOF() && !unicode.IsSpace(e.CurrentRune()) {
		e.AdvanceColumn(1)
	}
	for !e.AtEOF() && unicode.IsSpace(e.CurrentRune()) {
		e.AdvanceColumn(1)
	}
}

func (e *Editor) SetMark() {
	e.mark.line = e.line
	e.mark.column = e.column
	e.markActive = true
}

func (e *Editor) SwapPointAndMark() {
	tempLine := e.line
	e.line = e.mark.line
	e.mark.line = tempLine
	tempColumn := e.column
	e.column = e.mark.column
	e.mark.column = tempColumn
}

func (e *Editor) ForgetMark() {
	e.mark.line = 0
	e.mark.column = 0
	e.markActive = false
}

func (e *Editor) IsInsideRegion(line, column int) bool {
	if e.line < e.mark.line || (e.line == e.mark.line && e.column < e.mark.column) {
		if line > e.line || (line == e.line && column >= e.column) {
			if line < e.mark.line || (line == e.mark.line && column < e.mark.column) {
				return true
			}
		}
	}
	if e.line > e.mark.line || (e.line == e.mark.line && e.column > e.mark.column) {
		if line < e.line || (line == e.line && column < e.column) {
			if line > e.mark.line || (line == e.mark.line && column >= e.mark.column) {
				return true
			}
		}
	}
	return false
}

func (e *Editor) KillRegion() {
	if !e.markActive {
		return
	}
	if e.line < e.mark.line || (e.line == e.mark.line && e.column < e.mark.column) {
		e.SwapPointAndMark()
	}
	if e.line > e.mark.line || (e.line == e.mark.line && e.column > e.mark.column) {
		count := 0
		for e.line >= e.mark.line && e.column > e.mark.column {
			e.AdvanceColumn(-1)
			count++
		}
		for range count {
			e.DeleteRune()
		}
	}
}

func (e *Editor) Quit() {
	e.ForgetMark()
}

func (e *Editor) InsertRune(r rune) {
	if e.line == len(e.lines) {
		e.lines = append(e.lines, Line(""))
	}
	e.lines[e.line] = slices.Insert(e.lines[e.line], e.column, r)
	e.AdvanceColumn(1)
}

func (e *Editor) DeleteRune() {
	if e.line == len(e.lines) {
		return
	}
	if e.column == e.CurrentLineLength() {
		if e.line == len(e.lines)-1 {
			return
		}
		e.lines[e.line] = slices.Insert(e.lines[e.line], e.column, e.lines[e.line+1]...)
		e.lines = slices.Delete(e.lines, e.line+1, e.line+2)
	} else {
		e.lines[e.line] = slices.Delete(e.lines[e.line], e.column, e.column+1)
	}
}

func (e *Editor) SplitLine() {
	if e.line == len(e.lines) {
		e.lines = append(e.lines, Line(""))
	} else {
		nextLine := slices.Clone(e.lines[e.line][e.column:])
		e.lines = slices.Insert(e.lines, e.line+1, nextLine)
		e.lines[e.line] = e.lines[e.line][:e.column]
	}
	e.AdvanceLine(1)
	e.column = 0
}

func (e *Editor) Render(tp TilePane) {
	if e.line < e.top {
		e.top = e.line
	}
	if e.line >= e.top+tp.Height() {
		e.top = e.line - tp.Height() + 1
	}
	if e.top < 0 {
		e.top = 0
	}
	if e.column < e.left {
		e.left = e.column
	}
	if e.column >= e.left+tp.Width() {
		e.left = e.column - tp.Width() + 1
	}
	if e.left < 0 {
		e.left = 0
	}
	for y := 0; y < tp.Height(); y++ {
		lineIndex := e.top + y
		if lineIndex < len(e.lines) {
			line := e.lines[lineIndex]
			for x := 0; x < tp.Width(); x++ {
				runeIndex := e.left + x
				if runeIndex < len(line) {
					if lineIndex == e.line && runeIndex == e.column {
						tp.WithBg(ColorHighlight, func() {
							tp.DrawRune(x, y, line[runeIndex])
						})
					} else {
						if e.markActive && e.IsInsideRegion(lineIndex, runeIndex) {
							tp.WithBg(ColorMark, func() {
								tp.DrawRune(x, y, line[runeIndex])
							})
						} else {
							tp.DrawRune(x, y, line[runeIndex])
						}
					}
				} else if lineIndex == e.line && runeIndex == e.column {
					tp.WithBg(ColorHighlight, func() {
						tp.DrawRune(x, y, ' ')
					})
				}
			}
		} else if lineIndex == e.line {
			tp.WithBg(ColorHighlight, func() {
				tp.DrawRune(0, y, ' ')
			})
		}
	}
}

func (e *Editor) Close() error {
	return nil
}
