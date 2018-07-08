package core

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	. "github.com/sllt/parrot/types"
)

func flattenToWordsHelper(args []ParrotType) ([]string, error) {
	strArgs := []string{}

	for i := range args {
		switch c := args[i].(type) {
		case string:
			m := strings.Split(c, " ")
			strArgs = append(strArgs, m...)
		case Symbol:
			strArgs = append(strArgs, c.Val)
		case List:
			val, e := GetSlice(c)
			if e != nil {
				return []string{}, errors.New("convert List to string error")
			}
			words, e := flattenToWordsHelper(val)
			if e != nil {
				return []string{}, e
			}
			strArgs = append(strArgs, words...)
		default:
			return []string{}, errors.New("argment must be strings")
		}
	}

	return strArgs, nil
}

func Chomp(b []byte) []byte {
	if len(b) > 0 {
		n := len(b)
		if b[n-1] == '\n' {
			return b[:n-1]
		}
	}
	return b
}

func SystemFunction(args []ParrotType) (ParrotType, error) {
	if len(args) == 0 {
		return nil, errors.New("argment error")
	}

	flat, err := flattenToWordsHelper(args)
	if err != nil {
		return nil, fmt.Errorf("flatten on '%#v' failed with error '%s'", args, err)
	}
	if len(flat) == 0 {
		return nil, errors.New("argment error 1")
	}
	joined := strings.Join(flat, " ")

	cmd := "/bin/bash"

	var out []byte
	out, err = exec.Command(cmd, "-c", joined).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error from command: '%s'. Output:'%s'", err, string(Chomp(out)))
	}

	return string(Chomp(out)), nil
}
