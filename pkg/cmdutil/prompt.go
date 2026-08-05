package cmdutil

import (
	"github.com/charmbracelet/huh"
)

func askInputHuh(prompt, defaultVal string) (string, error) {
	var result string
	err := huh.NewInput().
		Title(prompt).
		Value(&result).
		Run()
	if err != nil {
		return defaultVal, err
	}
	if result == "" {
		return defaultVal, nil
	}
	return result, nil
}

func askPasswordHuh(prompt string) (string, error) {
	var result string
	err := huh.NewInput().
		Title(prompt).
		EchoMode(huh.EchoModePassword).
		Value(&result).
		Run()
	return result, err
}

func askSelectHuh(prompt string, options []string) (string, error) {
	var result string
	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, o)
	}
	err := huh.NewSelect[string]().
		Title(prompt).
		Options(opts...).
		Value(&result).
		Run()
	return result, err
}

func askConfirmHuh(prompt string) (bool, error) {
	var result bool
	err := huh.NewConfirm().
		Title(prompt).
		Value(&result).
		Run()
	return result, err
}

func AskInput(prompt, defaultVal string, useTUI bool) (string, error) {
	if useTUI {
		return askInputHuh(prompt, defaultVal)
	}
	return askInputSurvey(prompt, defaultVal)
}

func AskPassword(prompt string, useTUI bool) (string, error) {
	if useTUI {
		return askPasswordHuh(prompt)
	}
	return askPasswordSurvey(prompt)
}

func AskSelect(prompt string, options []string, useTUI bool) (string, error) {
	if useTUI {
		return askSelectHuh(prompt, options)
	}
	return askSelectSurvey(prompt, options)
}

func AskConfirm(prompt string, useTUI bool) (bool, error) {
	if useTUI {
		return askConfirmHuh(prompt)
	}
	return askConfirmSurvey(prompt)
}

// ConfirmDestructiveAction requires an explicit flag outside an interactive
// terminal and otherwise uses the configured prompt style.
func ConfirmDestructiveAction(f *Factory, prompt string) (bool, error) {
	if !f.IOStreams.IsStdinTerminal() {
		return false, FlagErrorf("--yes is required in non-interactive mode")
	}
	return AskConfirm(prompt, f.IsTUI())
}
