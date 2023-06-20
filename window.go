package main

import (
    "fmt"
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
)

type Palette struct {
    Background int
    Foreground int
}

type Window struct {
    X int
    Y int
    Width int
    Height int
}

type MainWindow struct {
    Window
    Selection int
    Notes []*IndexEntry
    LastKey string
}

type Modal struct {
    Window
    Title string
    Fields []*ModalField
}

type ModalField struct {
    Label string
    Value string
}

func SetPalette(p *Palette) {
    fmt.Printf("\033[%dm", p.Background)
    fmt.Printf("\033[%dm", p.Foreground)
}

func (w Window) GetTextBounds() (int, int, int, int) {
    return w.Y + 2,
           w.Y + w.Height - 1,
           w.X + 2,
           w.X + w.Width - 1
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

func DrawInterior(window *MainWindow) {
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
    fmt.Printf("\033[%dm", palette.Background)
    fmt.Printf("\033[%dm", palette.Foreground)

    note := window.Notes[noteIdx]
    contents := note.Title
    padding := colmax - rowmin - len(contents) + 1
    if padding < 0 {
        padding = 0
        contents = contents[:colmax-rowmin-2] + "..."
    }
    padstring := strings.Repeat(" ", padding)
    fmt.Printf("%s%s", contents, padstring)

    fmt.Printf("\033[%dm", DefaultPalette.Background)
    fmt.Printf("\033[%dm", DefaultPalette.Foreground)
}

func (window *MainWindow) Draw() {
    window.DrawBorders()
    DrawInterior(window)
    DrawCornerBox(window)

    for i, _ := range window.Notes {
        if i == window.Selection {
            DrawNoteRow(window, i, HighlightPalette)
        } else {
            DrawNoteRow(window, i, DefaultPalette)
        }
    }
}

func (modal *Modal) Draw() {
    modal.DrawBorders()

    rowmin, _, colmin, _ := modal.GetTextBounds()

    titley, titlex := rowmin, colmin
    DrawString(titley, titlex, modal.Title)

    fieldy, fieldx := titley+2, titlex
    for _, mf := range modal.Fields {
        DrawString(fieldy, fieldx, mf.Label)
        DrawString(fieldy+1, fieldx, "xxxxxxxxxxxxxxxxxx")
        fieldy += 3
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

func (window *MainWindow) RequestInput(title string, fields []string) *Modal {
    rowmin, rowmax, colmin, colmax := window.GetTextBounds()

    minvaluew := 40
    maxdescw := len(MaxBy(fields, func(f string) int { return len(f) }).Value)
    modalw, modalh := Max(minvaluew, maxdescw).Value, len(fields) * 4 + 3
    modalx := (colmax - colmin - modalw) / 2
    modaly := (rowmax - rowmin - modalh) / 2

    modalFields := make([]*ModalField, len(fields))
    for i, f := range fields {
        modalFields[i] = &ModalField{f, ""}
    }
    modal := &Modal{
        Window{modalx, modaly, modalw, modalh},
        title,
        modalFields,
    }
    return modal
}
