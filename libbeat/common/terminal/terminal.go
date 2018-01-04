package terminal

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ReadInput Capture user input for a question
func ReadInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSuffix(input, "\n"), nil
}

// PromptYesNo Returns true if the user has enterred Y or YES, capitalization is ignored, we are
// matching elasticsearch behavior
func PromptYesNo(prompt string, defaultAnswer bool) bool {
	var defaultYNprompt string

	if defaultAnswer == true {
		defaultYNprompt = "[Y/n]"
	} else {
		defaultYNprompt = "[y/N]"
	}

	fmt.Printf("%s %s: ", prompt, defaultYNprompt)

	for {
		input, err := ReadInput()
		if err != nil {
			panic("could not read from input")
		}

		response := strings.TrimSpace(input)
		response = strings.ToLower(response)
		if response == "" {
			return defaultAnswer
		} else if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}

		fmt.Printf("Did not understand the answer '%s'\n", input)
	}
}
