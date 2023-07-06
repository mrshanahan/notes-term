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
    LastKeyWindow *TextLabel
}

func NewMainWindow(termw, termh int, notes []*IndexEntry) *MainWindow {
    // TODO: This is nasty. Make this all constructable at once & w/o repeating
    //       the logic of GetTextBounds() in multiple places.
    window := &MainWindow{Window{0, 0, termw, termh, true}, 0, notes, nil}
    _, rowmax, _, colmax := window.GetTextBounds()

    boxw, boxh := 22, 3
    boxx, boxy := colmax-boxw+1, rowmax-boxh+1
    lastKey := NewSizedBorderedTextLabel(boxx, boxy, boxw, boxh, "")
    window.LastKeyWindow = lastKey

    return window
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

func NewTextInput(x, y, w int, value string) *TextInput {
    return &TextInput{Window{x, y, w, 3, true}, value}
}

type Button struct {
    Window
    Text string
    Pressed bool
    IsSelected bool
}

func NewButton(x, y, w, h int, text string) *Button {
    return &Button{Window{x, y, w, h, true}, text, false, false}
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
    if DEBUG {
        window.LastKeyWindow.Draw()
    }

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

func (b *Button) Draw() {
    if b.IsSelected {
        SetPalette(HighlightPalette)
        defer SetPalette(DefaultPalette)
    }

    y, _, xmin, xmax := b.GetTextBounds()
    textw := xmax - xmin + 1
    b.DrawBorders()
    strlen := len(b.Text)
    buflen := textw - strlen
    var bufleft, bufright int
    if buflen < 0 {
        bufleft, bufright = 0, 0
    } else if buflen % 2 == 0 {
        bufleft, bufright = buflen / 2, buflen / 2
    } else {
        bufleft, bufright = buflen / 2, buflen / 2 + 1
    }
    buftext := fmt.Sprintf("%s%s%s",
        strings.Repeat(" ", bufleft),
        b.Text,
        strings.Repeat(" ", bufright))
    DrawString(y, xmin, buftext)
}
