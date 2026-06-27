package cli

import (
	"context"

	"github.com/finetension/toss-openapi-cli/internal/auth"
	"github.com/finetension/toss-openapi-cli/internal/invest"
	"github.com/spf13/cobra"
)

func newInvestCommand(deps Dependencies) *cobra.Command {
	cmd := newGroupCommand("invest", "Toss Invest Open API commands.")
	cmd.AddCommand(newInvestAccountCommand(deps))
	cmd.AddCommand(newInvestAssetCommand(deps))
	cmd.AddCommand(newInvestAuthCommand(deps))
	cmd.AddCommand(newInvestMarketDataCommand(deps))
	cmd.AddCommand(newInvestMarketInfoCommand(deps))
	cmd.AddCommand(newInvestOrderInfoCommand(deps))
	cmd.AddCommand(newInvestOrderHistoryCommand(deps))
	cmd.AddCommand(newInvestOrderCommand(deps))
	cmd.AddCommand(newInvestStockInfoCommand(deps))
	return cmd
}

func accessTokenForInvest(ctx context.Context, deps Dependencies) (string, error) {
	service := auth.NewService(deps.SecretStore, deps.EnvLookup)
	issuer := deps.TokenIssuer
	if issuer == nil {
		issuer = invest.NewClient("", nil)
	}
	return service.AccessToken(ctx, issuer)
}
