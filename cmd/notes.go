package main

import (
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	// "time"
	// "strings"

	term "golang.org/x/term"
	// termios "github.com/pkg/term/termios"
	// unix "golang.org/x/sys/unix"

	"mrshanahan.com/notes-term/internal/auth"
	w "mrshanahan.com/notes-term/internal/window"

	// "mrshanahan.com/notes-term/internal/notes"

	nc "github.com/mrshanahan/notes-api/pkg/client"
	// "github.com/mrshanahan/notes-api/pkg/notes"
)

var (
	client *nc.Client
)

func OpenEditor(path string) {
	cmd := exec.Command("nvim", path)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

type LocalCopyResult struct {
	Path         string
	IsCancelled  bool
	OpenReadOnly bool
}

func ensureLocalCacheFolder() (string, error) {
	var path = filepath.Join(os.Getenv("HOME"), ".notes-term")
	if err := os.MkdirAll(path, 0770); err != nil {
		return "", err
	}
	return path, nil
}

func ensureDraftsRoot() (string, error) {
	cacheDir, err := ensureLocalCacheFolder()
	if err != nil {
		return "", err
	}
	draftDir := filepath.Join(cacheDir, "drafts")
	if err = os.MkdirAll(draftDir, 0700); err != nil {
		return "", err
	}
	return draftDir, nil
}

func createLocalNoteCopy(window *w.MainWindow, remoteContent []byte) (*LocalCopyResult, error) {
	draftsDir, err := ensureDraftsRoot()
	if err != nil {
		return nil, err
	}

	// TODO: This is a bit long, maybe truncate
	hash := getContentHash(remoteContent)
	path := getDraftPath(draftsDir, hash)

	f, err := getIfExists(path)
	if err != nil {
		return nil, err
	}

	if f != nil {
		// TODO: Extract this out & make it available to main event loop
		finfo, _ := f.Stat()
		modtime := finfo.ModTime()
		msg := fmt.Sprintf("An unsaved draft for this note was found locally. Continue editing? (Last edited: %s)", modtime)
		selection := window.RequestOptionSelection(msg, []string{"Edit", "View (read-only)", "Discard", "Cancel"})
		switch selection {
		case 0: // Edit
			return &LocalCopyResult{Path: path}, nil
		case 1: // View (read-only)
			return &LocalCopyResult{Path: path, OpenReadOnly: true}, nil
		case 2: // Discard
			err = os.Remove(path)
			if err != nil {
				return nil, err
			}
		case 3, -1: // Cancel
			return &LocalCopyResult{IsCancelled: true}, nil
		default:
			return nil, fmt.Errorf("unexpected option choice for dealing with local copies: %d", selection)
		}
	}

	// NB: We should be here if 1) the file did not exist or 2) we discarded it.

	// If file already exists we've done something wrong, so just
	// we include O_EXCL here to fail fast
	f, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0660)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	r := bytes.NewReader(remoteContent)
	_, err = io.Copy(f, r)
	if err != nil {
		_ = os.Remove(path)
		return nil, err
	}

	return &LocalCopyResult{Path: path}, nil
}

func getContentHash(content []byte) string {
	h := sha256.New()
	h.Write(content)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func getDraftPath(root string, hash string) string {
	name := fmt.Sprintf("note-%s.txt", hash)
	path := filepath.Join(root, name)
	return path
}

func getIfExists(path string) (*os.File, error) {
	f, err := os.Open(path)
	if err != nil && os.IsNotExist(err) {
		return nil, nil
	}
	return f, err
}

func exitWithFatalError(err error) {
	w.Move(0, 0)
	w.ClearScreen()
	fmt.Fprintf(os.Stderr, "error: %s\n", err) // TODO: fix weird % at end of line
	os.Exit(-1)
}

func initState(url string) (*w.MainWindow, func()) {
	auth.InitializeAuth()
	token, err := auth.Login()
	if err != nil {
		exitWithFatalError(err) // TODO: better error message
	}

	client = nc.NewClient(url, token)

	fd := os.Stdin.Fd()
	w.DisableEcho(fd)
	// t.Lflag = t.Lflag | unix.ECHO
	// defer termios.Tcsetattr(fd, termios.TCSANOW, t)

	oldState, err := term.MakeRaw(int(fd))
	if err != nil {
		// term.Restore(int(fd), oldState)
		w.ShowCursor()
		exitWithFatalError(err) // TODO: better error message
	}
	// defer term.Restore(int(fd), oldState)

	w.ClearScreen()
	termw, termh, err := term.GetSize(int(fd))
	if err != nil {
		term.Restore(int(fd), oldState)
		w.ShowCursor()
		exitWithFatalError(err) // TODO: better error message
	}

	w.HideCursor()
	// defer ShowCursor()

	notes, err := client.ListNotes()
	if err != nil {
		term.Restore(int(fd), oldState)
		w.ShowCursor()
		exitWithFatalError(err) // TODO: better error message
	}

	w.SetPalette(w.DefaultPalette)
	window := w.NewMainWindow(termw, termh, notes)
	window.Draw()

	return window, func() {
		term.Restore(int(fd), oldState)
		w.ShowCursor()
	}
}

func main() {
	var debugFlag *bool = flag.Bool("debug", false, "Enable debugging features")
	var urlParam *string = flag.String("url", "http://localhost:3333/", "Base URL for the Notes API service")
	flag.Parse()

	w.Debug = *debugFlag

	window, cleanup := initState(*urlParam)
	defer cleanup()

	var input uint32 = 0
	exiting := false
	for !exiting {
		// TODO: interrupts
		input = w.ReadInput()
		idx := window.Selection
		switch input {
		case 'k': // up
			if idx <= 0 {
				idx = len(window.Notes) - 1
			} else {
				idx -= 1
			}
		case 'j': // down
			if idx >= len(window.Notes)-1 {
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
				// TODO: This should be a formal cache of some sort, maybe a local DB with the hash + contents
				result, err := createLocalNoteCopy(window, content)
				if err != nil {
					window.ShowErrorBox(fmt.Errorf("error when creating temp file: %w", err))
				} else if !result.IsCancelled {
					path := result.Path
					OpenEditor(path)

					newContent, err := os.ReadFile(path)
					if err == nil {
						if result.OpenReadOnly {
							window.ShowInfoBox("File was opened as read-only and so was not saved.")
						} else {
							err := client.UpdateNoteContent(note.ID, newContent)
							if err != nil {
								window.ShowErrorBox(err)
							} else {
								_ = os.Remove(path)
							}
						}
					}
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

	w.Move(0, 0)
	w.ClearScreen()
}
