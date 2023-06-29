package main

import (
    "fmt"
)

type Modal struct {
    Window
    Title *TextLabel
    Fields []*ModalField
    OK *Button
    Cancel *Button
}

func NewModal(window *Window, title string, fields []string) *Modal {
    rowmin, rowmax, colmin, colmax := window.GetTextBounds()
    minvaluew := 80
    maxdescw := len(MaxBy(fields, func(f string) int { return len(f) }).Value)
    modalw, modalh := Max(minvaluew, maxdescw).Value, (len(fields) + 1) * 6 + 3
    modalx := (colmax - colmin - modalw) / 2
    modaly := (rowmax - rowmin - modalh) / 2

    titlex, titley := modalx+2, modaly+2
    titleLabel := NewTextLabel(titlex, titley, title)

    fieldy, fieldx := titley+2, titlex+1
    modalFields := make([]*ModalField, len(fields))
    for i, f := range fields {
        modalFields[i] = &ModalField{
            NewTextLabel(fieldx, fieldy, f),
            NewTextInput(fieldx, fieldy+1, 70),
        }
        fieldy += 6
    }

    buttonbuf := 5
    buttony, buttonw, buttonh := fieldy, 20, 3
    okx := modalx + buttonbuf
    cancelx := modalx + modalw - buttonw - buttonbuf
    okButton := NewButton(okx, buttony, buttonw, buttonh, "OK")
    cancelButton := NewButton(cancelx, buttony, buttonw, buttonh, "Cancel")
    modal := &Modal{
        Window{modalx, modaly, modalw, modalh, true},
        titleLabel,
        modalFields,
        okButton,
        cancelButton,
    }
    return modal
}

type ModalField struct {
    Label *TextLabel
    Input *TextInput
}
func (modal *Modal) Draw() {
    modal.DrawBorders()
    modal.DrawInterior()
    modal.Title.Draw()
    for _, mf := range modal.Fields {
        mf.Label.Draw()
        mf.Input.Draw()
    }
    modal.OK.Draw()
    modal.Cancel.Draw()
}

func (modal *Modal) SelectField(idx int) {
    fields := modal.Fields
    if idx >= 0 && idx <= len(fields)-1 {
        f := fields[idx]
        y, _, x, _ := f.Input.GetTextBounds()
        valLen := len(f.Input.Value)
        Move(y, x + valLen)
    } else {
        panic("ahhh")
    }
}

func (modal *Modal) SelectOKButton() {
}

func (window *MainWindow) RequestInput(title string, fields []string) map[string]string {
    modal := NewModal(&window.Window, title, fields)
    modal.Draw()
    defer window.Draw()

    ShowCursor()
    defer HideCursor()

    success := ModalEventLoop(window, modal)

    if success {
        return modal.GetFieldValues()
    } else {
        return nil
    }
}

func (modal *Modal) ShowErrorBox(err error) {
    // TODO: Line splitting/some control over display
    SetPalette(ErrorPalette)
    defer SetPalette(DefaultPalette)

    HideCursor()
    defer ShowCursor()

    rowmin, rowmax, colmin, colmax := modal.GetTextBounds()

    errString := fmt.Sprintf("%s", err)

    x, y, w, h := colmin, rowmin, colmax-colmin-3, rowmax-rowmin-3
    label := NewSizedBorderedTextLabel(x, y, w, h, errString)
    label.Draw()
    ReadInput()
}

func (m *Modal) GetFieldValues() map[string]string {
    vals := map[string]string{}
    for _, f := range m.Fields {
        vals[f.Label.Value] = f.Input.Value
    }
    return vals
}

func (m *Modal) Validate() error {
    for _, mf := range m.Fields {
        if len(mf.Input.Value) == 0 {
            return fmt.Errorf("Non-empty value required for field %s!", mf.Label.Value)
        }
    }
    return nil
}

func (m *Modal) Save() error {
    return nil
}

func IsAscii(inp uint32) bool {
    return inp <= 127
}

func ModalEventLoop(main *MainWindow, modal *Modal) bool {
    idx := 0
    modal.SelectField(idx)
    var input uint32 = 0
    for {
        // TODO: Others to handle:
        // - CTRL+Backspace
        // - Delete
        // - Terminal movement (CTRL+W, etc.)
        // - Scrolling
        // - Non-ASCII characters (Unicode alphanumeric/spaces?)
        input = ReadInput()
        switch (input) {
        case 0x03, 0x1b: // CTRL+C/ESC
            return false
        case 0x0d: // ENTER
            if modal.OK.IsSelected || (idx >= 0 && idx <= len(modal.Fields)-1) {
                err := modal.Validate()
                if err == nil {
                    err = modal.Save()
                }
                if err == nil {
                    return true
                }
                modal.ShowErrorBox(err)
                modal.Draw()

                if idx >= 0 && idx <= len(modal.Fields)-1 {
                    modal.SelectField(idx)
                }
            } else if modal.Cancel.IsSelected {
                return false
            }
        case 0x09: // TAB
            if modal.OK.IsSelected {
                modal.Cancel.IsSelected, modal.OK.IsSelected = true, false
                HideCursor()
                modal.Draw()
            } else if modal.Cancel.IsSelected {
                idx, modal.Cancel.IsSelected = 0, false
                modal.Draw()
                modal.SelectField(idx)
                ShowCursor()
            } else if (idx >= len(modal.Fields)-1) {
                modal.OK.IsSelected, idx = true, -1
                HideCursor()
                modal.Draw()
            } else {
                idx += 1
                modal.SelectField(idx)
            }
        case 0x5a5b1b: // SHIFT+TAB
            if modal.Cancel.IsSelected {
                modal.OK.IsSelected, modal.Cancel.IsSelected = true, false
                HideCursor()
                modal.Draw()
            } else if modal.OK.IsSelected {
                idx, modal.OK.IsSelected = len(modal.Fields)-1, false
                modal.Draw()
                modal.SelectField(idx)
                ShowCursor()
            } else if idx <= 0 {
                modal.Cancel.IsSelected, idx = true, -1
                HideCursor()
                modal.Draw()
            } else {
                idx -= 1
                modal.SelectField(idx)
            }
        case 0x7f: // Backspace
            fieldInput := modal.Fields[idx].Input
            if len(fieldInput.Value) > 0 {
                fieldInput.Value = fieldInput.Value[:len(fieldInput.Value)-1]
                fieldInput.Draw()
                modal.SelectField(idx)
            }
        default:
            if IsAscii(input) && idx >= 0 && idx <= len(modal.Fields)-1 {
                fieldInput := modal.Fields[idx].Input
                fieldInput.Value = fmt.Sprintf("%s%c", fieldInput.Value, input)
                fieldInput.Draw()
                modal.SelectField(idx)
            }
        }
    }
}
