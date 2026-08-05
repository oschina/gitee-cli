package ai

import (
	"io"
	"os"
	"testing"

	"github.com/AlecAivazis/survey/v2"
	surveyterm "github.com/AlecAivazis/survey/v2/terminal"
)

func TestReadInteractiveChatLine(t *testing.T) {
	in, out := os.Stdin, os.Stdout
	input, eof, err := readInteractiveChatLine(in, out, func(prompt survey.Prompt, response interface{}, opts ...survey.AskOpt) error {
		textPrompt, ok := prompt.(*survey.Input)
		if !ok {
			t.Fatalf("expected survey input prompt, got %T", prompt)
		}
		if textPrompt.Message != "User :" {
			t.Fatalf("unexpected prompt message %q", textPrompt.Message)
		}
		options := &survey.AskOptions{}
		for _, opt := range opts {
			if err := opt(options); err != nil {
				t.Fatalf("apply survey option: %v", err)
			}
		}
		if !options.PromptConfig.ShowCursor {
			t.Fatal("expected the system cursor to remain visible for IME input")
		}
		if options.PromptConfig.Icons.Question.Text != "👤" {
			t.Fatalf("unexpected prompt icon %q", options.PromptConfig.Icons.Question.Text)
		}
		if options.Stdio.In != in || options.Stdio.Out != out {
			t.Fatal("expected the chat terminal files to be passed to survey")
		}
		*(response.(*string)) = "中文"
		return nil
	})
	if err != nil {
		t.Fatalf("read interactive chat line: %v", err)
	}
	if eof {
		t.Fatal("expected submitted input, got EOF")
	}
	if input != "中文" {
		t.Fatalf("expected Chinese input, got %q", input)
	}

	for _, wantErr := range []error{surveyterm.InterruptErr, io.EOF} {
		_, eof, err = readInteractiveChatLine(in, out, func(survey.Prompt, interface{}, ...survey.AskOpt) error {
			return wantErr
		})
		if err != nil || !eof {
			t.Fatalf("expected %v to be treated as EOF, got eof=%v err=%v", wantErr, eof, err)
		}
	}
}
