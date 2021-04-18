package cmd

import "strings"

const (
	RegisterNewUser = "login"
	NewChat         = "chat"
	CloseLocalChat  = "\\q"
	CloseRemoteChat
)

func Parse(data []byte) (command, arg string) {
	text := strings.TrimSuffix(string(data), "\n")
	text = strings.TrimSpace(text)

	parts := strings.Split(text, " ")

	if len(parts) < 2 {
		return text, ""
	}

	return parts[0], strings.Join(parts[1:], " ")
}
