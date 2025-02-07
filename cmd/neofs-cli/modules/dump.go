package cmd

import (
	"github.com/mr-tron/base58"
	"github.com/nspcc-dev/neofs-node/pkg/services/control"
	controlSvc "github.com/nspcc-dev/neofs-node/pkg/services/control/server"
	"github.com/nspcc-dev/neofs-sdk-go/util/signature"
	"github.com/spf13/cobra"
)

const (
	dumpFilepathFlag     = "path"
	dumpIgnoreErrorsFlag = "no-errors"
)

var dumpShardCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump objects from shard",
	Long:  "Dump objects from shard to a file",
	Run:   dumpShard,
}

func dumpShard(cmd *cobra.Command, _ []string) {
	key, err := getKeyNoGenerate()
	exitOnErr(cmd, err)

	body := new(control.DumpShardRequest_Body)

	rawID, err := base58.Decode(shardID)
	exitOnErr(cmd, errf("incorrect shard ID encoding: %w", err))
	body.SetShardID(rawID)

	p, _ := cmd.Flags().GetString(dumpFilepathFlag)
	body.SetFilepath(p)

	ignore, _ := cmd.Flags().GetBool(dumpIgnoreErrorsFlag)
	body.SetIgnoreErrors(ignore)

	req := new(control.DumpShardRequest)
	req.SetBody(body)

	err = controlSvc.SignMessage(key, req)
	exitOnErr(cmd, errf("could not sign request: %w", err))

	cli, err := getControlSDKClient(key)
	exitOnErr(cmd, err)

	resp, err := control.DumpShard(cli.Raw(), req)
	exitOnErr(cmd, errf("rpc error: %w", err))

	sign := resp.GetSignature()

	err = signature.VerifyDataWithSource(
		resp,
		func() ([]byte, []byte) {
			return sign.GetKey(), sign.GetSign()
		},
	)
	exitOnErr(cmd, errf("invalid response signature: %w", err))

	cmd.Println("Shard has been dumped successfully.")
}

func initControlDumpShardCmd() {
	initCommonFlagsWithoutRPC(dumpShardCmd)

	flags := dumpShardCmd.Flags()
	flags.String(controlRPC, controlRPCDefault, controlRPCUsage)
	flags.StringVarP(&shardID, shardIDFlag, "", "", "Shard ID in base58 encoding")
	flags.String(dumpFilepathFlag, "", "File to write objects to")
	flags.Bool(dumpIgnoreErrorsFlag, false, "Skip invalid/unreadable objects")

	_ = dumpShardCmd.MarkFlagRequired(shardIDFlag)
	_ = dumpShardCmd.MarkFlagRequired(dumpFilepathFlag)
	_ = dumpShardCmd.MarkFlagRequired(controlRPC)
}
