package main

import (
    "flag"
    "fmt"
    "os"
    "os/exec"
    // "strings"
    term "golang.org/x/term"
    // termios "github.com/pkg/term/termios"
    // unix "golang.org/x/sys/unix"
)

var (
    DEBUG = false
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
    notes, err := LoadIndex()
    if err != nil {
        fmt.Printf("error: failed to open notes index: %s\n", err)
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
    var debugFlag *bool = flag.Bool("debug", false, "Enable debugging features")
    flag.Parse()

    DEBUG = *debugFlag

    window, cleanup := initState()
    defer cleanup()

    var input uint32 = 0
    exiting := false
    for !exiting {
        // TODO: interrupts
        input = ReadInput()
        idx := window.Selection
        switch (input) {
        case 'k': // up
            if (idx <= 0) {
                idx = len(window.Notes)-1
            } else {
                idx -= 1
            }
        case 'j': // down
            if (idx >= len(window.Notes)-1) {
                idx = 0
            } else {
                idx += 1
            }
        case '\u000e': // CTRL+N
            values := window.RequestInput("Create note", []string{"Title"})
            if values != nil {
                newEntry := NewNote(values["Title"])
                window.Notes = append(window.Notes, newEntry)
                SaveIndex(window.Notes)
            }
        case '\u0012': // CTRL+R
            values := window.RequestInputWithDefaults("Rename note", map[string]string{"Title": window.Notes[idx].Title})
            if values != nil {
                window.Notes[idx].Title = values["Title"]
                SaveIndex(window.Notes)
            }
        case '\u000d': // Enter
            OpenEditor(window.Notes[idx].Path)
            HideCursor()
        case '\u0004': // CTRL+D
            showtitle := window.Notes[idx].Title
            if len(showtitle) > 20 {
                showtitle = showtitle[:20]
            }
            confirmmsg := fmt.Sprintf("Delete note '%s'?", showtitle)
            yes := window.RequestConfirmation(confirmmsg)
            if yes {
                err := DeleteNote(window.Notes[idx])
                if err != nil {
                    window.ShowErrorBox(err)
                } else {
                    window.Notes = append(window.Notes[:idx], window.Notes[idx+1:]...)
                    err = SaveIndex(window.Notes)
                    if err != nil {
                        window.ShowErrorBox(err)
                    }
                }
            }
        case 'q', '\u0003': // CTRL+C
            exiting = window.RequestConfirmation("Are you sure you want to leave?")
        }
        window.LastKey = fmt.Sprintf("0x%x", input)
        window.Selection = idx
        window.Draw()
    }

    Move(0,0)
    ClearScreen()
}
