package main

import (
	"bufio"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

//go:embed assets/app-icon.svg
var appIconSVG []byte

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

const (
	sourceAll          = "All"
	sourceUser         = "User"
	sourceSystem       = "System"
	sourceProcess      = "Process"
	sourceUserOverride = "User override"
)

type envItem struct {
	Value  string
	Source string
}

type envStore struct {
	items map[string]envItem
}

func newEnvStore() *envStore {
	s := &envStore{items: make(map[string]envItem)}
	s.ReloadFromProcess()
	return s
}

func (s *envStore) ReloadFromProcess() {
	processValues := make(map[string]string)
	for _, pair := range os.Environ() {
		key, value, ok := strings.Cut(pair, "=")
		if ok {
			processValues[key] = value
		}
	}

	sources := detectEnvSources(processValues)
	s.items = make(map[string]envItem, len(processValues))
	for key, value := range processValues {
		source := sources[key]
		if source == "" {
			source = sourceProcess
		}
		s.items[key] = envItem{Value: value, Source: source}
	}
}

func (s *envStore) KeysFiltered(filter, sourceFilter string) []string {
	filter = strings.ToLower(strings.TrimSpace(filter))
	keys := make([]string, 0, len(s.items))
	for key, item := range s.items {
		if !matchesSourceFilter(item.Source, sourceFilter) {
			continue
		}
		if filter == "" || strings.Contains(strings.ToLower(key), filter) || strings.Contains(strings.ToLower(item.Value), filter) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func matchesSourceFilter(itemSource, sourceFilter string) bool {
	if sourceFilter == "" || sourceFilter == sourceAll {
		return true
	}
	switch sourceFilter {
	case sourceUser:
		return itemSource == sourceUser || itemSource == sourceUserOverride
	default:
		return itemSource == sourceFilter
	}
}

func normalizeEditableSource(source string) string {
	switch source {
	case sourceUser, sourceUserOverride:
		return sourceUser
	case sourceSystem:
		return sourceSystem
	default:
		return sourceProcess
	}
}

func defaultSourceForFilter(sourceFilter string) string {
	switch sourceFilter {
	case sourceUser, sourceSystem, sourceProcess:
		return sourceFilter
	default:
		return sourceProcess
	}
}

func (s *envStore) Set(key, value, source string) error {
	key = strings.TrimSpace(key)
	if !envKeyPattern.MatchString(key) {
		return errors.New("invalid variable name")
	}

	item, exists := s.items[key]
	if !exists {
		item = envItem{Source: normalizeEditableSource(source)}
	}
	item.Value = value
	item.Source = normalizeEditableSource(source)
	s.items[key] = item
	return nil
}

func (s *envStore) Delete(key string) {
	delete(s.items, key)
}

func (s *envStore) Rename(oldKey, newKey, value, source string) error {
	newKey = strings.TrimSpace(newKey)
	if !envKeyPattern.MatchString(newKey) {
		return errors.New("invalid variable name")
	}

	item, exists := s.items[oldKey]
	if !exists {
		item = envItem{Source: normalizeEditableSource(source)}
	}

	if oldKey != newKey {
		if _, exists := s.items[newKey]; exists {
			return errors.New("variable already exists")
		}
		delete(s.items, oldKey)
	}

	item.Value = value
	item.Source = normalizeEditableSource(source)
	s.items[newKey] = item
	return nil
}

func (s *envStore) LoadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("line %d is missing '='", lineNo)
		}
		key = strings.TrimSpace(key)
		value := strings.TrimSpace(rawValue)
		value = strings.Trim(value, "\"")
		if err := s.Set(key, value, sourceProcess); err != nil {
			return fmt.Errorf("line %d: %w", lineNo, err)
		}
	}
	return scanner.Err()
}

func (s *envStore) SaveDotEnv(path string) error {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	keys := s.KeysFiltered("", sourceAll)
	for _, key := range keys {
		value := strings.ReplaceAll(s.items[key].Value, "\n", "\\n")
		if strings.ContainsAny(value, " #") {
			value = fmt.Sprintf("\"%s\"", strings.ReplaceAll(value, "\"", "\\\""))
		}
		if _, err := fmt.Fprintf(w, "%s=%s\n", key, value); err != nil {
			return err
		}
	}
	return w.Flush()
}

func (s *envStore) Item(key string) (envItem, bool) {
	item, ok := s.items[key]
	return item, ok
}

func main() {
	icon := fyne.NewStaticResource("app-icon.svg", appIconSVG)

	a := app.NewWithID("se.envedit.app")
	a.SetIcon(icon)

	w := a.NewWindow("Env Edit")
	w.SetIcon(icon)
	w.Resize(fyne.NewSize(1100, 760))

	store := newEnvStore()
	selectedKey := ""

	status := widget.NewLabel("Ready")
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search key or value...")

	sourceFilter := widget.NewRadioGroup([]string{sourceAll, sourceUser, sourceSystem, sourceProcess}, nil)
	sourceFilter.Horizontal = true
	sourceFilter.SetSelected(sourceAll)

	keyEntry := widget.NewEntry()
	scopeSelect := widget.NewSelect([]string{sourceUser, sourceSystem, sourceProcess}, nil)
	scopeSelect.SetSelected(sourceProcess)

	valueEntry := widget.NewMultiLineEntry()
	valueEntry.Wrapping = fyne.TextWrapWord
	valueEntry.SetMinRowsVisible(14)

	visibleKeys := func() []string {
		return store.KeysFiltered(searchEntry.Text, sourceFilter.Selected)
	}

	keyList := widget.NewList(
		func() int { return len(visibleKeys()) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			keys := visibleKeys()
			if id < 0 || id >= len(keys) {
				obj.(*widget.Label).SetText("")
				return
			}

			item, _ := store.Item(keys[id])
			obj.(*widget.Label).SetText(fmt.Sprintf("%s [%s]", keys[id], item.Source))
		},
	)

	resetEditor := func() {
		selectedKey = ""
		keyEntry.SetText("")
		valueEntry.SetText("")
		scopeSelect.SetSelected(defaultSourceForFilter(sourceFilter.Selected))
		keyList.UnselectAll()
	}

	refreshList := func() {
		keyList.Refresh()
	}

	keyList.OnSelected = func(id widget.ListItemID) {
		keys := visibleKeys()
		if id < 0 || id >= len(keys) {
			return
		}

		selectedKey = keys[id]
		item, _ := store.Item(selectedKey)
		keyEntry.SetText(selectedKey)
		valueEntry.SetText(item.Value)
		scopeSelect.SetSelected(normalizeEditableSource(item.Source))
		status.SetText("Editing: " + selectedKey)
	}

	saveCurrent := func() {
		key := strings.TrimSpace(keyEntry.Text)
		value := valueEntry.Text
		scope := normalizeEditableSource(scopeSelect.Selected)
		if key == "" {
			status.SetText("Key cannot be empty")
			return
		}

		var err error
		if selectedKey == "" {
			err = store.Set(key, value, scope)
		} else {
			err = store.Rename(selectedKey, key, value, scope)
		}
		if err != nil {
			status.SetText("Error: " + err.Error())
			return
		}

		selectedKey = key
		if item, ok := store.Item(key); ok {
			scopeSelect.SetSelected(normalizeEditableSource(item.Source))
		}
		refreshList()
		status.SetText("Saved: " + key + " [" + scope + "]")
	}

	newButton := widget.NewButton("New variable", func() {
		resetEditor()
		status.SetText("Create a new variable")
	})

	saveButton := widget.NewButton("Save changes", saveCurrent)

	deleteButton := widget.NewButton("Delete", func() {
		if selectedKey == "" {
			status.SetText("Select a variable to delete")
			return
		}

		confirmKey := selectedKey
		dialog.ShowConfirm("Confirm", "Delete variable '"+confirmKey+"'?", func(ok bool) {
			if !ok {
				return
			}

			store.Delete(confirmKey)
			resetEditor()
			refreshList()
			status.SetText("Deleted: " + confirmKey)
		}, w)
	})

	reloadButton := widget.NewButton("Reload from system", func() {
		store.ReloadFromProcess()
		resetEditor()
		refreshList()
		status.SetText("Reloaded environment variables from the current process")
	})

	importButton := widget.NewButton("Import .env", func() {
		dialog.ShowFileOpen(func(r fyne.URIReadCloser, err error) {
			if err != nil {
				status.SetText("File selection error: " + err.Error())
				return
			}
			if r == nil {
				return
			}

			path := r.URI().Path()
			_ = r.Close()
			if err := store.LoadDotEnv(path); err != nil {
				status.SetText("Import failed: " + err.Error())
				return
			}

			refreshList()
			status.SetText("Imported: " + filepath.Base(path))
		}, w)
	})

	exportButton := widget.NewButton("Export .env", func() {
		dialog.ShowFileSave(func(wc fyne.URIWriteCloser, err error) {
			if err != nil {
				status.SetText("File selection error: " + err.Error())
				return
			}
			if wc == nil {
				return
			}

			path := wc.URI().Path()
			_ = wc.Close()
			if err := store.SaveDotEnv(path); err != nil {
				status.SetText("Export failed: " + err.Error())
				return
			}

			status.SetText("Exported: " + filepath.Base(path))
		}, w)
	})

	searchEntry.OnChanged = func(_ string) {
		refreshList()
	}

	sourceFilter.OnChanged = func(_ string) {
		if selectedKey == "" {
			scopeSelect.SetSelected(defaultSourceForFilter(sourceFilter.Selected))
		}
		refreshList()
	}

	buttons := container.NewGridWithColumns(3,
		newButton,
		saveButton,
		deleteButton,
	)

	actions := container.NewGridWithColumns(3,
		reloadButton,
		importButton,
		exportButton,
	)

	editorTop := container.NewVBox(
		widget.NewLabel("Key"),
		keyEntry,
		widget.NewLabel("Scope"),
		scopeSelect,
		buttons,
	)

	editorBottom := container.NewVBox(
		widget.NewLabel("Value"),
		valueEntry,
	)

	editorSplit := container.NewVSplit(editorTop, editorBottom)
	editorSplit.Offset = 0.24

	left := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Variables"),
			searchEntry,
			sourceFilter,
		),
		nil,
		nil,
		nil,
		keyList,
	)

	split := container.NewHSplit(left, editorSplit)
	split.Offset = 0.4

	content := container.NewBorder(
		nil,
		container.NewVBox(actions, status),
		nil,
		nil,
		split,
	)

	w.SetContent(content)
	w.ShowAndRun()
}
