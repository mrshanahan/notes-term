package main

import (
    "bytes"
    "flag"
    "fmt"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "time"
    // "strings"

    term "golang.org/x/term"
    // termios "github.com/pkg/term/termios"
    // unix "golang.org/x/sys/unix"

    w "mrshanahan.com/notes-term/internal/window"
    // "mrshanahan.com/notes-term/internal/notes"


    nc "github.com/mrshanahan/notes-api/pkg/client"
    // "github.com/mrshanahan/notes-api/pkg/notes"
)

var (
    client *nc.Client
)

const (
    TEMP_FILE_ROOT = "/tmp/notes/"
)

func OpenEditor(path string) {
    cmd := exec.Command("nvim", path)
    cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
    err := cmd.Run()
    if err != nil {
        panic(err)
    }
}

func createTempFile(content []byte) (string, error) {
    err := os.MkdirAll(TEMP_FILE_ROOT, 0770)
    if err != nil {
        return "", err
    }

    f, err := os.CreateTemp(TEMP_FILE_ROOT, "note*")
    if err != nil {
        return "", err
    }
    defer f.Close()
    path := f.Name()

    r := bytes.NewReader(content)
    _, err = io.Copy(f, r)
    if err != nil {
        _ = os.Remove(path)
        return "", err
    }

    return path, nil
}

func newTempFileName() string {
    x := time.Now().UnixMilli()
    return fmt.Sprintf("note%015d.txt", x)
}

func initState() (*w.MainWindow, func()) {
    client = nc.NewClient("http://localhost:3333/")

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

    notes, err := client.ListNotes()
    if err != nil {
        panic(err)
    }

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
                // TODO: Make all of these asynchronous
                newNote, err := client.CreateNote(values["Title"])
                if err != nil {
                    window.ShowErrorBox(err)
                } else {
                    window.Notes = append(window.Notes, newNote)
                }
            }
        case '\u0012': // CTRL+R
            values := window.RequestInputWithDefaults("Rename note", map[string]string{"Title": window.Notes[idx].Title})
            if values != nil {
                note := window.Notes[idx]
                // TODO: Make all of these asynchronous
                err := client.UpdateNote(note.ID, values["Title"])
                if err != nil {
                    window.ShowErrorBox(err)
                } else {
                    updatedNote, err := client.GetNote(note.ID)
                    if err != nil {
                        window.ShowErrorBox(err)
                        window.Notes[idx].Title = values["Title"]
                    } else {
                        window.Notes[idx] = updatedNote
                    }
                }
            }
        case '\u000d': // Enter
            note := window.Notes[idx]
            content, err := client.GetNoteContent(note.ID)
            if err != nil {
                window.ShowErrorBox(err)
            } else {
                path, err := createTempFile(content)
                if err != nil {
                    window.ShowErrorBox(fmt.Errorf("error when creating temp file: %w", err))
                } else {
                    OpenEditor(path)

                    newContent, err := os.ReadFile(path)
                    if err == nil {
                        if newContent != nil && len(newContent) > 0 {
                            err := client.UpdateNoteContent(note.ID, newContent)
                            if err != nil {
                                window.ShowErrorBox(err)
                            }
                        }
                    }
                    _ = os.Remove(path)
                }
            }
            w.HideCursor()
        case '\u0004': // CTRL+D
            showtitle := window.Notes[idx].Title
            if len(showtitle) > 20 {
                showtitle = showtitle[:20]
            }
            confirmmsg := fmt.Sprintf("Delete note '%s'?", showtitle)
            yes := window.RequestConfirmation(confirmmsg)
            if yes {
                err := client.DeleteNote(window.Notes[idx].ID)
                if err != nil {
                    // TODO: Title describing failed action
                    window.ShowErrorBox(err)
                } else {
                    window.Notes = append(window.Notes[:idx], window.Notes[idx+1:]...)
                }
            }
        case '\u0009': // CTRL+I
            values := window.RequestInput("Enter path to existing note", []string{"Path"})
            if values != nil {
                path := values["Path"]
                _, defaultTitle := filepath.Split(path)
                values = window.RequestInputWithDefaults("New name", map[string]string{"Title": defaultTitle})
                // TODO: Move this to a separate func to avoid Matryoshka effect
                if values != nil {
                    content, err := os.ReadFile(path)
                    if err != nil {
                        window.ShowErrorBox(err)
                    } else {
                        note, err := client.CreateNote(values["Title"])
                        if err != nil {
                            window.ShowErrorBox(err)
                        } else {
                            err := client.UpdateNoteContent(note.ID, content)
                            if err != nil {
                                window.ShowErrorBox(fmt.Errorf("error while setting content; cleaning up: %w", err))
                                err := client.DeleteNote(note.ID)
                                if err != nil {
                                    window.ShowErrorBox(fmt.Errorf("error while cleaning up; manually update content for note %d: %w", note.ID, err))
                                }
                            } else {
                                window.Notes = append(window.Notes, note)
                            }
                        }
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
