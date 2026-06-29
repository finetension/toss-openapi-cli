package cli

import (
	"fmt"
	"strings"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type helpRegistry struct {
	Commands []helpCommandInfo `json:"commands"`
}

type helpCommandInfo struct {
	Path        string         `json:"path"`
	Use         string         `json:"use"`
	Short       string         `json:"short,omitempty"`
	Description string         `json:"description,omitempty"`
	OperationID string         `json:"operationId,omitempty"`
	RateLimit   string         `json:"rateLimit,omitempty"`
	OASDetails  []string       `json:"oasDetails,omitempty"`
	CLIDetails  []string       `json:"cliDetails,omitempty"`
	Examples    []string       `json:"examples,omitempty"`
	Flags       []helpFlagInfo `json:"flags,omitempty"`
}

type helpFlagInfo struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand,omitempty"`
	Type      string `json:"type,omitempty"`
	Usage     string `json:"usage,omitempty"`
	Default   string `json:"default,omitempty"`
	Source    string `json:"source,omitempty"`
}

func newHelpCommand(root *cobra.Command) *cobra.Command {
	var all bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		Long:  "Help provides help for any command in the application.",
		Args: func(cmd *cobra.Command, args []string) error {
			if all && len(args) > 0 {
				return apperr.Usage("help --all does not accept a command path")
			}
			if jsonOutput && !all {
				return apperr.Usage("help --json requires --all")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				if jsonOutput {
					if err := output.WriteJSON(cmd.OutOrStdout(), buildHelpRegistry(root)); err != nil {
						return apperr.Unexpected(err)
					}
					return nil
				}
				return writeAllHelpText(root)
			}

			if len(args) == 0 {
				return root.Help()
			}

			target, remaining, err := root.Find(args)
			if err != nil {
				return apperr.Usage(err.Error())
			}
			if len(remaining) > 0 {
				return apperr.Usage(fmt.Sprintf("unknown command %q for %q", remaining[0], target.CommandPath()))
			}
			return target.Help()
		},
	}
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Show help for all visible commands.")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print all-command help metadata as JSON. Requires --all.")
	return cmd
}

func writeAllHelpText(root *cobra.Command) error {
	commands := collectHelpCommands(root)
	for i, cmd := range commands {
		if i > 0 {
			if _, err := fmt.Fprintln(root.OutOrStdout()); err != nil {
				return apperr.Unexpected(err)
			}
		}
		if _, err := fmt.Fprintf(root.OutOrStdout(), "# %s\n\n", cmd.CommandPath()); err != nil {
			return apperr.Unexpected(err)
		}
		cmd.InitDefaultHelpFlag()
		if err := cmd.Help(); err != nil {
			return apperr.Unexpected(err)
		}
	}
	return nil
}

func buildHelpRegistry(root *cobra.Command) helpRegistry {
	commands := collectHelpCommands(root)
	items := make([]helpCommandInfo, 0, len(commands))
	for _, cmd := range commands {
		info := helpCommandInfo{
			Path:        cmd.CommandPath(),
			Use:         cmd.UseLine(),
			Short:       cmd.Short,
			Description: strings.TrimSpace(cmd.Long),
			Flags:       collectHelpFlags(cmd),
		}

		if help, ok := helpForCommand(cmd); ok {
			info.OperationID = help.OperationID
			info.RateLimit = help.RateLimit
			info.OASDetails = append([]string{}, help.OASDetails...)
			info.CLIDetails = append([]string{}, help.CLIDetails...)
			info.Examples = append([]string{}, help.Examples...)
			info.Description = help.Description
		} else if strings.TrimSpace(cmd.Example) != "" {
			info.Examples = splitRenderedExamples(cmd.Example)
		}

		items = append(items, info)
	}
	return helpRegistry{Commands: items}
}

func collectHelpCommands(root *cobra.Command) []*cobra.Command {
	commands := []*cobra.Command{root}
	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, child := range cmd.Commands() {
			if !child.IsAvailableCommand() || child.Name() == "help" {
				continue
			}
			commands = append(commands, child)
			walk(child)
		}
	}
	walk(root)
	return commands
}

func helpForCommand(cmd *cobra.Command) (commandHelp, bool) {
	key := ""
	if cmd.Annotations != nil {
		key = cmd.Annotations[helpCatalogKeyAnnotation]
	}
	if key == "" {
		return commandHelp{}, false
	}
	help, ok := helpCatalog[key]
	return help, ok
}

func collectHelpFlags(cmd *cobra.Command) []helpFlagInfo {
	help, hasHelp := helpForCommand(cmd)
	var flags []helpFlagInfo
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden || flag.Name == "help" {
			return
		}
		flags = append(flags, helpFlagInfo{
			Name:      flag.Name,
			Shorthand: flag.Shorthand,
			Type:      flag.Value.Type(),
			Usage:     flag.Usage,
			Default:   flag.DefValue,
			Source:    helpFlagSource(flag.Name, help, hasHelp),
		})
	})
	return flags
}

func helpFlagSource(name string, help commandHelp, ok bool) string {
	if ok {
		if _, exists := help.OASFlags[name]; exists {
			return "oas"
		}
		if _, exists := help.CLIFlags[name]; exists {
			return "cli"
		}
	}
	return "cli"
}

func splitRenderedExamples(rendered string) []string {
	lines := strings.Split(rendered, "\n")
	examples := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			examples = append(examples, line)
		}
	}
	return examples
}
