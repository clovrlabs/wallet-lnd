package main

import (
	"context"
	"errors"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/urfave/cli"
)

var skipFundingConfirmationCommand = cli.Command{
	Name:     "skipfundingconf",
	Category: "Channels",
	Usage:    "Allow spending/receiving using an unconfirmed channel.",
	Description: `
	Allow spending/receiving using an unconfirmed channel.

	This makes a pending channel fully operational`,
	ArgsUsage: "value preimage",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "funding_txid",
			Usage: "the txid of the channel's funding transaction",
		},
		cli.IntFlag{
			Name: "output_index",
			Usage: "the output index for the funding output of the funding " +
				"transaction",
		},
		cli.Uint64Flag{
			Name:  "short_chan_id",
			Usage: "the agreed fake short channel id both parties agreed on ",
		},
	},
	Action: actionDecorator(activatePendingChannel),
}

func activatePendingChannel(ctx *cli.Context) error {
	ctxb := context.Background()
	client, cleanUp := getClient(ctx)
	defer cleanUp()

	channelPoint, err := parseChannelPoint(ctx)
	if err != nil {
		return err
	}
	if !ctx.IsSet("short_chan_id") {
		return errors.New("short_chan_id is required")
	}
	shortChannelID := ctx.Uint64("short_chan_id")

	resp, err := client.SkipFundingConfirmation(
		ctxb, &lnrpc.SkipFundingConfirmationRequest{
			ChannelId: shortChannelID, ChanPoint: channelPoint})

	if err != nil {
		return err
	}
	printRespJSON(resp)

	return nil
}
