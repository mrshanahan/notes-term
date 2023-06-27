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

    window := &MainWindow{Window{0, 0, termw, termh, true}, 0, notes, ""}
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
    // var modal *Modal = nil
    for input != 'q' {
        // TODO: interrupts
        input = ReadInput()
        idx := window.Selection
        switch (input) {
        case 'k': // up
            if (idx <= 0) {
                idx = 0
            } else {
                idx -= 1
            }
        case 'j': // down
            if (idx >= len(window.Notes)-1) {
                idx = len(window.Notes)-1
            } else {
                idx += 1
            }
        case '\u000e': // CTRL+n
            window.RequestInput("Create note", []string{"Title", "Test"})
        case '\u000d': // Enter
            OpenEditor(window.Notes[idx].Path)
        }
        window.LastKey = fmt.Sprintf("0x%x", input)
        window.Selection = idx
        window.Draw()
    }

    Move(0,0)
    ClearScreen()
}
