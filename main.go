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

func HighlightRow(contents string, row, rowmin, colmin, colmax int) {
    Move(row, colmin)
    fmt.Printf("\033[%dm", HIGHLIGHT_BACKGROUND_COLOR)
    fmt.Printf("\033[%dm", HIGHLIGHT_FOREGROUND_COLOR)

    padding := colmax - rowmin - len(contents)
    if padding < 0 {
        padding = 0
        contents = contents[:colmax-rowmin-2] + "..."
    }
    padstring := strings.Repeat(" ", padding)
    fmt.Printf("%s%s", contents, padstring)

    fmt.Printf("\033[%dm", DEFAULT_BACKGROUND_COLOR)
    fmt.Printf("\033[%dm", DEFAULT_FOREGROUND_COLOR)
}

func ResetRow(contents string, row, rowmin, colmin, colmax int) {
    Move(row, colmin)
    fmt.Printf("\033[%dm", DEFAULT_BACKGROUND_COLOR)
    fmt.Printf("\033[%dm", DEFAULT_FOREGROUND_COLOR)

    padding := colmax - rowmin - len(contents)
    if padding < 0 {
        padding = 0
        contents = contents[:colmax-rowmin-2] + "..."
    }
    padstring := strings.Repeat(" ", padding)
    fmt.Printf("%s%s", contents, padstring)
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

func main() {
    notes := LoadIndex("index.txt")

    fd := os.Stdin.Fd()
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

    fmt.Print("\033[?25l")
    defer fmt.Print("\033[?25h")

    SetBackgroundColor()
    defer ResetBackgroundColor()

    rowmin, rowmax, colmin, colmax := 2, termh-1, 2, termw-1
    DrawBorders(rowmin, rowmax, colmin, colmax)
    DrawInterior(rowmin, rowmax, colmin, colmax)

    for i, note := range notes {
        Move(rowmin+i, colmin)
        fmt.Print(note)
    }

    lastrow := rowmin + len(notes) - 1

    row, col := rowmin, colmin 
    HighlightRow(notes[row-rowmin], row, rowmin, colmin, colmax)
    Move(row, col)

    for c != 'q' {
        // TODO: interrupts
        os.Stdin.Read(b)
        c = b[0]
        prevrow := row
        switch (c) {
        case 'k':
            if (row <= rowmin) {
                row = rowmin
            } else {
                row -= 1
            }
            break
        case 'j':
            if (row >= lastrow) {
                row = lastrow
            } else {
                row += 1
            }
            break
        // case 'h':
        //     if (col <= colmin) {
        //         col = colmin
        //     } else {
        //         col -= 1
        //     }
        //     break
        // case 'l':
        //     if (col >= colmax) {
        //         col = colmax 
        //     } else {
        //         col += 1
        //     }
        //     break
        }
        ResetRow(notes[prevrow-rowmin], prevrow, rowmin, colmin, colmax)
        HighlightRow(notes[row-rowmin], row, rowmin, colmin, colmax)
        Move(row, col)
    }

    Move(0,0)
    ClearScreen()
}
