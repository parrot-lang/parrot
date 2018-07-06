package readline

import (
	"github.com/chzyer/readline"
)

func Readline(staff string) (string, error) {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:                 staff,
		HistoryFile:            "/tmp/parrot",
		DisableAutoSaveHistory: false,
	})

	defer rl.Close()

	if err != nil {
		return "", err
	}
	line, err := rl.Readline()
	if err != nil {
		return "", err
	}
	return line, nil
}
