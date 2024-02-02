package window

import (
    "errors"
    "fmt"

    "mrshanahan.com/notes-term/internal/util"
)

type Modal struct {
    Window
    Title *TextLabel
    Fields []*ModalField
    OK *Button
    Cancel *Button
    Selection *ModalInputSelection
}

type OptionModal struct {
    Window
    Title *TextLabel
    Options []*Button
    Selection *ModalOptionSelection
}

type ModalField struct {
    Label *TextLabel
    Input *TextInput
}

// TODO: Move selection object out of Modal & manage it separately
//       from Modal state, e.g. in event loop
type ModalInputSelection struct {
    NumFields int
    CurrentField int
    OKSelected bool
    CancelSelected bool
}

// TODO: Merge these two selection types
type ModalOptionSelection struct {
    NumOptions int
    CurrentOption int
}

func NewModalInputSelection(numFields int) *ModalInputSelection {
    var curfield int
    if numFields > 0 {
        curfield = 0
    } else {
        curfield = -1
    }
    return &ModalInputSelection{
        NumFields: numFields,
        CurrentField: curfield,
        OKSelected: numFields <= 0,
        CancelSelected: false,
    }
}

func NewModalOptionSelection(numOptions int) *ModalOptionSelection {
    return &ModalOptionSelection{
        numOptions,
        0,
    }
}

// TODOs:
// - Extract Modal into an interface
// - Smarter tiling - max 3 per row but use squares if possible (e.g. 2x2)
func NewOptionModal(window *Window, title string, options []string) *OptionModal {
    // Find max width of buttons
    rowmin, rowmax, colmin, colmax := window.GetTextBounds()

    buttonxbuf, buttonybuf := 5, 2
    buttonh := 3
    var buttonw int
    if len(options) > 0 {
        buttonw = len(util.MaxBy(options, func(f string) int { return len(f) }).Value)
    } else {
        buttonw = len(title)
    }

    // Tile buttons in rows of 3
    buttonsperrow := 3
    numrows := (len(options)-1) / buttonsperrow
    modalh := numrows * (buttonh + buttonybuf) + buttonybuf
    modalw := util.Min(len(options), buttonsperrow).Value * (buttonw + buttonxbuf)

    modalx := (colmax - colmin - modalw) / 2
    modaly := (rowmax - rowmin - modalh) / 2

    titlex, titley := modalx+2, modaly+2
    titleLabel := NewTextLabel(titlex, titley, title)

    firstbuttonx := modalx + buttonxbuf
    firstbuttony := modaly + buttonybuf
    buttons := make([]*Button, len(options))
    for i, o := range options {
        buttonx := firstbuttonx + ((buttonw + buttonxbuf) * (i % buttonsperrow))
        buttony := firstbuttony + ((buttonh + buttonybuf) * (i / buttonsperrow))
        buttons[i] = NewButton(buttonx, buttony, buttonw, buttonh, o)
    }

    selection := NewModalOptionSelection(len(options))
    modal := &OptionModal{
        Window{modalx, modaly, modalw, modalh, true, []int{}},
        titleLabel,
        buttons,
        selection,
    }
    return modal
}

func (modal *OptionModal) Draw() {
    modal.DrawBorders()
    modal.DrawInterior()
    modal.Title.Draw()
    for _, mo := range modal.Options {
        mo.Draw()
    }
}

func (m *OptionModal) GetOptionValue() int {
    return m.Selection.CurrentOption
}

func (m *OptionModal) Validate() error {
    if m.Selection.CurrentOption >= m.Selection.NumOptions || m.Selection.CurrentOption < 0 {
        return errors.New("invalid button selection! either you really fucked up or I did")
    }
    return nil
}

func (m *OptionModal) Save() error {
    return nil
}

func (modal *OptionModal) UpdateFromSelection() {
    selection := modal.Selection
    for i, b := range modal.Options {
        b.IsSelected = i == selection.CurrentOption
    }
    modal.Draw()
}

func (s *ModalOptionSelection) SelectNext() {
    s.CurrentOption = (s.CurrentOption + 1) % s.NumOptions
}

func (s *ModalOptionSelection) SelectPrev() {
    s.CurrentOption = (s.CurrentOption + s.NumOptions - 1) % s.NumOptions
}

func (modal *OptionModal) GetCurrentOption() *Button {
    return modal.Options[modal.Selection.CurrentOption]
}

func (modal *OptionModal) ResetSelection() {
    modal.Selection.CurrentOption = 0
    modal.UpdateFromSelection()
}

func IsAscii(inp uint32) bool {
    return inp <= 127
}

func OptionModalEventLoop(main *MainWindow, modal *OptionModal) bool {
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
                err := modal.Validate()
                if err == nil {
                    err = modal.Save()
                }
                if err == nil {
                    return true
                }
                modal.ShowErrorBox(err)
                modal.Draw()
                // TODO: reset input to invalid field?
                modal.ResetSelection()
        case 0x09: // TAB
            modal.Selection.SelectNext()
            modal.UpdateFromSelection()
        case 0x5a5b1b: // SHIFT+TAB
            modal.Selection.SelectPrev()
            modal.UpdateFromSelection()
        }
    }
}

func NewInputModal(window *Window, title string, okLabel string, cancelLabel string, fields map[string]string) *Modal {
    rowmin, rowmax, colmin, colmax := window.GetTextBounds()
    minvaluew := 80
    var maxdescw int
    if len(fields) > 0 {
        maxdescw = len(util.MaxBy(util.Keys[string, string](fields), func(f string) int { return len(f) }).Value)
    } else {
        maxdescw = len(title)
    }

    fieldh := 6
    modalw, modalh := util.Max(minvaluew, maxdescw).Value, (len(fields) + 1) * fieldh + 3
    modalx := (colmax - colmin - modalw) / 2
    modaly := (rowmax - rowmin - modalh) / 2

    titlex, titley := modalx+2, modaly+2
    titleLabel := NewTextLabel(titlex, titley, title)

    fieldy, fieldx := titley+2, titlex+1
    modalFields := make([]*ModalField, len(fields))
    i := 0
    for k, v := range fields {
        modalFields[i] = &ModalField{
            NewTextLabel(fieldx, fieldy, k),
            NewTextInput(fieldx, fieldy+1, 70, v),
        }
        fieldy += fieldh
        i += 1
    }

    buttonbuf := 5
    buttony, buttonw, buttonh := fieldy, 20, 3
    okx := modalx + buttonbuf
    cancelx := modalx + modalw - buttonw - buttonbuf
    okButton := NewButton(okx, buttony, buttonw, buttonh, okLabel)
    cancelButton := NewButton(cancelx, buttony, buttonw, buttonh, cancelLabel)
    selection := NewModalInputSelection(len(modalFields))
    modal := &Modal{
        Window{modalx, modaly, modalw, modalh, true, []int{}},
        titleLabel,
        modalFields,
        okButton,
        cancelButton,
        selection,
    }
    return modal
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

// TODO: Optional title
func (window *Window) ShowErrorBox(err error) {
    // TODO: Line splitting/some control over display
    SetPalette(ErrorPalette)
    defer SetPalette(DefaultPalette)

    HideCursor()
    defer ShowCursor()

    rowmin, rowmax, colmin, colmax := window.GetTextBounds()

    errString := fmt.Sprintf("%s", err)

    x, y, w, h := colmin, rowmin, colmax-colmin-3, rowmax-rowmin-3
    label := NewSizedBorderedTextLabel(x, y, w, h, errString, []int{})
    label.Draw()
    ReadInput()
}

// TODO: Optional title
func (window *Window) ShowInfoBox(msg string) {
    // TODO: Line splitting/some control over display
    // SetPalette(ErrorPalette)
    // defer SetPalette(DefaultPalette)

    HideCursor()
    defer ShowCursor()

    rowmin, rowmax, colmin, colmax := window.GetTextBounds()

    x, y, w, h := colmin, rowmin, colmax-colmin-3, rowmax-rowmin-3
    label := NewSizedBorderedTextLabel(x, y, w, h, msg, []int{})
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
    // TODO: Validate all fields each go
    for _, mf := range m.Fields {
        if len(mf.Input.Value) == 0 {
            return fmt.Errorf("non-empty value required for field %s", mf.Label.Value)
        }
    }
    return nil
}

func (m *Modal) Save() error {
    return nil
}

func (modal *Modal) UpdateFromSelection() {
    selection := modal.Selection
    if selection.CurrentField >= 0 {
        modal.OK.IsSelected = false
        modal.Cancel.IsSelected = false
        modal.Draw()
        modal.SelectField(selection.CurrentField)
        ShowCursor()
    } else if selection.OKSelected {
        modal.OK.IsSelected = true
        modal.Cancel.IsSelected = false
        HideCursor()
        modal.Draw()
    } else {
        modal.Cancel.IsSelected = true
        modal.OK.IsSelected = false
        HideCursor()
        modal.Draw()
    }
}

func (s *ModalInputSelection) SelectNext() {
    if s.NumFields > 0 {
        if s.CurrentField >= s.NumFields-1 {
            s.CurrentField, s.OKSelected = -1, true
        } else if s.OKSelected {
            s.OKSelected, s.CancelSelected = false, true
        } else if s.CancelSelected {
            s.CancelSelected, s.CurrentField = false, 0
        } else {
            s.CurrentField += 1
        }
    } else {
        if s.OKSelected {
            s.OKSelected, s.CancelSelected = false, true
        } else {
            s.OKSelected, s.CancelSelected = true, false
        }
    }
}

func (s *ModalInputSelection) SelectPrev() {
    if s.NumFields > 0 {
        if s.CurrentField == 0 {
            s.CurrentField, s.CancelSelected = -1, true
        } else if s.CancelSelected {
            s.CancelSelected, s.OKSelected = false, true
        } else if s.OKSelected {
            s.OKSelected, s.CurrentField = false, s.NumFields-1
        } else {
            s.CurrentField -= 1
        }
    } else {
        if s.OKSelected {
            s.OKSelected, s.CancelSelected = false, true
        } else {
            s.OKSelected, s.CancelSelected = true, false
        }
    }
}

func (modal *Modal) GetCurrentField() *ModalField {
    if modal.Selection.CurrentField >= 0 &&
       !modal.Selection.OKSelected &&
       !modal.Selection.CancelSelected {
        return modal.Fields[modal.Selection.CurrentField]
    }
    return nil
}

func (modal *Modal) ResetSelection() {
    if modal.Selection.NumFields > 0 {
        modal.Selection.CurrentField = 0
        modal.Selection.OKSelected, modal.Selection.CancelSelected = false, false
    } else {
        modal.Selection.OKSelected, modal.Selection.CancelSelected = true, false
    }
    modal.UpdateFromSelection()
}

func ModalEventLoop(main *MainWindow, modal *Modal) bool {
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
            if modal.OK.IsSelected || modal.GetCurrentField() != nil {
                err := modal.Validate()
                if err == nil {
                    err = modal.Save()
                }
                if err == nil {
                    return true
                }
                modal.ShowErrorBox(err)
                modal.Draw()
                // TODO: reset input to invalid field?
                modal.UpdateFromSelection()
            } else if modal.Cancel.IsSelected {
                return false
            }
        case 0x09: // TAB
            modal.Selection.SelectNext()
            modal.UpdateFromSelection()
        case 0x5a5b1b: // SHIFT+TAB
            modal.Selection.SelectPrev()
            modal.UpdateFromSelection()
        case 0x7f: // Backspace
            field := modal.GetCurrentField()
            if field != nil && len(field.Input.Value) > 0 {
                field.Input.Value = field.Input.Value[:len(field.Input.Value)-1]
                field.Input.Draw()
                modal.UpdateFromSelection()
            }
        default:
            field := modal.GetCurrentField()
            if field != nil && IsAscii(input) {
                field.Input.Value = fmt.Sprintf("%s%c", field.Input.Value, input)
                field.Input.Draw()
                modal.UpdateFromSelection()
            }
        }
    }
}

func (window *MainWindow) RequestInput(title string, fields []string) map[string]string {
    fieldsWithDefaults := map[string]string{}
    for _, f := range fields {
        fieldsWithDefaults[f] = ""
    }
    return window.RequestInputWithDefaults(title, fieldsWithDefaults)
}

func (window *MainWindow) RequestInputWithDefaults(title string, fields map[string]string) map[string]string {
    modal := NewInputModal(&window.Window, title, "OK", "Cancel", fields)
    modal.Draw()
    defer window.Draw()

    modal.UpdateFromSelection()

    ShowCursor()
    defer HideCursor()

    success := ModalEventLoop(window, modal)

    if success {
        return modal.GetFieldValues()
    } else {
        return nil
    }
}

func (window *MainWindow) RequestConfirmation(prompt string) bool {
    return window.RequestConfirmationWithButtons(prompt, "OK", "Cancel")
}

func (window *MainWindow) RequestConfirmationWithButtons(prompt string, ok string, cancel string) bool {
    modal := NewInputModal(&window.Window, prompt, ok, cancel, map[string]string{})
    modal.Draw()
    defer window.Draw()

    modal.UpdateFromSelection()

    success := ModalEventLoop(window, modal)
    return success
}

func (window *MainWindow) RequestOptionSelection(prompt string, options []string) int {
    modal := NewOptionModal(&window.Window, prompt, options)
    modal.Draw()
    defer window.Draw()

    modal.UpdateFromSelection()

    success := OptionModalEventLoop(window, modal)
    if success {
        return modal.GetOptionValue()
    }
    return -1
}