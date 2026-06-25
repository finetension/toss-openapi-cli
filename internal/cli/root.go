package cli

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/auth"
	"github.com/finetension/toss-openapi-cli/internal/invest"
	"github.com/finetension/toss-openapi-cli/internal/output"
	"github.com/finetension/toss-openapi-cli/internal/version"
)

type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

type Dependencies struct {
	SecretStore auth.SecretStore
	EnvLookup   auth.EnvLookup
	TokenIssuer auth.TokenIssuer
}

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
	cmd := &cobra.Command{
		Use:           "tosscli",
		Short:         "Unofficial CLI built on public Toss Open APIs.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

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

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("version does not accept arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := output.WriteJSON(cmd.OutOrStdout(), version.Get()); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
}

func newInvestCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invest",
		Short: "Toss Invest Open API commands.",
	}
	cmd.AddCommand(newInvestAuthCommand(deps))
	return cmd
}

func newInvestAuthCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Toss Invest authentication.",
	}
	cmd.AddCommand(newInvestAuthLoginCommand(deps))
	cmd.AddCommand(newInvestAuthLogoutCommand(deps))
	cmd.AddCommand(newInvestAuthStatusCommand(deps))
	return cmd
}

func newInvestAuthLoginCommand(deps Dependencies) *cobra.Command {
	var clientID string
	var clientSecret string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Configure Toss Invest credentials.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("auth login does not accept arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			credentials, err := readLoginCredentials(cmd, clientID, clientSecret)
			if err != nil {
				return err
			}
			service := auth.NewService(deps.SecretStore, deps.EnvLookup)
			issuer := deps.TokenIssuer
			if issuer == nil {
				issuer = invest.NewClient("", nil)
			}
			status, err := service.Login(context.Background(), issuer, credentials)
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), status); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&clientID, "client-id", "", "Toss Invest client ID.")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "Toss Invest client secret.")
	return cmd
}

func newInvestAuthLogoutCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear stored Toss Invest credentials and token.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("auth logout does not accept arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			service := auth.NewService(deps.SecretStore, deps.EnvLookup)
			status, err := service.Logout()
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), status); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
}

func newInvestAuthStatusCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show Toss Invest authentication status.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("auth status does not accept arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			service := auth.NewService(deps.SecretStore, deps.EnvLookup)
			if err := output.WriteJSON(cmd.OutOrStdout(), service.Status()); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
}

func readLoginCredentials(cmd *cobra.Command, clientID string, clientSecret string) (auth.Credentials, error) {
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	reader := bufio.NewReader(cmd.InOrStdin())

	if clientID == "" {
		_, _ = fmt.Fprint(cmd.ErrOrStderr(), "Client ID: ")
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return auth.Credentials{}, apperr.Wrap(apperr.CodeUsage, "Failed to read client ID", apperr.ExitUsage, err)
		}
		clientID = strings.TrimSpace(line)
	}
	if clientSecret == "" {
		_, _ = fmt.Fprint(cmd.ErrOrStderr(), "Client Secret: ")
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return auth.Credentials{}, apperr.Wrap(apperr.CodeUsage, "Failed to read client secret", apperr.ExitUsage, err)
		}
		clientSecret = strings.TrimSpace(line)
	}
	if clientID == "" {
		return auth.Credentials{}, apperr.Usage("client ID is required")
	}
	if clientSecret == "" {
		return auth.Credentials{}, apperr.Usage("client secret is required")
	}
	return auth.Credentials{ClientID: clientID, ClientSecret: clientSecret}, nil
}

func normalizeCobraError(err error) error {
	if err == nil {
		return nil
	}
	var app *apperr.AppError
	if errors.As(err, &app) {
		return app
	}
	msg := err.Error()
	if strings.HasPrefix(msg, "unknown command") ||
		strings.HasPrefix(msg, "unknown flag") ||
		strings.HasPrefix(msg, "invalid argument") {
		return apperr.Usage(msg)
	}
	return apperr.Unexpected(err)
}
