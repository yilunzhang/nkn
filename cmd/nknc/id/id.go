package id

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"

	api "github.com/nknorg/nkn/v2/api/common"
	"github.com/nknorg/nkn/v2/api/httpjson/client"
	nknc "github.com/nknorg/nkn/v2/cmd/nknc/common"
	"github.com/nknorg/nkn/v2/common"
	"github.com/nknorg/nkn/v2/config"
	"github.com/nknorg/nkn/v2/vault"

	"github.com/urfave/cli"
)

func generateIDAction(c *cli.Context) error {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	walletName := c.String("wallet")
	passwd := c.String("password")
	myWallet, err := vault.OpenWallet(walletName, nknc.GetPassword(passwd))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var txnFee common.Fixed64
	fee := c.String("fee")
	if fee == "" {
		txnFee = common.Fixed64(0)
	} else {
		txnFee, _ = common.StringToFixed64(fee)
	}

	var regFee common.Fixed64
	fee = c.String("regfee")
	if fee == "" {
		regFee = common.Fixed64(0)
	} else {
		regFee, _ = common.StringToFixed64(fee)
	}

	nonce := c.Uint64("nonce")

	var resp []byte
	switch {
	case c.Bool("genid"):
		account, err := myWallet.GetDefaultAccount()
		if err != nil {
			return err
		}

		walletAddr, err := account.ProgramHash.ToAddress()
		if err != nil {
			return err
		}

		remoteNonce, height, err := client.GetNonceByAddr(nknc.Address(), walletAddr)
		if err != nil {
			return err
		}

		if nonce == 0 {
			nonce = remoteNonce
		}

		txn, _ := api.MakeGenerateIDTransaction(context.Background(), myWallet, regFee, nonce, txnFee, config.MaxGenerateIDTxnHash.GetValueAtHeight(height+1))
		buff, _ := txn.Marshal()
		resp, err = client.Call(nknc.Address(), "sendrawtransaction", 0, map[string]interface{}{"tx": hex.EncodeToString(buff)})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}
	default:
		cli.ShowSubcommandHelp(c)
		return nil
	}
	nknc.FormatOutput(resp)

	return nil
}

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:        "id",
		Usage:       "generate id for nknd",
		Description: "With nknc id, you could generate ID.",
		ArgsUsage:   "[args]",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "genid",
				Usage: "generate id",
			},
			cli.StringFlag{
				Name:  "wallet, w",
				Usage: "wallet name",
				Value: config.Parameters.WalletFile,
			},
			cli.StringFlag{
				Name:  "password, p",
				Usage: "wallet password",
			},
			cli.StringFlag{
				Name:  "regfee",
				Usage: "registration fee",
				Value: "",
			},
			cli.StringFlag{
				Name:  "fee, f",
				Usage: "transaction fee",
				Value: "",
			},
			cli.Uint64Flag{
				Name:  "nonce",
				Usage: "nonce",
			},
		},
		Action: generateIDAction,
		OnUsageError: func(c *cli.Context, err error, isSubcommand bool) error {
			nknc.PrintError(c, err, "id")
			return cli.NewExitError("", 1)
		},
	}
}
