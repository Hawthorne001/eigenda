package retriever

import (
	"errors"
	"time"

	"github.com/Layr-Labs/eigenda/common"
	"github.com/Layr-Labs/eigenda/common/geth"
	"github.com/Layr-Labs/eigenda/encoding/kzg"
	"github.com/Layr-Labs/eigenda/retriever/flags"
	"github.com/urfave/cli"
)

type Config struct {
	EncoderConfig   kzg.KzgConfig
	EthClientConfig geth.EthClientConfig
	LoggerConfig    common.LoggerConfig
	MetricsConfig   MetricsConfig

	Timeout                       time.Duration
	NumConnections                int
	EigenDADirectory              string
	BLSOperatorStateRetrieverAddr string
	EigenDAServiceManagerAddr     string

	EigenDAVersion int
}

func NewConfig(ctx *cli.Context) (*Config, error) {
	version := ctx.GlobalInt(flags.EigenDAVersionFlag.Name)
	if version != 1 && version != 2 {
		return nil, errors.New("invalid EigenDA version")
	}
	loggerConfig, err := common.ReadLoggerCLIConfig(ctx, flags.FlagPrefix)
	if err != nil {
		return nil, err
	}

	return &Config{
		LoggerConfig:    *loggerConfig,
		EncoderConfig:   kzg.ReadCLIConfig(ctx),
		EthClientConfig: geth.ReadEthClientConfig(ctx),
		MetricsConfig: MetricsConfig{
			HTTPPort: ctx.GlobalString(flags.MetricsHTTPPortFlag.Name),
		},
		Timeout:                       ctx.Duration(flags.TimeoutFlag.Name),
		NumConnections:                ctx.Int(flags.NumConnectionsFlag.Name),
		EigenDADirectory:              ctx.GlobalString(flags.EigenDADirectoryFlag.Name),
		BLSOperatorStateRetrieverAddr: ctx.GlobalString(flags.BlsOperatorStateRetrieverFlag.Name),
		EigenDAServiceManagerAddr:     ctx.GlobalString(flags.EigenDAServiceManagerFlag.Name),
		EigenDAVersion:                version,
	}, nil
}
