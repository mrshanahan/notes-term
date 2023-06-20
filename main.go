package main

import (
    "fmt"
    "os"
    "os/exec"
    // "strings"
    term "golang.org/x/term"
    // termios "github.com/pkg/term/termios"
    // unix "golang.org/x/sys/unix"
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
)

func OpenEditor(path string) {
    cmd := exec.Command("nvim", path)
    cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
    err := cmd.Run()
    if err != nil {
        panic(err)
    }
}

func initState() (*MainWindow, func()) {
    notes := LoadIndex("/home/matt/.notes/index.txt")
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

    SetPalette(DefaultPalette)
    // defer ResetBackgroundColor()

    window := &MainWindow{Window{0, 0, termw, termh}, 0, notes, ""}
    window.Draw()

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
    var modal *Modal = nil
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
            modal = window.RequestInput("Create note", []string{"Title", "Test"})
            modal.Draw()
            os.Stdin.Read(make([]byte, 4))
            window.Draw()
            break
        case '\u000d': // Enter
            OpenEditor(window.Notes[idx].Path)
            break
        }
        window.LastKey = fmt.Sprintf("0x%x", input)
        window.Selection = idx
        window.Draw()
    }

    Move(0,0)
    ClearScreen()
}
