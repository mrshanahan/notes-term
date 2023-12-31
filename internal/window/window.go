package window

import (
    // "errors"
    "fmt"
    "os"
    "strings"
    // term "golang.org/x/term"
    termios "github.com/pkg/term/termios"
    unix "golang.org/x/sys/unix"

    "github.com/mrshanahan/notes-api/pkg/notes"
    "mrshanahan.com/notes-term/internal/util"
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
    Debug = false
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
    CustomBordering []int
}

type MainWindow struct {
    Window
    Selection int
    Notes []*notes.Note
    LastKeyWindow *TextLabel
    HelpWindow *MultilineTextLabel
    HelpCollapsedLabel *TextLabel
    HelpCollapsed bool
}

func NewMainWindow(termw, termh int, notes []*notes.Note) *MainWindow {
    // TODO: This is nasty. Make this all constructable at once & w/o repeating
    //       the logic of GetTextBounds() in multiple places.
    window := &MainWindow{Window{0, 0, termw, termh, true, []int{}}, 0, notes, nil, nil, nil, true}
    _, rowmax, colmin, colmax := window.GetTextBounds()

    lastkeyw, lastkeyh := 22, 3
    lastkeyx, lastkeyy := colmax-lastkeyw+1, rowmax-lastkeyh+1
    lastkeyBordering := []int{
        BOX_DOUBLE_UPPER_LEFT,
        BOX_DOUBLE_VERTICAL_LEFT,
        BOX_DOUBLE_HORIZONTAL_UP,
        BOX_DOUBLE_LOWER_RIGHT,
    }
    lastKey := NewSizedBorderedTextLabel(lastkeyx, lastkeyy, lastkeyw, lastkeyh, "", lastkeyBordering)
    window.LastKeyWindow = lastKey

    helpText := []string{
        "j/k       Up/down",
        "CTRL+N    Create note",
        "CTRL+R    Rename note",
        "CTRL+D    Delete note",
        "CTRL+I    Import note",
        "Enter     Edit note",
        "q/CTRL+C  Exit",
    }
    helpw := len(util.MaxBy(helpText, func (x string) int { return len(x) }).Value) + 2
    helph := len(helpText) + 2
    helpx, helpy := colmin-2, rowmax-helph+1
    helpBordering := []int{
        BOX_DOUBLE_VERTICAL_RIGHT,
        BOX_DOUBLE_UPPER_RIGHT,
        BOX_DOUBLE_LOWER_LEFT,
        BOX_DOUBLE_HORIZONTAL_UP,
    }
    helpWindow := NewSizedBorderedMultilineTextLabel(helpx, helpy, helpw, helph, helpText, helpBordering)
    window.HelpWindow = helpWindow

    collapseText := "CTRL+H to open help"
    collapsew, collapseh := len(collapseText)+2, 3
    collapsex, collapsey := colmin-2, rowmax-collapseh+1
    collapseLabel := NewSizedBorderedTextLabel(collapsex, collapsey, collapsew, collapseh, collapseText, helpBordering)
    window.HelpCollapsedLabel = collapseLabel

    return window
}

type TextLabel struct {
    Window
    Value string
}

func NewTextLabel(x, y int, value string) *TextLabel {
    return &TextLabel{Window{x, y, len(value), 1, false, []int{}}, value}
}

func NewBorderedTextLabel(x, y int, value string) *TextLabel {
    return &TextLabel{Window{x, y, len(value)+2, 3, true, []int{}}, value}
}

func NewSizedBorderedTextLabel(x, y, w, h int, value string, bordering []int) *TextLabel {
    return &TextLabel{Window{x, y, w, h, true, bordering}, value}
}

type MultilineTextLabel struct {
    Window
    Value []string
}

func NewSizedBorderedMultilineTextLabel(x, y, w, h int, value []string, bordering []int) *MultilineTextLabel {
    return &MultilineTextLabel{Window{x, y, w, h, true, bordering}, value}
}

type TextInput struct {
    Window
    Value string
}

func NewTextInput(x, y, w int, value string) *TextInput {
    return &TextInput{Window{x, y, w, 3, true, []int{}}, value}
}

type Button struct {
    Window
    Text string
    Pressed bool
    IsSelected bool
}

func NewButton(x, y, w, h int, text string) *Button {
    return &Button{Window{x, y, w, h, true, []int{}}, text, false, false}
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
    input := util.LEBytesToUInt32(inputBuf)
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

    var cornul, cornur, cornll, cornlr int
    if len(window.CustomBordering) == 4 {
        cornul, cornur, cornll, cornlr = window.CustomBordering[0], window.CustomBordering[1], window.CustomBordering[2], window.CustomBordering[3]
    } else {
        cornul, cornur, cornll, cornlr = BOX_DOUBLE_UPPER_LEFT, BOX_DOUBLE_UPPER_RIGHT, BOX_DOUBLE_LOWER_LEFT, BOX_DOUBLE_LOWER_RIGHT
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
    DrawChar(bordrowmin, bordcolmin, cornul)
    DrawChar(bordrowmin, bordcolmax, cornur)
    DrawChar(bordrowmax, bordcolmin, cornll)
    DrawChar(bordrowmax, bordcolmax, cornlr)
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
    if Debug {
        window.LastKeyWindow.Draw()
    }
    if window.HelpCollapsed {
        window.HelpCollapsedLabel.Draw()
    } else {
        window.HelpWindow.Draw()
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

func (label *MultilineTextLabel) Draw() {
    label.DrawBorders()
    label.DrawInterior()
    ymin, _, xmin, _ := label.GetTextBounds()
    y := ymin
    for _, l := range label.Value {
        DrawString(y, xmin, l)
        y += 1
    }
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
