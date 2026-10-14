package copyplugin

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.copy", &Plugin{})
}

var (
	out      io.Writer               = os.Stdout
	writeAll func(text string) error = clipboard.WriteAll
)

// Plugin registers the /copy chat command.
type Plugin struct{}

func (p *Plugin) Init(_ *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name:    "copy",
		Aliases: []string{"c"},
		Help:    "Copy an assistant answer to clipboard. Optional index: 0=last (default), 1=previous, …",
		Execute: func(_ context.Context, params *plugin.ChatCommandInput) (*plugin.ChatCommandResponse, error) {
			index, err := parseCopyIndex(params.Args)
			if err != nil {
				fmt.Fprintln(out, err.Error())
				return plugin.NopChatCommandResponse, nil
			}

			answers := assistantAnswers(params.AgentHandler.Messages())
			if len(answers) == 0 {
				fmt.Fprintln(out, "Nothing to copy: no conversation history")
				return plugin.NopChatCommandResponse, nil
			}
			if index >= len(answers) {
				fmt.Fprintf(out, "Nothing to copy: only %d assistant answer(s) in history\n", len(answers))
				return plugin.NopChatCommandResponse, nil
			}

			content := answers[index]
			if content == "" {
				fmt.Fprintln(out, "Nothing to copy: selected assistant message is empty")
				return plugin.NopChatCommandResponse, nil
			}

			if err := writeAll(content); err != nil {
				fmt.Fprintf(out, "Error copying to clipboard: %v\n", err)
				return plugin.NopChatCommandResponse, nil
			}

			fmt.Fprintln(out, "Answer copied to clipboard")
			return plugin.NopChatCommandResponse, nil
		},
	})
	return nil
}

// assistantAnswers returns assistant message contents newest-first (index 0 = last answer).
func assistantAnswers(msgs []openai.ChatCompletionMessage) []string {
	var out []string
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "assistant" {
			out = append(out, msgs[i].Content)
		}
	}
	return out
}

func parseCopyIndex(args string) (int, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(args)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("usage: /copy [index] — index must be a non-negative integer (0 = last answer)")
	}
	return n, nil
}
