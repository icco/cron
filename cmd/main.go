package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/icco/cron"
)

func completer(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "run", Description: "Run a certain command set."},
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}

func executor(in string) {
	in = strings.TrimSpace(in)

	blocks := strings.Split(in, " ")
	switch blocks[0] {
	case "run":
		if len(blocks) != 2 {
			fmt.Println("Sorry, I don't understand.")
			return
		}
		err := cron.Act(context.Background(), blocks[1])
		if err != nil {
			fmt.Println(err.Error())
		}
		return
	default:
		fmt.Println("Sorry, I don't understand.")
		return
	}
}

func main() {
	p := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix("> "),
		prompt.OptionTitle("cron test"),
	)
	p.Run()
}
