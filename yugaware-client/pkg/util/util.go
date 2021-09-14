package util

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

func MaskOut(value string) string {
	buf := strings.Builder{}

	for i := 0; i < len(value); i++ {
		buf.WriteRune('*')
	}

	return buf.String()
}

func PasswordPrompt() (string, error) {
	promptError := func(err error) (string, error) {
		return "", fmt.Errorf("unable to get password: %w", err)
	}
	fmt.Print("Enter password: ")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return promptError(err)
	}
	fmt.Println()

	fmt.Print("Confirm password: ")
	confirmation, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return promptError(err)
	}
	if !bytes.Equal(password, confirmation) {
		return promptError(fmt.Errorf("passwords did not match"))
	}

	return string(password), nil
}
