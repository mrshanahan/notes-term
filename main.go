package main

import (
    "fmt"
    "os"
    "os/exec"
    "strings"
    term "golang.org/x/term"
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
    Selection int
    Notes []*IndexEntry
    Width int
    Height int
    LastKey string
}

var (
    DefaultPalette = &Palette{
        DEFAULT_BACKGROUND_COLOR,
        DEFAULT_FOREGROUND_COLOR,
    }
    HighlightPalette = &Palette{
        HIGHLIGHT_BACKGROUND_COLOR,
        HIGHLIGHT_FOREGROUND_COLOR,
    }
)

func SetPalette(p *Palette) {
    fmt.Printf("\033[%dm", p.Background)
    fmt.Printf("\033[%dm", p.Foreground)
}

func (w *Window) GetTextBounds() (int, int, int, int) {
    return 2, w.Height-1, 2, w.Width-1
}

func Move(row, col int) {
    fmt.Printf("\033[%d;%dH", row, col)
}

func SetBackgroundColor() {
    fmt.Printf("\033[%dm", DEFAULT_BACKGROUND_COLOR)
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

func DrawBorders(window *Window) {
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

func DrawCornerBox(window *Window) {
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

func DrawInterior(window *Window) {
    rowmin, rowmax, colmin, colmax := window.GetTextBounds()
    for r := rowmin; r <= rowmax; r++ {
        for c := colmin; c <= colmax; c++ {
            DrawChar(r, c, ' ')
        }
    }
}

func DrawNoteRow(window *Window, noteIdx int, palette *Palette) {
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

func Draw(window *Window) {
    DrawBorders(window)
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

func OpenEditor(path string) {
    cmd := exec.Command("nvim", path)
    cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
    err := cmd.Run()
    if err != nil {
        panic(err)
    }
}

func initState() (*Window, func()) {
    notes := LoadIndex("index.txt")
    if len(notes) == 0 {
        fmt.Println("no notes")
        os.Exit(1)
    }

    fd := os.Stdin.Fd()
    DisableEcho(fd)
    // t.Lflag = t.Lflag | unix.ECHO
    // defer termios.Tcsetattr(fd, termios.TCSANOW, t)

    oldState, err := term.MakeRaw(int(fd))
    if err != nil {
        panic(err)
    }
    // defer term.Restore(int(fd), oldState)

    ClearScreen()
    termw, termh, err := term.GetSize(int(fd))
    if err != nil {
        panic(err)
    }

    HideCursor()
    // defer ShowCursor()

    SetBackgroundColor()
    // defer ResetBackgroundColor()

    window := &Window{0, notes, termw, termh, ""}
    Draw(window)

    return window, func() {
        term.Restore(int(fd), oldState)
        ShowCursor()
        // This might not be needed - everything seems to work w/o it
        // ResetBackgroundColor()
    }
}

func main() {
    window, cleanup := initState()
    defer cleanup()

    var input uint32 = 0
    for input != 'q' {
        // TODO: interrupts
        inputBuf := make([]byte, 4)
        os.Stdin.Read(inputBuf)
        input = LEBytesToUInt32(inputBuf)
        idx := window.Selection
        switch (input) {
        case 'k': // up
            if (idx <= 0) {
                idx = 0
            } else {
                idx -= 1
            }
            break
        case 'j': // down
            if (idx >= len(window.Notes)-1) {
                idx = len(window.Notes)-1
            } else {
                idx += 1
            }
            break
        case '\u000e': // CTRL+n
            break
        case '\u000d': // Enter
            OpenEditor(window.Notes[idx].Path)
            break
        }
        window.LastKey = fmt.Sprintf("0x%x", input)
        window.Selection = idx
        Draw(window)
    }

    Move(0,0)
    ClearScreen()
}
