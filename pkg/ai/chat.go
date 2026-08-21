package ai

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	surveyterm "github.com/AlecAivazis/survey/v2/terminal"
	"golang.org/x/term"

	"gitee.com/oschina/gitee-cli/internal/i18n"
)

const chatPrompt = "👤 User : "

type surveyAskOne func(survey.Prompt, interface{}, ...survey.AskOpt) error

func readInteractiveChatLine(in, out *os.File, ask surveyAskOne) (string, bool, error) {
	var input string
	err := ask(
		&survey.Input{Message: "User :"},
		&input,
		survey.WithStdio(in, out, out),
		survey.WithShowCursor(true),
		survey.WithIcons(func(icons *survey.IconSet) {
			icons.Question.Text = "👤"
			icons.Question.Format = ""
		}),
	)
	if errors.Is(err, surveyterm.InterruptErr) || errors.Is(err, io.EOF) {
		return "", true, nil
	}
	return input, false, err
}

func Chat(ctx context.Context, client *Client, in io.Reader, out io.Writer) {
	fmt.Fprintf(out, "\n🤖 Gitee CLI AI : %s\n\n", i18n.T("ai.chat_welcome"))

	var history []Message
	scanner := bufio.NewScanner(in)
	inFile, inIsFile := in.(*os.File)
	outFile, outIsFile := out.(*os.File)
	interactive := inIsFile && outIsFile &&
		term.IsTerminal(int(inFile.Fd())) && term.IsTerminal(int(outFile.Fd()))

	for {
		var input string
		if interactive {
			line, eof, err := readInteractiveChatLine(inFile, outFile, survey.AskOne)
			if err != nil || eof {
				fmt.Fprintf(out, "\n👋 %s\n", i18n.T("ai.chat_exit"))
				return
			}
			input = line
		} else {
			fmt.Fprint(out, chatPrompt)
			if scanner.Scan() {
				input = scanner.Text()
			} else {
				fmt.Fprintf(out, "\n👋 %s\n", i18n.T("ai.chat_exit"))
				return
			}
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if strings.EqualFold(input, "exit") {
			fmt.Fprintf(out, "\n🤖 Gitee CLI AI : 👋 %s\n\n", i18n.T("ai.chat_bye"))
			return
		}

		history = append(history, Message{Role: "user", Content: input})

		fmt.Fprintf(out, "\n🤖 Gitee CLI AI : ")
		reply, err := client.CompleteStream(ctx, history, func(chunk string) {
			fmt.Fprint(out, chunk)
		})
		fmt.Fprintf(out, "\n\n")

		if err != nil {
			fmt.Fprintf(out, "❌ %s\n\n", i18n.Tf("ai.chat_error", err))
			history = history[:len(history)-1]
			continue
		}

		history = append(history, Message{Role: "assistant", Content: reply})
	}
}
