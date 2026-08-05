package cmdutil

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"

	"gitee.com/oschina/gitee-cli/internal/i18n"
	"gitee.com/oschina/gitee-cli/pkg/iostreams"
)

// AIDraft holds the title and body produced by an AI generation step.
type AIDraft struct {
	Title string
	Body  string
}

// ConfirmAIDraft displays the AI-generated draft and presents the
// Use / Edit / Regenerate / Write-manually / Cancel loop to the user.
//
// regenerate is called when the user picks "Regenerate"; it should return
// a fresh AIDraft or an error. The loop repeats until the user commits to
// one option or cancels.
//
// Returns (nil, nil) when the user chose "Write manually" so callers can
// fall through to their own interactive path.
// Returns (nil, ErrUserCancelled) when the user chose "Cancel".
func ConfirmAIDraft(
	ios *iostreams.IOStreams,
	draft *AIDraft,
	editFilename string,
	regenerate func() (*AIDraft, error),
	whatToDoKey string,
	useKey, editKey, regenKey, manualKey, cancelKey string,
	useTUI bool,
) (*AIDraft, error) {
	printDraft(ios, draft)
	if !ios.IsStdinTerminal() {
		return draft, nil
	}

	for {
		choice, err := AskSelect(i18n.T(whatToDoKey), []string{
			i18n.T(useKey),
			i18n.T(editKey),
			i18n.T(regenKey),
			i18n.T(manualKey),
			i18n.T(cancelKey),
		}, useTUI)
		if err != nil {
			return nil, err
		}

		switch choice {
		case i18n.T(useKey):
			return draft, nil
		case i18n.T(editKey):
			initial := "# Title\n" + draft.Title + "\n\n# Body\n" + draft.Body
			edited, err := OpenEditor(ios, editFilename, initial)
			if err != nil {
				fmt.Fprintf(ios.ErrOut, "Warning: could not open editor: %v\n", err)
			} else {
				draft = ParseEditedDraft(edited, draft)
				printDraft(ios, draft)
			}
		case i18n.T(regenKey):
			newDraft, err := regenerate()
			if err != nil {
				fmt.Fprintf(ios.ErrOut, "Regeneration failed: %v\n", err)
				return draft, nil
			}
			draft = newDraft
			printDraft(ios, draft)
		case i18n.T(manualKey):
			return nil, nil
		case i18n.T(cancelKey):
			return nil, huh.ErrUserAborted
		}
	}
}

// ParseEditedDraft parses a "# Title\n...\n# Body\n..." editor buffer,
// falling back to fallback fields for any section that is missing or empty.
func ParseEditedDraft(edited string, fallback *AIDraft) *AIDraft {
	title := fallback.Title
	body := fallback.Body

	lines := strings.Split(edited, "\n")
	var inTitle, inBody bool
	var bodyLines []string

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "# Title"):
			inTitle, inBody = true, false
		case strings.HasPrefix(line, "# Body"):
			inTitle, inBody = false, true
		case inTitle && strings.TrimSpace(line) != "":
			title = strings.TrimSpace(line)
			inTitle = false
		case inBody:
			bodyLines = append(bodyLines, line)
		}
	}
	if len(bodyLines) > 0 {
		body = strings.TrimSpace(strings.Join(bodyLines, "\n"))
	}

	return &AIDraft{Title: title, Body: body}
}

func printDraft(ios *iostreams.IOStreams, d *AIDraft) {
	fmt.Fprintf(ios.Out, "\n%s\n%s\n\n%s\n\n",
		"────────────────────────────────────────",
		"Title: "+d.Title,
		d.Body,
	)
	fmt.Fprintln(ios.Out, "────────────────────────────────────────")
}
