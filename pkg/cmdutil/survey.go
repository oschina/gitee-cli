package cmdutil

import (
	"github.com/AlecAivazis/survey/v2"
)

func askInputSurvey(prompt, defaultVal string) (string, error) {
	var result string
	err := survey.AskOne(&survey.Input{
		Message: prompt,
		Default: defaultVal,
	}, &result)
	return result, err
}

func askPasswordSurvey(prompt string) (string, error) {
	var result string
	err := survey.AskOne(&survey.Password{
		Message: prompt,
	}, &result)
	return result, err
}

func askSelectSurvey(prompt string, options []string) (string, error) {
	var result string
	err := survey.AskOne(&survey.Select{
		Message: prompt,
		Options: options,
	}, &result)
	return result, err
}

func askConfirmSurvey(prompt string) (bool, error) {
	var result bool
	err := survey.AskOne(&survey.Confirm{
		Message: prompt,
		Default: true,
	}, &result)
	return result, err
}
