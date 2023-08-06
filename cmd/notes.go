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

    w "mrshanahan.com/notes-term/internal/window"
    "mrshanahan.com/notes-term/internal/notes"
)

func OpenEditor(path string) {
    cmd := exec.Command("nvim", path)
    cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
    err := cmd.Run()
    if err != nil {
        panic(err)
    }
}

func initState() (*w.MainWindow, func()) {
    notes, err := notes.LoadIndex()
    if err != nil {
        fmt.Printf("error: failed to open notes index: %s\n", err)
        os.Exit(1)
    }

    fd := os.Stdin.Fd()
    w.DisableEcho(fd)
    // t.Lflag = t.Lflag | unix.ECHO
    // defer termios.Tcsetattr(fd, termios.TCSANOW, t)

    oldState, err := term.MakeRaw(int(fd))
    if err != nil {
        panic(err)
    }
    // defer term.Restore(int(fd), oldState)

    w.ClearScreen()
    termw, termh, err := term.GetSize(int(fd))
    if err != nil {
        panic(err)
    }

    w.HideCursor()
    // defer ShowCursor()

    w.SetPalette(w.DefaultPalette)
    // defer ResetBackgroundColor()

    window := w.NewMainWindow(termw, termh, notes)
    window.Draw()

    return window, func() {
        term.Restore(int(fd), oldState)
        w.ShowCursor()
        // This might not be needed - everything seems to work w/o it
        // ResetBackgroundColor()
    }
}

func main() {
    var debugFlag *bool = flag.Bool("debug", false, "Enable debugging features")
    flag.Parse()

    w.Debug = *debugFlag

    window, cleanup := initState()
    defer cleanup()

    var input uint32 = 0
    exiting := false
    for !exiting {
        // TODO: interrupts
        input = w.ReadInput()
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
                newEntry := notes.NewNote(values["Title"])
                window.Notes = append(window.Notes, newEntry)
                notes.SaveIndex(window.Notes)
            }
        case '\u0012': // CTRL+R
            values := window.RequestInputWithDefaults("Rename note", map[string]string{"Title": window.Notes[idx].Title})
            if values != nil {
                window.Notes[idx].Title = values["Title"]
                notes.SaveIndex(window.Notes)
            }
        case '\u000d': // Enter
            OpenEditor(window.Notes[idx].Path)
            w.HideCursor()
        case '\u0004': // CTRL+D
            showtitle := window.Notes[idx].Title
            if len(showtitle) > 20 {
                showtitle = showtitle[:20]
            }
            confirmmsg := fmt.Sprintf("Delete note '%s'?", showtitle)
            yes := window.RequestConfirmation(confirmmsg)
            if yes {
                err := notes.DeleteNote(window.Notes[idx])
                if err != nil {
                    window.ShowErrorBox(err)
                } else {
                    window.Notes = append(window.Notes[:idx], window.Notes[idx+1:]...)
                    err = notes.SaveIndex(window.Notes)
                    if err != nil {
                        window.ShowErrorBox(err)
                    }
                }
            }
        case '\u0009': // CTRL+I
            values := window.RequestInput("Enter path to existing note", []string{"Path"})
            if values != nil {
                path := values["Path"]
                newEntry, err := notes.ImportNote(path)
                if err != nil {
                    window.ShowErrorBox(err)
                } else {
                    values = window.RequestInputWithDefaults("New name", map[string]string{"Title": newEntry.Title})
                    if values != nil {
                        newEntry.Title = values["Title"]
                        window.Notes = append(window.Notes, newEntry)
                        notes.SaveIndex(window.Notes)
                    } else {
                        notes.DeleteNote(newEntry)
                    }
                }
            }
        case '\u0008': // CTRL+H
            window.HelpCollapsed = !window.HelpCollapsed
        case 'q', '\u0003': // q/CTRL+C
            exiting = window.RequestConfirmation("Are you sure you want to leave?")
        }
        window.LastKeyWindow.Value = fmt.Sprintf(" 0x%x", input)
        window.Selection = idx
        window.Draw()
    }

    w.Move(0,0)
    w.ClearScreen()
}
