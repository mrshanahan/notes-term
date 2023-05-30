package main

import (
    "bufio"
    "fmt"
    // "io"
    "os"
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
    Notes []string
    Width int
    Height int
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

func DrawBorders(rowmin, rowmax, colmin, colmax int) {
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

func DrawInterior(rowmin, rowmax, colmin, colmax int) {
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

    contents := window.Notes[noteIdx]
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
    rowmin, rowmax, colmin, colmax := window.GetTextBounds()
    DrawBorders(rowmin, rowmax, colmin, colmax)
    DrawInterior(rowmin, rowmax, colmin, colmax)

    for i, _ := range window.Notes {
        if i == window.Selection {
            DrawNoteRow(window, i, HighlightPalette)
        } else {
            DrawNoteRow(window, i, DefaultPalette)
        }
    }
}

func LoadIndex(path string) []string {
    f, err := os.Open(path)
    if err != nil {
        panic(err)
    }
    defer f.Close()

    lines := []string{}
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        line := scanner.Text()
        lines = append(lines, line)
    }
    if err = scanner.Err(); err != nil {
        panic(err)
    }
    return lines
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

func main() {
    notes := LoadIndex("index.txt")

    fd := os.Stdin.Fd()
    DisableEcho(fd)
    // t.Lflag = t.Lflag | unix.ECHO
    // defer termios.Tcsetattr(fd, termios.TCSANOW, t)

    oldState, err := term.MakeRaw(int(fd))
    if err != nil {
        panic(err)
    }
    defer term.Restore(int(fd), oldState)

    ClearScreen()
    c, b := byte(0), []byte{0}
    termw, termh, err := term.GetSize(int(fd))
    if err != nil {
        panic(err)
    }

    HideCursor()
    defer ShowCursor()

    SetBackgroundColor()
    defer ResetBackgroundColor()

    window := &Window{0, notes, termw, termh}
    Draw(window)

    for c != 'q' {
        // TODO: interrupts
        os.Stdin.Read(b)
        c = b[0]
        idx := window.Selection
        switch (c) {
        case 'k':
            if (idx <= 0) {
                idx = 0
            } else {
                idx -= 1
            }
            break
        case 'j':
            if (idx >= len(window.Notes)-1) {
                idx = len(window.Notes)-1
            } else {
                idx += 1
            }
            break
        }
        window.Selection = idx
        Draw(window)
    }

    Move(0,0)
    ClearScreen()
}
