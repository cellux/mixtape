package main

import (
	"reflect"
	"strings"
)

type ListEntry interface {
	GetUniqueId() any
	Format() string
}

type ListDisplay struct {
	entries    []ListEntry
	formatted  map[any]string
	index      int
	top        int
	lastHeight int
	searchText string
}

func CreateListDisplay() *ListDisplay {
	return &ListDisplay{}
}

func (ld *ListDisplay) Reset() {
	ld.index = 0
	ld.top = 0
	ld.searchText = ""
}

func (ld *ListDisplay) PageSize() int {
	if ld.lastHeight > 0 {
		return ld.lastHeight
	}
	return 1
}

func (ld *ListDisplay) SetEntries(entries []ListEntry) {
	ld.entries = entries
	if ld.index >= len(ld.entries) {
		ld.index = len(ld.entries) - 1
	}
	if ld.index < 0 {
		ld.index = 0
	}
	ld.EnsureVisible()
}

func (ld *ListDisplay) rebuildFormats() {
	ld.formatted = make(map[any]string, len(ld.entries))
	for _, e := range ld.entries {
		ld.formatted[e.GetUniqueId()] = e.Format()
	}
}

func (ld *ListDisplay) format(e ListEntry) string {
	if ld.formatted == nil {
		ld.rebuildFormats()
	}
	id := e.GetUniqueId()
	if s, ok := ld.formatted[id]; ok {
		return s
	}
	formatted := e.Format()
	ld.formatted[id] = formatted
	return formatted
}

func (ld *ListDisplay) GetFilteredEntries() []ListEntry {
	if ld.searchText == "" {
		return ld.entries
	}
	needle := strings.ToLower(ld.searchText)
	var out []ListEntry
	for _, e := range ld.entries {
		if strings.Contains(strings.ToLower(ld.format(e)), needle) {
			out = append(out, e)
		}
	}
	return out
}

func (ld *ListDisplay) GetFilteredSelectionIndex() int {
	filtered := ld.GetFilteredEntries()
	if len(filtered) == 0 {
		return 0
	}
	var current ListEntry
	if ld.index >= 0 && ld.index < len(ld.entries) {
		current = ld.entries[ld.index]
	} else {
		current = filtered[0]
	}
	for i, e := range filtered {
		if reflect.DeepEqual(e.GetUniqueId(), current.GetUniqueId()) {
			return i
		}
	}
	return 0
}

func (ld *ListDisplay) SelectFiltered(idx int) {
	filtered := ld.GetFilteredEntries()
	if len(filtered) == 0 {
		ld.index = 0
		ld.top = 0
		return
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= len(filtered) {
		idx = len(filtered) - 1
	}
	selected := filtered[idx]
	for i, e := range ld.entries {
		if reflect.DeepEqual(e.GetUniqueId(), selected.GetUniqueId()) {
			ld.index = i
			break
		}
	}
	ld.EnsureVisible()
}

func (ld *ListDisplay) SelectEntry(entry ListEntry) bool {
	entryId := entry.GetUniqueId()
	for i, e := range ld.entries {
		if reflect.DeepEqual(e.GetUniqueId(), entryId) {
			ld.index = i
			ld.EnsureVisible()
			return true
		}
	}
	return false
}

func (ld *ListDisplay) SelectById(id any) bool {
	for i, e := range ld.entries {
		if reflect.DeepEqual(e.GetUniqueId(), id) {
			ld.index = i
			ld.EnsureVisible()
			return true
		}
	}
	return false
}

func (ld *ListDisplay) EnsureVisible() {
	if ld.lastHeight <= 0 {
		return
	}
	filtered := ld.GetFilteredEntries()
	if len(filtered) == 0 {
		ld.top = 0
		ld.index = 0
		return
	}
	selIdx := ld.GetFilteredSelectionIndex()
	if ld.top < 0 {
		ld.top = 0
	}
	if selIdx < ld.top {
		ld.top = selIdx
	}
	if selIdx >= ld.top+ld.lastHeight {
		ld.top = selIdx - ld.lastHeight + 1
	}
	maxTop := len(filtered) - 1
	if ld.top > maxTop {
		ld.top = maxTop
		if ld.top < 0 {
			ld.top = 0
		}
	}
}

func (ld *ListDisplay) MoveBy(delta int) {
	selIdx := ld.GetFilteredSelectionIndex()
	ld.SelectFiltered(selIdx + delta)
}

func (ld *ListDisplay) MoveTo(idx int) {
	ld.SelectFiltered(idx)
}

func (ld *ListDisplay) SelectedEntry() ListEntry {
	if len(ld.entries) == 0 || ld.index < 0 || ld.index >= len(ld.entries) {
		return nil
	}
	return ld.entries[ld.index]
}

func (ld *ListDisplay) Render(tp *TilePane) {
	ld.lastHeight = tp.Height()
	if ld.lastHeight <= 0 {
		return
	}
	ld.EnsureVisible()

	availableWidth := tp.Width()
	if availableWidth <= 0 {
		return
	}

	filtered := ld.GetFilteredEntries()
	selectedEntry := ld.SelectedEntry()

	row := 0
	for i := ld.top; i < len(filtered) && row < ld.lastHeight; i, row = i+1, row+1 {
		entry := filtered[i]
		line := entry.Format()
		runes := []rune(line)
		if len(runes) > availableWidth {
			runes = runes[:availableWidth]
			line = string(runes)
		}
		isSelected := selectedEntry != nil && reflect.DeepEqual(entry.GetUniqueId(), selectedEntry.GetUniqueId())
		if isSelected {
			tp.WithFgBg(ColorBlack, ColorWhite, func() {
				tp.DrawString(0, row, line)
			})
		} else {
			tp.DrawString(0, row, line)
		}
	}
}
