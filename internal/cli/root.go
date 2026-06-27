package cli

import (
	"bytes"
	"fmt"
	"os"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/output"
	"github.com/finetension/toss-openapi-cli/internal/version"
	"github.com/spf13/cobra"
)

func Execute() int {
	streams := IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
	cmd := NewRootCommand(streams, Dependencies{})
	if err := cmd.Execute(); err != nil {
		return output.WriteError(cmd.OutOrStdout(), normalizeCobraError(err))
	}
	return apperr.ExitSuccess
}

func NewRootCommand(streams IOStreams, deps Dependencies) *cobra.Command {
	var showVersion bool
	cmd := &cobra.Command{
		Use:           "tosscli",
		Short:         "Unofficial CLI built on public Toss Open APIs.",
		Long:          "Unofficial CLI built on public Toss Open APIs.\n\nSuccessful command output is JSON on stdout. Errors are structured JSON.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage(fmt.Sprintf("unknown command %q", args[0]))
			}
			if showVersion {
				if err := output.WriteJSON(cmd.OutOrStdout(), version.Get()); err != nil {
					return apperr.Unexpected(err)
				}
				return nil
			}
			return cmd.Help()
		},
	}
	cmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Print version information.")

	if streams.Out != nil {
		cmd.SetOut(streams.Out)
	}
	if streams.ErrOut != nil {
		cmd.SetErr(streams.ErrOut)
	}
	if streams.In != nil {
		cmd.SetIn(streams.In)
	}

	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(newDoctorCommand(deps))
	cmd.AddCommand(newInvestCommand(deps))
	return cmd
}

func ExecuteForTest(args ...string) (stdout string, stderr string, exitCode int) {
	return ExecuteForTestWithDeps(Dependencies{}, args...)
}

func ExecuteForTestWithDeps(deps Dependencies, args ...string) (stdout string, stderr string, exitCode int) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd := NewRootCommand(IOStreams{Out: &out, ErrOut: &errOut}, deps)
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		exitCode = output.WriteError(&out, normalizeCobraError(err))
		return out.String(), errOut.String(), exitCode
	}
	return out.String(), errOut.String(), apperr.ExitSuccess
}
