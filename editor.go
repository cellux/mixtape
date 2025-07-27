package main

import (
	"bufio"
	"github.com/atotto/clipboard"
	"slices"
	"strings"
	"unicode"
)

const (
	TabWidth = 2
)

type EditorLine = []rune

type EditorPoint struct {
	line   int
	column int
}

type Editor struct {
	lines       []EditorLine
	point       EditorPoint
	mark        EditorPoint
	markActive  bool
	yankedRunes []rune
	top         int
	left        int
	height      int
}

// https://stackoverflow.com/a/61938973
func splitLines(s string) []string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if len(s) > 0 && s[len(s)-1] == '\n' {
		lines = append(lines, "")
	}
	return lines
}

func CreateEditor(text string) *Editor {
	var lines []EditorLine
	for _, line := range splitLines(text) {
		lines = append(lines, EditorLine(line))
	}
	if len(lines) == 0 {
		lines = append(lines, EditorLine(""))
	}
	return &Editor{
		lines: lines,
	}
}

func (e *Editor) GetLine(index int) EditorLine {
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

func (e *Editor) CurrentLine() EditorLine {
	return e.GetLine(e.point.line)
}

func (e *Editor) CurrentLineLength() int {
	return e.GetLineLength(e.point.line)
}

func (e *Editor) CurrentRune() rune {
	currentLine := e.CurrentLine()
	if currentLine == nil {
		return 0x85 // NEL (NExtLine)
	}
	if e.point.column == len(currentLine) {
		return 0x85 // NEL (NExtLine)
	}
	return currentLine[e.point.column]
}

func (e *Editor) AtFirstLine() bool {
	return e.point.line == 0
}

func (e *Editor) AtLastLine() bool {
	return e.point.line == len(e.lines)-1
}

func (e *Editor) AtBOL() bool {
	return e.point.column == 0
}

func (e *Editor) AtEOL() bool {
	return e.point.column == e.CurrentLineLength()
}

func (e *Editor) AtBOF() bool {
	return e.AtFirstLine() && e.AtBOL()
}

func (e *Editor) AtEOF() bool {
	return e.AtLastLine() && e.AtEOL()
}

func (e *Editor) AdvanceLine(amount int) {
	p := &e.point
	p.line += amount
	if p.line >= len(e.lines) {
		p.line = len(e.lines) - 1
	}
	if p.line < 0 {
		p.line = 0
	}
	if length := e.CurrentLineLength(); p.column > length {
		p.column = length
	}
}

func (e *Editor) AdvanceColumn(amount int) {
	p := &e.point
	p.column += amount
	if p.column > e.CurrentLineLength() {
		if p.line < len(e.lines)-1 {
			e.AdvanceLine(1)
			p.column = 0
		} else {
			p.column = e.CurrentLineLength()
		}
	} else if p.column < 0 {
		if p.line > 0 {
			e.AdvanceLine(-1)
			p.column = e.CurrentLineLength()
		} else {
			p.column = 0
		}
	}
}

func (e *Editor) MoveToBOL() {
	e.point.column = 0
}

func (e *Editor) MoveToEOL() {
	e.point.column = e.CurrentLineLength()
}

func (e *Editor) MoveToBOF() {
	e.point.line = 0
	e.point.column = 0
}

func (e *Editor) MoveToEOF() {
	e.point.line = len(e.lines) - 1
	e.MoveToEOL()
}

func isWordConstituent(r rune) bool {
	if unicode.IsLetter(r) {
		return true
	}
	if unicode.IsNumber(r) {
		return true
	}
	return false
}

func (e *Editor) WordLeft() {
	if !e.AtBOF() {
		e.AdvanceColumn(-1)
	}
	for !e.AtBOF() && !isWordConstituent(e.CurrentRune()) {
		e.AdvanceColumn(-1)
	}
	m := e.point
	for !e.AtBOF() && isWordConstituent(e.CurrentRune()) {
		e.AdvanceColumn(-1)
	}
	if e.point != m && !isWordConstituent(e.CurrentRune()) {
		e.AdvanceColumn(1)
	}
}

func (e *Editor) WordRight() {
	for !e.AtEOF() && unicode.IsSpace(e.CurrentRune()) {
		e.AdvanceColumn(1)
	}
	for !e.AtEOF() && !isWordConstituent(e.CurrentRune()) {
		e.AdvanceColumn(1)
	}
	for !e.AtEOF() && isWordConstituent(e.CurrentRune()) {
		e.AdvanceColumn(1)
	}
}

func (e *Editor) GetPoint() EditorPoint {
	return e.point
}

func (e *Editor) SetPoint(p EditorPoint) {
	e.point = p
}

func (e *Editor) GetMark() EditorPoint {
	return e.mark
}

func (e *Editor) SetMark() {
	e.mark = e.point
	e.markActive = true
}

func (e *Editor) SwapPointAndMark() {
	e.point, e.mark = e.mark, e.point
}

func (e *Editor) ForgetMark() {
	e.mark.line = 0
	e.mark.column = 0
	e.markActive = false
}

func (e *Editor) PointAndMarkInOrder() (EditorPoint, EditorPoint) {
	p := e.point
	m := e.mark
	if p.line > m.line {
		p, m = m, p
	}
	if p.line == m.line && p.column > m.column {
		p, m = m, p
	}
	return p, m
}

func (e *Editor) InsideRegion(line, column int) bool {
	p, m := e.PointAndMarkInOrder()
	if line > p.line || (line == p.line && column >= p.column) {
		if line < m.line || (line == m.line && column < m.column) {
			return true
		}
	}
	return false
}

func (e *Editor) KillBetween(start, end EditorPoint) []rune {
	e.point = end
	var result []rune
	for e.point.line > start.line || e.point.column > start.column {
		e.AdvanceColumn(-1)
		result = append(result, e.DeleteRune())
	}
	e.ForgetMark()
	slices.Reverse(result)
	return result
}

func (e *Editor) KillRegion() []rune {
	if !e.markActive {
		return nil
	}
	p, m := e.PointAndMarkInOrder()
	return e.KillBetween(p, m)
}

func (e *Editor) YankRegion() {
	if !e.markActive {
		return
	}
	p, m := e.PointAndMarkInOrder()
	var yankedRunes []rune
	for line := p.line; line <= m.line; line++ {
		if line == p.line && line == m.line {
			for i := p.column; i < m.column; i++ {
				yankedRunes = append(yankedRunes, e.lines[p.line][i])
			}
		} else if line == p.line {
			for i := p.column; i < e.GetLineLength(p.line); i++ {
				yankedRunes = append(yankedRunes, e.lines[p.line][i])
			}
			yankedRunes = append(yankedRunes, '\n')
		} else if line == m.line {
			for i := 0; i < m.column; i++ {
				yankedRunes = append(yankedRunes, e.lines[m.line][i])
			}
		} else {
			for i := 0; i < e.GetLineLength(line); i++ {
				yankedRunes = append(yankedRunes, e.lines[line][i])
			}
			yankedRunes = append(yankedRunes, '\n')
		}
	}
	e.yankedRunes = yankedRunes
	e.ForgetMark()
}

func (e *Editor) Paste() {
	sourceRunes := e.yankedRunes
	if sourceRunes == nil {
		clipboardContents, err := clipboard.ReadAll()
		if err == nil {
			sourceRunes = []rune(clipboardContents)
		} else {
			return
		}
	}
	e.InsertRunes(sourceRunes)
}

func (e *Editor) ResetState() {
	e.ForgetMark()
}

func (e *Editor) InsertRune(r rune) {
	if r == '\n' {
		e.SplitLine()
	} else {
		p := e.point
		e.lines[p.line] = slices.Insert(e.lines[p.line], p.column, r)
		e.AdvanceColumn(1)
	}
}

func (e *Editor) InsertRunes(rs []rune) {
	for _, r := range rs {
		e.InsertRune(r)
	}
}

func (e *Editor) InsertSpacesUntilNextTabStop() {
	e.InsertRune(' ')
	for (e.point.column % TabWidth) != 0 {
		e.InsertRune(' ')
	}
}

func (e *Editor) DeleteRune() (deletedRune rune) {
	p := e.point
	if e.AtEOF() {
		return 0
	} else if e.AtEOL() {
		deletedRune = '\n'
		e.lines[p.line] = slices.Insert(e.lines[p.line], p.column, e.lines[p.line+1]...)
		e.lines = slices.Delete(e.lines, p.line+1, p.line+2)
	} else {
		deletedRune = e.lines[p.line][p.column]
		e.lines[p.line] = slices.Delete(e.lines[p.line], p.column, p.column+1)
	}
	return deletedRune
}

func (e *Editor) SplitLine() {
	p := &e.point
	nextLine := slices.Clone(e.lines[p.line][p.column:])
	e.lines = slices.Insert(e.lines, p.line+1, nextLine)
	e.lines[p.line] = e.lines[p.line][:p.column]
	e.AdvanceLine(1)
	p.column = 0
}

func (e *Editor) Render(tp TilePane) {
	p := e.point
	e.height = tp.Height()
	if p.line < e.top {
		e.top = p.line
	}
	if p.line >= e.top+tp.Height() {
		e.top = p.line - tp.Height() + 1
	}
	if e.top < 0 {
		e.top = 0
	}
	if p.column < e.left {
		e.left = p.column
	}
	if p.column >= e.left+tp.Width() {
		e.left = p.column - tp.Width() + 1
	}
	if e.left < 0 {
		e.left = 0
	}
	for y := 0; y < tp.Height(); y++ {
		lineIndex := e.top + y
		if lineIndex >= len(e.lines) {
			break
		}
		line := e.lines[lineIndex]
		for x := 0; x < tp.Width(); x++ {
			runeIndex := e.left + x
			if runeIndex < len(line) {
				r := line[runeIndex]
				if lineIndex == p.line && runeIndex == p.column {
					tp.WithBg(ColorHighlight, func() {
						tp.DrawRune(x, y, r)
					})
				} else if e.markActive && e.InsideRegion(lineIndex, runeIndex) {
					tp.WithBg(ColorMark, func() {
						tp.DrawRune(x, y, r)
					})
				} else {
					tp.DrawRune(x, y, r)
				}
			} else if lineIndex == p.line && runeIndex == p.column {
				tp.WithBg(ColorHighlight, func() {
					tp.DrawRune(x, y, ' ')
				})
			}
		}
	}
}

func (e *Editor) GetBytes() []byte {
	bytes := make([]byte, 0, 65536)
	for i, line := range e.lines {
		if i > 0 {
			bytes = append(bytes, '\n')
		}
		bytes = append(bytes, []byte(string(line))...)
	}
	return bytes
}

func (e *Editor) Close() error {
	return nil
}
