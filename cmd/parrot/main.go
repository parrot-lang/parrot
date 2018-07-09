package main

import (
	"fmt"
	"os"
	"strings"
)

import (
	. "github.com/sllt/parrot"
	"github.com/sllt/parrot/readline"
	. "github.com/sllt/parrot/types"
)

func main() {
	if len(os.Args) > 1 {
		args := make([]ParrotType, 0, len(os.Args)-2)
		for _, a := range os.Args[2:] {
			args = append(args, a)
		}
		Repl_env.Set(Symbol{"*ARGV*"}, List{args, nil})
		if _, e := Rep("(load-file \"" + os.Args[1] + "\")"); e != nil {
			fmt.Printf("Error: %v\n", e)
			os.Exit(1)
		}
		os.Exit(0)
	}

	Rep("(println \"Parrot 0.06-alpha [Go 1.9.4] \")")
	for {
		text, err := readline.Readline("user> ")
		text = strings.TrimRight(text, "\n")
		if err != nil {
			return
		}
		var out ParrotType
		var e error
		if out, e = Rep(text); e != nil {
			if e.Error() == "<empty line>" {
				continue
			}
			fmt.Printf("Error: %v\n", e)
			continue
		}
		fmt.Printf("%v\n", out)
	}
}
