package util

import (
	"fmt"

	"github.com/nspcc-dev/neo-go/cli/options"
	"github.com/nspcc-dev/neo-go/cli/paramcontext"
	"github.com/urfave/cli"
)

func sendTx(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) == 0 {
		return cli.NewExitError("missing input file", 1)
	} else if len(args) > 1 {
		return cli.NewExitError("only one input file is accepted", 1)
	}

	pc, err := paramcontext.Read(args[0])
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	tx, err := pc.GetCompleteTransaction()
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed to complete transaction: %w", err), 1)
	}

	gctx, cancel := options.GetTimeoutContext(ctx)
	defer cancel()

	c, err := options.GetRPCClient(gctx, ctx)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed to create RPC client: %w", err), 1)
	}
	res, err := c.SendRawTransaction(tx)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed to submit transaction to RPC node: %w", err), 1)
	}
	fmt.Fprintln(ctx.App.Writer, res.StringLE())
	return nil
}
