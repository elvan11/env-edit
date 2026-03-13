package main

import (
	"bufio"
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

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type envStore struct {
	values map[string]string
}

func newEnvStore() *envStore {
	s := &envStore{values: make(map[string]string)}
	s.ReloadFromProcess()
	return s
}

func (s *envStore) ReloadFromProcess() {
	s.values = make(map[string]string)
	for _, pair := range os.Environ() {
		key, value, ok := strings.Cut(pair, "=")
		if ok {
			s.values[key] = value
		}
	}
}

func (s *envStore) KeysFiltered(filter string) []string {
	filter = strings.ToLower(strings.TrimSpace(filter))
	keys := make([]string, 0, len(s.values))
	for key, value := range s.values {
		if filter == "" || strings.Contains(strings.ToLower(key), filter) || strings.Contains(strings.ToLower(value), filter) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func (s *envStore) Set(key, value string) error {
	key = strings.TrimSpace(key)
	if !envKeyPattern.MatchString(key) {
		return errors.New("ogiltigt variabelnamn")
	}
	s.values[key] = value
	return nil
}

func (s *envStore) Delete(key string) {
	delete(s.values, key)
}

func (s *envStore) Rename(oldKey, newKey, value string) error {
	newKey = strings.TrimSpace(newKey)
	if !envKeyPattern.MatchString(newKey) {
		return errors.New("ogiltigt variabelnamn")
	}
	if oldKey != newKey {
		if _, exists := s.values[newKey]; exists {
			return errors.New("variabeln finns redan")
		}
		delete(s.values, oldKey)
	}
	s.values[newKey] = value
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
			return fmt.Errorf("rad %d saknar '='", lineNo)
		}
		key = strings.TrimSpace(key)
		value := strings.TrimSpace(rawValue)
		value = strings.Trim(value, "\"")
		if err := s.Set(key, value); err != nil {
			return fmt.Errorf("rad %d: %w", lineNo, err)
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
	keys := s.KeysFiltered("")
	for _, key := range keys {
		value := strings.ReplaceAll(s.values[key], "\n", "\\n")
		if strings.ContainsAny(value, " #") {
			value = fmt.Sprintf("\"%s\"", strings.ReplaceAll(value, "\"", "\\\""))
		}
		if _, err := fmt.Fprintf(w, "%s=%s\n", key, value); err != nil {
			return err
		}
	}
	return w.Flush()
}

func main() {
	a := app.NewWithID("se.envedit.app")
	w := a.NewWindow("Env Edit")
	w.Resize(fyne.NewSize(980, 620))

	store := newEnvStore()
	selectedKey := ""

	status := widget.NewLabel("Klar")
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Sök nyckel eller värde…")

	keyEntry := widget.NewEntry()
	valueEntry := widget.NewMultiLineEntry()
	valueEntry.Wrapping = fyne.TextWrapWord

	keyList := widget.NewList(
		func() int { return len(store.KeysFiltered(searchEntry.Text)) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			keys := store.KeysFiltered(searchEntry.Text)
			if id < 0 || id >= len(keys) {
				obj.(*widget.Label).SetText("")
				return
			}
			obj.(*widget.Label).SetText(keys[id])
		},
	)

	resetEditor := func() {
		selectedKey = ""
		keyEntry.SetText("")
		valueEntry.SetText("")
		keyList.UnselectAll()
	}

	refreshList := func() {
		keyList.Refresh()
	}

	keyList.OnSelected = func(id widget.ListItemID) {
		keys := store.KeysFiltered(searchEntry.Text)
		if id < 0 || id >= len(keys) {
			return
		}
		selectedKey = keys[id]
		keyEntry.SetText(selectedKey)
		valueEntry.SetText(store.values[selectedKey])
		status.SetText("Redigerar: " + selectedKey)
	}

	saveCurrent := func() {
		key := strings.TrimSpace(keyEntry.Text)
		value := valueEntry.Text
		if key == "" {
			status.SetText("Nyckel får inte vara tom")
			return
		}
		var err error
		if selectedKey == "" {
			err = store.Set(key, value)
		} else {
			err = store.Rename(selectedKey, key, value)
		}
		if err != nil {
			status.SetText("Fel: " + err.Error())
			return
		}
		selectedKey = key
		refreshList()
		status.SetText("Sparad: " + key)
	}

	newButton := widget.NewButton("Ny variabel", func() {
		resetEditor()
		status.SetText("Skapa ny variabel")
	})

	saveButton := widget.NewButton("Spara ändring", saveCurrent)

	deleteButton := widget.NewButton("Ta bort", func() {
		if selectedKey == "" {
			status.SetText("Välj en variabel att ta bort")
			return
		}
		confirmKey := selectedKey
		dialog.ShowConfirm("Bekräfta", "Ta bort variabeln '"+confirmKey+"'?", func(ok bool) {
			if !ok {
				return
			}
			store.Delete(confirmKey)
			resetEditor()
			refreshList()
			status.SetText("Borttagen: " + confirmKey)
		}, w)
	})

	reloadButton := widget.NewButton("Läs om från system", func() {
		store.ReloadFromProcess()
		resetEditor()
		refreshList()
		status.SetText("Läste in miljövariabler från nuvarande process")
	})

	importButton := widget.NewButton("Importera .env", func() {
		dialog.ShowFileOpen(func(r fyne.URIReadCloser, err error) {
			if err != nil {
				status.SetText("Fel vid filval: " + err.Error())
				return
			}
			if r == nil {
				return
			}
			path := r.URI().Path()
			_ = r.Close()
			if err := store.LoadDotEnv(path); err != nil {
				status.SetText("Import misslyckades: " + err.Error())
				return
			}
			refreshList()
			status.SetText("Importerade: " + filepath.Base(path))
		}, w)
	})

	exportButton := widget.NewButton("Exportera .env", func() {
		dialog.ShowFileSave(func(wc fyne.URIWriteCloser, err error) {
			if err != nil {
				status.SetText("Fel vid filval: " + err.Error())
				return
			}
			if wc == nil {
				return
			}
			path := wc.URI().Path()
			_ = wc.Close()
			if err := store.SaveDotEnv(path); err != nil {
				status.SetText("Export misslyckades: " + err.Error())
				return
			}
			status.SetText("Exporterade: " + filepath.Base(path))
		}, w)
	})

	searchEntry.OnChanged = func(_ string) {
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

	editor := container.NewBorder(
		container.NewVBox(widget.NewLabel("Nyckel"), keyEntry),
		buttons,
		nil,
		nil,
		container.NewVBox(widget.NewLabel("Värde"), valueEntry),
	)

	left := container.NewBorder(
		container.NewVBox(widget.NewLabel("Variabler"), searchEntry),
		nil,
		nil,
		nil,
		keyList,
	)

	split := container.NewHSplit(left, editor)
	split.Offset = 0.38

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
