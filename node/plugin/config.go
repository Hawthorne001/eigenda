package plugin

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Layr-Labs/eigenda/common"
	"github.com/Layr-Labs/eigenda/core"
	"github.com/Layr-Labs/eigenda/node/flags"
	"github.com/urfave/cli"
)

const (
	OperationOptIn        = "opt-in"
	OperationOptOut       = "opt-out"
	OperationUpdateSocket = "update-socket"
	OperationListQuorums  = "list-quorums"
)

var (
	/* Required Flags */

	PubIPProviderFlag = cli.StringFlag{
		Name:     "public-ip-provider",
		Usage:    "The ip provider service used to obtain a operator's public IP [seeip (default), ipify), or comma separated list of providers",
		Required: false,
		EnvVar:   common.PrefixEnvVar(flags.EnvVarPrefix, "PUBLIC_IP_PROVIDER"),
	}

	// The operation to run.
	OperationFlag = cli.StringFlag{
		Name:     "operation",
		Required: true,
		Usage:    "Supported operations: opt-in, opt-out, update-socket, list-quorums",
		EnvVar:   common.PrefixEnvVar(flags.EnvVarPrefix, "OPERATION"),
	}

	// The files for encrypted private keys.
	EcdsaKeyFileFlag = cli.StringFlag{
		Name:     "ecdsa-key-file",
		Required: true,
		Usage:    "Path to the encrypted ecdsa key",
		EnvVar:   common.PrefixEnvVar(flags.EnvVarPrefix, "ECDSA_KEY_FILE"),
	}
	BlsKeyFileFlag = cli.StringFlag{
		Name:     "bls-key-file",
		Required: true,
		Usage:    "Path to the encrypted bls key",
		EnvVar:   common.PrefixEnvVar(flags.EnvVarPrefix, "BLS_KEY_FILE"),
	}

	// The passwords to decrypt the private keys.
	EcdsaKeyPasswordFlag = cli.StringFlag{
		Name:     "ecdsa-key-password",
		Required: true,
		Usage:    "Password to decrypt the ecdsa key",
		EnvVar:   common.PrefixEnvVar(flags.EnvVarPrefix, "ECDSA_KEY_PASSWORD"),
	}
	BlsKeyPasswordFlag = cli.StringFlag{
		Name:     "bls-key-password",
		Required: true,
		Usage:    "Password to decrypt the bls key",
		EnvVar:   common.PrefixEnvVar(flags.EnvVarPrefix, "BLS_KEY_PASSWORD"),
	}
	BLSRemoteSignerUrlFlag = cli.StringFlag{
		Name:     "bls-remote-signer-url",
		Usage:    "The URL of the BLS remote signer",
		Required: false,
		EnvVar:   common.PrefixEnvVar(flags.EnvVarPrefix, "BLS_REMOTE_SIGNER_URL"),
	}

	BLSPublicKeyHexFlag = cli.StringFlag{
		Name:     "bls-public-key-hex",
		Usage:    "The hex-encoded public key of the BLS signer",
		Required: false,
		EnvVar:   common.PrefixEnvVar(flags.EnvVarPrefix, "BLS_PUBLIC_KEY_HEX"),
	}

	BLSSignerCertFileFlag = cli.StringFlag{
		Name:     "bls-signer-cert-file",
		Usage:    "The path to the BLS signer certificate file",
		Required: false,
		EnvVar:   common.PrefixEnvVar(flags.EnvVarPrefix, "BLS_SIGNER_CERT_FILE"),
	}

	BLSSignerAPIKeyFlag = cli.StringFlag{
		Name:     "bls-signer-api-key",
		Usage:    "The API key for the BLS signer. Only required if BLSRemoteSignerEnabled is true",
		Required: false,
		EnvVar:   common.PrefixEnvVar(flags.EnvVarPrefix, "BLS_SIGNER_API_KEY"),
	}

	// The socket and the quorums to register.
	SocketFlag = cli.StringFlag{
		Name:     "socket",
		Required: true,
		Usage:    "The socket of the EigenDA Node for serving dispersal and retrieval",
		EnvVar:   common.PrefixEnvVar(flags.EnvVarPrefix, "SOCKET"),
	}
	QuorumIDListFlag = cli.StringFlag{
		Name:     "quorum-id-list",
		Usage:    "Comma separated list of quorum IDs that the node will opt-in or opt-out, depending on the OperationFlag. If OperationFlag is opt-in, all quorums should not have been registered already; if it's opt-out, all quorums should have been registered already",
		Required: true,
		EnvVar:   common.PrefixEnvVar(flags.EnvVarPrefix, "QUORUM_ID_LIST"),
	}

	// The chain and contract addresses to register with.
	ChainRpcUrlFlag = cli.StringFlag{
		Name:     "chain-rpc",
		Usage:    "Chain rpc url",
		Required: true,
		EnvVar:   common.PrefixEnvVar(flags.EnvVarPrefix, "CHAIN_RPC"),
	}
	BlsOperatorStateRetrieverFlag = cli.StringFlag{
		Name:     "bls-operator-state-retriever",
		Usage:    "[Deprecated: use EigenDADirectory instead] Address of the BLS operator state Retriever",
		Required: false,
		EnvVar:   common.PrefixEnvVar(flags.EnvVarPrefix, "BLS_OPERATOR_STATE_RETRIVER"),
	}
	EigenDAServiceManagerFlag = cli.StringFlag{
		Name:     "eigenda-service-manager",
		Usage:    "[Deprecated: use EigenDADirectory instead] Address of the EigenDA Service Manager",
		Required: false,
		EnvVar:   common.PrefixEnvVar(flags.EnvVarPrefix, "EIGENDA_SERVICE_MANAGER"),
	}
	EigenDADirectoryFlag = cli.StringFlag{
		Name:     "eigenda-directory",
		Usage:    "Address of the EigenDA directory contract, which points to all other EigenDA contract addresses. This is the only contract entrypoint needed offchain.",
		Required: false,
		EnvVar:   common.PrefixEnvVar(flags.EnvVarPrefix, "EIGENDA_DIRECTORY"),
	}
	ChurnerUrlFlag = cli.StringFlag{
		Name:     "churner-url",
		Usage:    "URL of the Churner",
		Required: true,
		EnvVar:   common.PrefixEnvVar(flags.EnvVarPrefix, "CHURNER_URL"),
	}
	NumConfirmationsFlag = cli.IntFlag{
		Name:     "num-confirmations",
		Usage:    "Number of confirmations to wait for",
		Required: false,
		Value:    3,
		EnvVar:   common.PrefixEnvVar(flags.EnvVarPrefix, "NUM_CONFIRMATIONS"),
	}
)

type Config struct {
	PubIPProvider                 string
	Operation                     string
	EcdsaKeyFile                  string
	BlsKeyFile                    string
	EcdsaKeyPassword              string
	BlsKeyPassword                string
	BLSRemoteSignerUrl            string
	BLSPublicKeyHex               string
	BLSSignerCertFile             string
	Socket                        string
	QuorumIDList                  []core.QuorumID
	ChainRpcUrl                   string
	EigenDADirectory              string
	BLSOperatorStateRetrieverAddr string
	EigenDAServiceManagerAddr     string
	ChurnerUrl                    string
	NumConfirmations              int
	BLSSignerAPIKey               string
}

func NewConfig(ctx *cli.Context) (*Config, error) {
	idsStr := strings.Split(ctx.GlobalString(QuorumIDListFlag.Name), ",")
	ids := make([]core.QuorumID, 0)
	for _, id := range idsStr {
		val, err := strconv.Atoi(id)
		if err != nil {
			return nil, err
		}
		ids = append(ids, core.QuorumID(val))
	}
	if len(ids) == 0 {
		return nil, errors.New("no quorum ids provided")
	}

	op := ctx.GlobalString(OperationFlag.Name)
	if len(op) == 0 {
		return nil, errors.New("operation type not provided")
	}
	if op != OperationOptIn && op != OperationOptOut && op != OperationUpdateSocket && op != OperationListQuorums {
		return nil, errors.New("unsupported operation type")
	}

	return &Config{
		PubIPProvider:                 ctx.GlobalString(PubIPProviderFlag.Name),
		Operation:                     op,
		EcdsaKeyPassword:              ctx.GlobalString(EcdsaKeyPasswordFlag.Name),
		BlsKeyPassword:                ctx.GlobalString(BlsKeyPasswordFlag.Name),
		EcdsaKeyFile:                  ctx.GlobalString(EcdsaKeyFileFlag.Name),
		BlsKeyFile:                    ctx.GlobalString(BlsKeyFileFlag.Name),
		BLSRemoteSignerUrl:            ctx.GlobalString(BLSRemoteSignerUrlFlag.Name),
		BLSPublicKeyHex:               ctx.GlobalString(BLSPublicKeyHexFlag.Name),
		BLSSignerCertFile:             ctx.GlobalString(BLSSignerCertFileFlag.Name),
		Socket:                        ctx.GlobalString(SocketFlag.Name),
		QuorumIDList:                  ids,
		ChainRpcUrl:                   ctx.GlobalString(ChainRpcUrlFlag.Name),
		EigenDADirectory:              ctx.GlobalString(EigenDADirectoryFlag.Name),
		BLSOperatorStateRetrieverAddr: ctx.GlobalString(BlsOperatorStateRetrieverFlag.Name),
		EigenDAServiceManagerAddr:     ctx.GlobalString(EigenDAServiceManagerFlag.Name),
		ChurnerUrl:                    ctx.GlobalString(ChurnerUrlFlag.Name),
		NumConfirmations:              ctx.GlobalInt(NumConfirmationsFlag.Name),
		BLSSignerAPIKey:               ctx.GlobalString(BLSSignerAPIKeyFlag.Name),
	}, nil
}
