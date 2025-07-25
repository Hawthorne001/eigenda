package main

import (
	"fmt"

	"github.com/Layr-Labs/eigenda/common"
	"github.com/Layr-Labs/eigenda/common/aws"
	"github.com/Layr-Labs/eigenda/common/geth"
	"github.com/Layr-Labs/eigenda/core/thegraph"
	corev2 "github.com/Layr-Labs/eigenda/core/v2"
	"github.com/Layr-Labs/eigenda/disperser/cmd/controller/flags"
	"github.com/Layr-Labs/eigenda/disperser/controller"
	"github.com/Layr-Labs/eigenda/indexer"
	"github.com/urfave/cli"
)

const MaxUint16 = ^uint16(0)

type Config struct {
	EncodingManagerConfig          controller.EncodingManagerConfig
	DispatcherConfig               controller.DispatcherConfig
	NumConcurrentEncodingRequests  int
	NumConcurrentDispersalRequests int
	NodeClientCacheSize            int

	DynamoDBTableName string

	EthClientConfig                     geth.EthClientConfig
	AwsClientConfig                     aws.ClientConfig
	DisperserStoreChunksSigningDisabled bool
	DisperserKMSKeyID                   string
	LoggerConfig                        common.LoggerConfig
	IndexerConfig                       indexer.Config
	ChainStateConfig                    thegraph.Config
	UseGraph                            bool

	EigenDADirectory              string
	BLSOperatorStateRetrieverAddr string
	EigenDAServiceManagerAddr     string

	MetricsPort                  int
	ControllerReadinessProbePath string
	ControllerHealthProbePath    string
}

func NewConfig(ctx *cli.Context) (Config, error) {
	loggerConfig, err := common.ReadLoggerCLIConfig(ctx, flags.FlagPrefix)
	if err != nil {
		return Config{}, err
	}
	ethClientConfig := geth.ReadEthClientConfigRPCOnly(ctx)
	numRelayAssignments := ctx.GlobalInt(flags.NumRelayAssignmentFlag.Name)
	if numRelayAssignments < 1 || numRelayAssignments > int(MaxUint16) {
		return Config{}, fmt.Errorf("invalid number of relay assignments: %d", numRelayAssignments)
	}
	availableRelays := ctx.GlobalIntSlice(flags.AvailableRelaysFlag.Name)
	if len(availableRelays) == 0 {
		return Config{}, fmt.Errorf("no available relays specified")
	}
	relays := make([]corev2.RelayKey, len(availableRelays))
	for i, relay := range availableRelays {
		if relay < 0 || relay > 65_535 {
			return Config{}, fmt.Errorf("invalid relay: %d", relay)
		}
		relays[i] = corev2.RelayKey(relay)
	}
	config := Config{
		DynamoDBTableName:                   ctx.GlobalString(flags.DynamoDBTableNameFlag.Name),
		EthClientConfig:                     ethClientConfig,
		AwsClientConfig:                     aws.ReadClientConfig(ctx, flags.FlagPrefix),
		DisperserStoreChunksSigningDisabled: ctx.GlobalBool(flags.DisperserStoreChunksSigningDisabledFlag.Name),
		DisperserKMSKeyID:                   ctx.GlobalString(flags.DisperserKMSKeyIDFlag.Name),
		LoggerConfig:                        *loggerConfig,
		EncodingManagerConfig: controller.EncodingManagerConfig{
			PullInterval:                ctx.GlobalDuration(flags.EncodingPullIntervalFlag.Name),
			EncodingRequestTimeout:      ctx.GlobalDuration(flags.EncodingRequestTimeoutFlag.Name),
			StoreTimeout:                ctx.GlobalDuration(flags.EncodingStoreTimeoutFlag.Name),
			NumEncodingRetries:          ctx.GlobalInt(flags.NumEncodingRetriesFlag.Name),
			NumRelayAssignment:          uint16(numRelayAssignments),
			AvailableRelays:             relays,
			EncoderAddress:              ctx.GlobalString(flags.EncoderAddressFlag.Name),
			MaxNumBlobsPerIteration:     int32(ctx.GlobalInt(flags.MaxNumBlobsPerIterationFlag.Name)),
			OnchainStateRefreshInterval: ctx.GlobalDuration(flags.OnchainStateRefreshIntervalFlag.Name),
		},
		DispatcherConfig: controller.DispatcherConfig{
			PullInterval:                          ctx.GlobalDuration(flags.DispatcherPullIntervalFlag.Name),
			FinalizationBlockDelay:                ctx.GlobalUint64(flags.FinalizationBlockDelayFlag.Name),
			AttestationTimeout:                    ctx.GlobalDuration(flags.AttestationTimeoutFlag.Name),
			BatchAttestationTimeout:               ctx.GlobalDuration(flags.BatchAttestationTimeoutFlag.Name),
			SignatureTickInterval:                 ctx.GlobalDuration(flags.SignatureTickIntervalFlag.Name),
			NumRequestRetries:                     ctx.GlobalInt(flags.NumRequestRetriesFlag.Name),
			MaxBatchSize:                          int32(ctx.GlobalInt(flags.MaxBatchSizeFlag.Name)),
			SignificantSigningThresholdPercentage: uint8(ctx.GlobalUint(flags.SignificantSigningThresholdPercentageFlag.Name)),
			SignificantSigningMetricsThresholds:   ctx.GlobalStringSlice(flags.SignificantSigningMetricsThresholdsFlag.Name),
		},
		NumConcurrentEncodingRequests:  ctx.GlobalInt(flags.NumConcurrentEncodingRequestsFlag.Name),
		NumConcurrentDispersalRequests: ctx.GlobalInt(flags.NumConcurrentDispersalRequestsFlag.Name),
		NodeClientCacheSize:            ctx.GlobalInt(flags.NodeClientCacheNumEntriesFlag.Name),
		IndexerConfig:                  indexer.ReadIndexerConfig(ctx),
		ChainStateConfig:               thegraph.ReadCLIConfig(ctx),
		UseGraph:                       ctx.GlobalBool(flags.UseGraphFlag.Name),

		BLSOperatorStateRetrieverAddr: ctx.GlobalString(flags.BlsOperatorStateRetrieverFlag.Name),
		EigenDAServiceManagerAddr:     ctx.GlobalString(flags.EigenDAServiceManagerFlag.Name),
		EigenDADirectory:              ctx.GlobalString(flags.EigenDADirectoryFlag.Name),
		MetricsPort:                   ctx.GlobalInt(flags.MetricsPortFlag.Name),
		ControllerReadinessProbePath:  ctx.GlobalString(flags.ControllerReadinessProbePathFlag.Name),
		ControllerHealthProbePath:     ctx.GlobalString(flags.ControllerHealthProbePathFlag.Name),
	}
	if !config.DisperserStoreChunksSigningDisabled && config.DisperserKMSKeyID == "" {
		return Config{}, fmt.Errorf("DisperserKMSKeyID is required when StoreChunks() signing is enabled")
	}

	return config, nil
}
