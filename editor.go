package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/atotto/clipboard"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
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
	readOnly    bool
}

func (e *Editor) setYankedRunes(rs []rune) {
	e.yankedRunes = rs
	_ = clipboard.WriteAll(string(rs))
}

func CreateEditor() *Editor {
	return &Editor{}
}

func (e *Editor) SetReadOnly(readOnly bool) {
	e.readOnly = readOnly
}

func (e *Editor) ReadOnly() bool {
	return e.readOnly
}

func (e *Editor) SetText(text string) {
	sc := bufio.NewScanner(strings.NewReader(text))
	var lines []EditorLine
	for sc.Scan() {
		line := sc.Text()
		lines = append(lines, EditorLine(line))
	}
	lines = append(lines, EditorLine(""))
	e.lines = lines
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
	if e.point.column >= len(currentLine) {
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

func (e *Editor) KillBetween(start, end EditorPoint) (result []rune) {
	e.point = end
	for e.point.line > start.line || e.point.column > start.column {
		e.AdvanceColumn(-1)
		result = append(result, e.DeleteRune())
	}
	e.ForgetMark()
	slices.Reverse(result)
	e.setYankedRunes(result)
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
	e.setYankedRunes(yankedRunes)
	e.ForgetMark()
}

func (e *Editor) Paste() {
	clipboardContents, err := clipboard.ReadAll()
	if err == nil && clipboardContents != "" {
		e.yankedRunes = []rune(clipboardContents)
	}
	sourceRunes := e.yankedRunes
	if sourceRunes == nil {
		return
	}
	e.InsertRunes(sourceRunes)
}

func (e *Editor) Reset() {
	e.ForgetMark()
}

func (e *Editor) InsertRune(r rune) {
	if e.readOnly {
		return
	}
	if r == '\n' {
		e.SplitLine()
	} else {
		p := e.point
		e.lines[p.line] = slices.Insert(e.lines[p.line], p.column, r)
		e.AdvanceColumn(1)
	}
}

func (e *Editor) InsertRunes(rs []rune) {
	if e.readOnly {
		return
	}
	for _, r := range rs {
		e.InsertRune(r)
	}
}

func (e *Editor) InsertSpacesUntilNextTabStop() {
	if e.readOnly {
		return
	}
	e.InsertRune(' ')
	for (e.point.column % TabWidth) != 0 {
		e.InsertRune(' ')
	}
}

func (e *Editor) DeleteRune() (deletedRune rune) {
	if e.readOnly {
		return 0
	}
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
	if e.readOnly {
		return
	}
	p := &e.point
	nextLine := slices.Clone(e.lines[p.line][p.column:])
	e.lines = slices.Insert(e.lines, p.line+1, nextLine)
	e.lines[p.line] = e.lines[p.line][:p.column]
	e.AdvanceLine(1)
	p.column = 0
}

func (e *Editor) Render(tp TilePane, currentToken *Token) {
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
	var highlightLine int
	var highlightStart int
	var highlightEnd int
	if currentToken != nil {
		highlightLine = currentToken.pos.Line - 1
		highlightStart = currentToken.pos.Column - 1
		highlightEnd = highlightStart + currentToken.length
	}
	for y := 0; y < tp.Height(); y++ {
		lineIndex := e.top + y
		if lineIndex >= len(e.lines) {
			break
		}
		line := e.lines[lineIndex]
		for x := 0; x < tp.Width(); x++ {
			runeIndex := e.left + x
			insideCurrent := currentToken != nil && lineIndex == highlightLine && runeIndex >= highlightStart && runeIndex < highlightEnd
			if runeIndex < len(line) {
				r := line[runeIndex]
				if insideCurrent {
					tp.WithBg(ColorCurrentToken, func() {
						tp.DrawRune(x, y, r)
					})
				} else if lineIndex == p.line && runeIndex == p.column {
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

func (e *Editor) RenderStatusLine(tp TilePane, bufferName string, currentToken *Token, nftotal, nfdone int) {
	leftText := fmt.Sprintf("%s  Ln %d, Col %d", bufferName, e.point.line+1, e.point.column+1)
	var rightText string
	if currentToken != nil {
		rightText = currentToken.String()
	}
	if nftotal != 0 {
		rightText += fmt.Sprintf(" %d%%", nfdone*100/nftotal)
	}
	paddedWidth := tp.Width() - 2
	if paddedWidth <= 0 {
		return
	}
	leftTextSize := utf8.RuneCountInString(leftText)
	rightStart := max(paddedWidth-utf8.RuneCountInString(rightText), leftTextSize+1)
	tp.WithFgBg(ColorWhite, ColorBlue, func() {
		for x := range tp.Width() {
			tp.DrawRune(x, 0, ' ')
		}
		tp.DrawString(1, 0, leftText)
		if rightText != "" && 1+rightStart < paddedWidth {
			tp.DrawString(1+rightStart, 0, rightText)
		}
	})
}

func (e *Editor) GetBytes() []byte {
	lines := e.lines
	numEmptyLinesAtEnd := 0
	for i := len(lines) - 1; i >= 0 && len(lines[i]) == 0; i-- {
		numEmptyLinesAtEnd++
	}
	lines = lines[:len(lines)-numEmptyLinesAtEnd]
	var b bytes.Buffer
	for _, line := range lines {
		b.WriteString(string(line))
		b.WriteByte('\n')
	}
	return b.Bytes()
}

func (e *Editor) Close() error {
	return nil
}
