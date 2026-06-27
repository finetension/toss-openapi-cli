package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
	"github.com/finetension/toss-openapi-cli/internal/auth"
	"github.com/finetension/toss-openapi-cli/internal/invest"
	"github.com/finetension/toss-openapi-cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newInvestAuthCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Toss Invest authentication.",
	}
	cmd.AddCommand(newInvestAuthLoginCommand(deps))
	cmd.AddCommand(newInvestAuthLogoutCommand(deps))
	cmd.AddCommand(newInvestAuthStatusCommand(deps))
	cmd.AddCommand(newInvestAuthTokenCommand(deps))
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
			credentials, err := readLoginCredentials(cmd, deps.EnvLookup, clientID, clientSecret)
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
	applyHelp(cmd, "cli:auth-login")
	return cmd
}

func newInvestAuthLogoutCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
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
	applyHelp(cmd, "cli:auth-logout")
	return cmd
}

func newInvestAuthTokenCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Issue or refresh a Toss Invest access token.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return apperr.Usage("auth token does not accept arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			service := auth.NewService(deps.SecretStore, deps.EnvLookup)
			issuer := deps.TokenIssuer
			if issuer == nil {
				issuer = invest.NewClient("", nil)
			}
			status, err := service.Token(context.Background(), issuer)
			if err != nil {
				return err
			}
			if err := output.WriteJSON(cmd.OutOrStdout(), status); err != nil {
				return apperr.Unexpected(err)
			}
			return nil
		},
	}
	applyHelp(cmd, "cli:auth-token")
	return cmd
}

func newInvestAuthStatusCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
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
	applyHelp(cmd, "cli:auth-status")
	return cmd
}

func readLoginCredentials(cmd *cobra.Command, env auth.EnvLookup, clientID string, clientSecret string) (auth.Credentials, error) {
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	if env == nil {
		env = auth.DefaultEnvLookup
	}
	if clientID == "" {
		if value, ok := env("TOSS_INVEST_CLIENT_ID"); ok {
			clientID = strings.TrimSpace(value)
		}
	}
	if clientSecret == "" {
		if value, ok := env("TOSS_INVEST_CLIENT_SECRET"); ok {
			clientSecret = strings.TrimSpace(value)
		}
	}

	reader := bufio.NewReader(cmd.InOrStdin())

	if clientID == "" {
		line, err := readPromptLine(cmd.ErrOrStderr(), reader, "Client ID: ")
		if err != nil && !errors.Is(err, io.EOF) {
			return auth.Credentials{}, apperr.Wrap(apperr.CodeUsage, "Failed to read client ID", apperr.ExitUsage, err)
		}
		clientID = strings.TrimSpace(line)
	}
	if clientSecret == "" {
		line, err := readSecretPrompt(cmd, reader)
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

func readPromptLine(stderr io.Writer, reader *bufio.Reader, prompt string) (string, error) {
	_, _ = fmt.Fprint(stderr, prompt)
	return reader.ReadString('\n')
}

func readSecretPrompt(cmd *cobra.Command, reader *bufio.Reader) (string, error) {
	stderr := cmd.ErrOrStderr()
	_, _ = fmt.Fprint(stderr, "Client Secret: ")
	if file, ok := cmd.InOrStdin().(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		secret, err := term.ReadPassword(int(file.Fd()))
		_, _ = fmt.Fprintln(stderr)
		return string(secret), err
	}
	return reader.ReadString('\n')
}
