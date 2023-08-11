package cli

import (
	"fmt"
	"strings"
)

type Hook struct {
	Completion string `short:"c" long:"completion" description:"Give your shell name (zsh or bash) for adding automatic completion"`
}

var hook Hook

func (c *Hook) Execute(args []string) error {
	fmt.Print(c.makeCompletion(c.Completion))
	return nil
}

func (c *Hook) NoCheckUpdate() bool {
	return true
}

func (c *Hook) IsDirectCommand() bool {
	return true
}

func (c *Hook) makeCompletion(shellName string) string {
	shellName = strings.TrimSpace(strings.ToLower(shellName))
	switch shellName {
	case "zsh":
		return `
_gsloccomp() {
	local curcontext="$curcontext" state line
	typeset -A opt_args

	_arguments \
		'*: :->args'

	args=("${words[@]:1}")
	local data
	data=$(GO_FLAGS_COMPLETION=1 $words[1] "${args[@]}")
	_arguments "*: :($data)"

}
compdef _gsloccomp gslocli
`
	default:
		return `
_gsloccomp() {
    args=("${COMP_WORDS[@]:1:$COMP_CWORD}")
    local IFS=$'\n'
    COMPREPLY=($(GO_FLAGS_COMPLETION=1 ${COMP_WORDS[0]} "${args[@]}"))
    return 0
}

complete -F _gsloccomp gslocli
`
	}
	return "" // nolint
}

func init() {
	desc := `Hook your shell for using lbaas with autocompletion.`
	_, err := parser.AddCommand(
		"hook",
		desc,
		desc,
		&hook)
	if err != nil {
		panic(err)
	}
}
