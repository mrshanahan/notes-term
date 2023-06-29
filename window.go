package main

import (
    // "errors"
    "fmt"
    "os"
    "strings"
    // term "golang.org/x/term"
    termios "github.com/pkg/term/termios"
    unix "golang.org/x/sys/unix"
)

const (
    BOX_DOUBLE_HORIZONTAL = '\u2550'
    BOX_DOUBLE_VERTICAL = '\u2551'
    BOX_DOUBLE_UPPER_RIGHT = '\u2557'
    BOX_DOUBLE_LOWER_RIGHT = '\u255D'
    BOX_DOUBLE_LOWER_LEFT = '\u255A'
    BOX_DOUBLE_UPPER_LEFT = '\u2554'
    BOX_DOUBLE_VERTICAL_RIGHT = '\u2560'
    BOX_DOUBLE_VERTICAL_LEFT = '\u2563'
    BOX_DOUBLE_HORIZONTAL_DOWN = '\u2566'
    BOX_DOUBLE_HORIZONTAL_UP = '\u2569'

    DEFAULT_BACKGROUND_COLOR = 46 // cyan
    DEFAULT_FOREGROUND_COLOR = 37 // white
    HIGHLIGHT_BACKGROUND_COLOR = 47 // white
    HIGHLIGHT_FOREGROUND_COLOR = 36 // cyan
    ERROR_BACKGROUND_COLOR = 41 // red
    ERROR_FOREGROUND_COLOR = 37 // white
)

var (
    DefaultPalette = &Palette{
        DEFAULT_BACKGROUND_COLOR,
        DEFAULT_FOREGROUND_COLOR,
    }
    HighlightPalette = &Palette{
        HIGHLIGHT_BACKGROUND_COLOR,
        HIGHLIGHT_FOREGROUND_COLOR,
    }
    ErrorPalette = &Palette{
        ERROR_BACKGROUND_COLOR,
        ERROR_FOREGROUND_COLOR,
    }
)

// TODO: Window as interface

type Palette struct {
    Background int
    Foreground int
}

type Window struct {
    X int
    Y int
    Width int
    Height int
    HasBorders bool
}

type MainWindow struct {
    Window
    Selection int
    Notes []*IndexEntry
    LastKey string
}

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
    modalw, modalh := Max(minvaluew, maxdescw).Value, len(fields) * 6 + 3
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

    buttony, buttonw, buttonh := fieldy, 30, 1
    okx := titlex
    cancelx := modalx + modalw - buttonw - 2
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

type TextLabel struct {
    Window
    Value string
}

func NewTextLabel(x, y int, value string) *TextLabel {
    return &TextLabel{Window{x, y, len(value), 1, false}, value}
}

func NewBorderedTextLabel(x, y int, value string) *TextLabel {
    return &TextLabel{Window{x, y, len(value)+2, 3, true}, value}
}

func NewSizedBorderedTextLabel(x, y, w, h int, value string) *TextLabel {
    return &TextLabel{Window{x, y, w, h, true}, value}
}

type TextInput struct {
    Window
    Value string
}

func NewTextInput(x, y, w int) *TextInput {
    return &TextInput{Window{x, y, w, 3, true}, ""}
}

type Button struct {
    Window
    Text string
    Pressed bool
}

func NewButton(x, y, w, h int, text string) *Button {
    return &Button{Window{x, y, w, h, true}, text, false}
}

func SetPalette(p *Palette) {
    fmt.Printf("\033[%dm", p.Background)
    fmt.Printf("\033[%dm", p.Foreground)
}

func (w Window) GetTextBounds() (int, int, int, int) {
    posmod, sizemod := 0, 0
    if w.HasBorders {
        posmod, sizemod = 2, -1
    }

    return w.Y + posmod,
           w.Y + w.Height + sizemod,
           w.X + posmod,
           w.X + w.Width + sizemod
}

func ReadInput() uint32 {
    inputBuf := make([]byte, 4)
    os.Stdin.Read(inputBuf)
    input := LEBytesToUInt32(inputBuf)
    return input
}

func Move(row, col int) {
    fmt.Printf("\033[%d;%dH", row, col)
}

func ResetBackgroundColor() {
    fmt.Printf("\033[0m")
}

func ClearScreen() {
    fmt.Printf("\033[2J")
}

func HideCursor() {
    fmt.Print("\033[?25l")
}

func ShowCursor() {
    fmt.Print("\033[?25h")
}

func DrawChar(r, c int, b int) {
    Move(r, c)
    fmt.Printf("%c", b)
}

func DrawString(r, c int, s string) {
    Move(r, c)
    fmt.Print(s)
}

func (window Window) DrawBorders() {
    if !window.HasBorders {
        return
    }

    rowmin, rowmax, colmin, colmax := window.GetTextBounds()
    bordrowmin, bordrowmax := rowmin-1, rowmax+1
    bordcolmin, bordcolmax := colmin-1, colmax+1
    for c := bordcolmin+1; c <= bordcolmax-1; c++ {
        DrawChar(bordrowmin, c, BOX_DOUBLE_HORIZONTAL)
        DrawChar(bordrowmax, c, BOX_DOUBLE_HORIZONTAL)
    }
    for r := bordrowmin+1; r <= bordrowmax-1; r++ {
        DrawChar(r, bordcolmin, BOX_DOUBLE_VERTICAL)
        DrawChar(r, bordcolmax, BOX_DOUBLE_VERTICAL)
    }
    DrawChar(bordrowmin, bordcolmin, BOX_DOUBLE_UPPER_LEFT)
    DrawChar(bordrowmin, bordcolmax, BOX_DOUBLE_UPPER_RIGHT)
    DrawChar(bordrowmax, bordcolmin, BOX_DOUBLE_LOWER_LEFT)
    DrawChar(bordrowmax, bordcolmax, BOX_DOUBLE_LOWER_RIGHT)
}

func DrawCornerBox(window *MainWindow) {
    _, rowmax, _, colmax := window.GetTextBounds()
    boxwidth, boxheight := 20, 1
    boxcolmin, boxcolmax := colmax-boxwidth, colmax+1
    boxrowmin, boxrowmax := rowmax-boxheight, rowmax+1
    for c := boxcolmin; c <= boxcolmax; c++ {
        DrawChar(boxrowmin, c, BOX_DOUBLE_HORIZONTAL)
        DrawChar(boxrowmax, c, BOX_DOUBLE_HORIZONTAL)
    }
    for r := boxrowmin; r <= boxrowmax; r++ {
        DrawChar(r, boxcolmin, BOX_DOUBLE_VERTICAL)
        DrawChar(r, boxcolmax, BOX_DOUBLE_VERTICAL)
    }
    DrawChar(boxrowmin, boxcolmin, BOX_DOUBLE_UPPER_LEFT)
    DrawChar(boxrowmin, boxcolmax, BOX_DOUBLE_VERTICAL_LEFT)
    DrawChar(boxrowmax, boxcolmin, BOX_DOUBLE_HORIZONTAL_UP)
    DrawChar(boxrowmax, boxcolmax, BOX_DOUBLE_LOWER_RIGHT)
    DrawString(boxrowmin+1, boxcolmin+2, window.LastKey)
}

func (window *Window) DrawInterior() {
    rowmin, rowmax, colmin, colmax := window.GetTextBounds()
    for r := rowmin; r <= rowmax; r++ {
        for c := colmin; c <= colmax; c++ {
            DrawChar(r, c, ' ')
        }
    }
}

func DrawNoteRow(window *MainWindow, noteIdx int, palette *Palette) {
    rowmin, _, colmin, colmax := window.GetTextBounds()
    if noteIdx < 0 || noteIdx >= len(window.Notes) {
        panic(fmt.Sprintf("attempted to draw nonexistent note index: %d", noteIdx))
    }

    row, col := noteIdx + rowmin, colmin
    Move(row, col)
    SetPalette(palette)
    defer SetPalette(DefaultPalette)
    // fmt.Printf("\033[%dm", palette.Background)
    // fmt.Printf("\033[%dm", palette.Foreground)

    note := window.Notes[noteIdx]
    contents := note.Title
    padding := colmax - rowmin - len(contents) + 1
    if padding < 0 {
        padding = 0
        contents = contents[:colmax-rowmin-2] + "..."
    }
    padstring := strings.Repeat(" ", padding)
    fmt.Printf("%s%s", contents, padstring)

    // fmt.Printf("\033[%dm", DefaultPalette.Background)
    // fmt.Printf("\033[%dm", DefaultPalette.Foreground) }
}

func (window *MainWindow) Draw() {
    window.DrawBorders()
    window.DrawInterior()
    DrawCornerBox(window)

    for i, _ := range window.Notes {
        if i == window.Selection {
            DrawNoteRow(window, i, HighlightPalette)
        } else {
            DrawNoteRow(window, i, DefaultPalette)
        }
    }
}

func DisableEcho(fd uintptr) {
    t := &unix.Termios{}
    err := termios.Tcgetattr(fd, t)
    if err != nil {
        panic(err)
    }
    t.Lflag = t.Lflag & (^uint32(unix.ECHO))
    t.Lflag = t.Lflag & unix.ISIG
    err = termios.Tcsetattr(fd, termios.TCSANOW, t)
    if err != nil {
        panic(err)
    }
}

func LEBytesToUInt32(bs []byte) uint32 {
    i := uint32(bs[3])
    i = i << 8 | uint32(bs[2])
    i = i << 8 | uint32(bs[1])
    i = i << 8 | uint32(bs[0])
    return i
}

func LEBytesToString(bs []byte) string {
    s := "0x"
    for _, b := range bs {
        s = fmt.Sprintf("%s%02x", s, b)
    }
    return s
}

func MaxBy(xs []string, f func(string)int) *Maybe[string] {
    max := 0
    var maxTmp *Maybe[string] = nil
    for _, x := range xs {
        tmp := f(x)
        if maxTmp == nil || tmp > max {
            max, maxTmp = tmp, Just(x)
        }
    }
    return maxTmp
}

func Max(xs ...int) *Maybe[int] {
    var res *Maybe[int]
    for _, x := range xs {
        if res == nil || x > res.Value {
            res = Just(x)
        }
    }
    return res
}

type Maybe[T any] struct {
    Value T
}

func Just[T any](x T) *Maybe[T] {
    return &Maybe[T]{x}
}

func (label *TextLabel) Draw() {
    label.DrawBorders()
    label.DrawInterior()
    ymin, _, xmin, _ := label.GetTextBounds()
    DrawString(ymin, xmin, label.Value)
}

func (inp *TextInput) Draw() {
    y, _, xmin, xmax := inp.GetTextBounds()
    inp.DrawBorders()
    strlen := len(inp.Value)
    DrawString(y, xmin, inp.Value)
    DrawString(y, xmin+strlen, strings.Repeat(" ", xmax-xmin-strlen))
}

func (modal *Modal) Draw() {
    modal.DrawBorders()
    modal.DrawInterior()
    modal.Title.Draw()
    for _, mf := range modal.Fields {
        mf.Label.Draw()
        mf.Input.Draw()
    }
}

func (modal *Modal) SelectField(idx int) {
    fields := modal.Fields
    if idx <= len(fields)-1 {
        f := fields[idx]
        y, _, x, _ := f.Input.GetTextBounds()
        valLen := len(f.Input.Value)
        Move(y, x + valLen)
    } else {
        panic("ahhh")
    }
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
            err := modal.Validate()
            if err == nil {
                err = modal.Save()
            }
            if err == nil {
                return true
            }
            modal.ShowErrorBox(err)
            modal.Draw()
            modal.SelectField(idx)
        case 0x09: // TAB
            if (idx >= len(modal.Fields)-1) {
                idx = 0
            } else {
                idx += 1
            }
            modal.SelectField(idx)
        case 0x5a5b1b: // SHIFT+TAB
            if (idx <= 0) {
                idx = len(modal.Fields)-1
            } else {
                idx -= 1
            }
            modal.SelectField(idx)
        case 0x7f: // Backspace
            fieldInput := modal.Fields[idx].Input
            if len(fieldInput.Value) > 0 {
                fieldInput.Value = fieldInput.Value[:len(fieldInput.Value)-1]
                fieldInput.Draw()
                modal.SelectField(idx)
            }
        default:
            if IsAscii(input) {
                fieldInput := modal.Fields[idx].Input
                fieldInput.Value = fmt.Sprintf("%s%c", fieldInput.Value, input)
                fieldInput.Draw()
                modal.SelectField(idx)
            }
        }
    }
}
