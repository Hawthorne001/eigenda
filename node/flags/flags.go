package flags

import (
	"fmt"
	"time"

	"github.com/docker/go-units"

	"github.com/Layr-Labs/eigenda/common"
	"github.com/Layr-Labs/eigenda/common/geth"
	"github.com/Layr-Labs/eigenda/encoding/kzg"
	"github.com/urfave/cli"
)

const (
	FlagPrefix   = "node"
	EnvVarPrefix = "NODE"

	// Node mode values
	ModeV1Only  = "v1-only"
	ModeV2Only  = "v2-only"
	ModeV1AndV2 = "v1-and-v2"
)

var (
	/* Required Flags */

	HostnameFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "hostname"),
		Usage:    "Hostname at which node is available",
		Required: true,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "HOSTNAME"),
	}
	DispersalPortFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "dispersal-port"),
		Usage:    "Port at which node registers to listen for dispersal calls",
		Required: true,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "DISPERSAL_PORT"),
	}
	RetrievalPortFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "retrieval-port"),
		Usage:    "Port at which node registers to listen for retrieval calls",
		Required: true,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "RETRIEVAL_PORT"),
	}
	InternalDispersalPortFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "internal-dispersal-port"),
		Usage:    "Port at which node listens for dispersal calls (used when node is behind NGINX)",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "INTERNAL_DISPERSAL_PORT"),
	}
	InternalRetrievalPortFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "internal-retrieval-port"),
		Usage:    "Port at which node listens for retrieval calls (used when node is behind NGINX)",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "INTERNAL_RETRIEVAL_PORT"),
	}
	V2DispersalPortFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "v2-dispersal-port"),
		Usage:    "Port at which node registers to listen for v2 dispersal calls",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "V2_DISPERSAL_PORT"),
	}
	V2RetrievalPortFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "v2-retrieval-port"),
		Usage:    "Port at which node registers to listen for v2 retrieval calls",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "V2_RETRIEVAL_PORT"),
	}
	InternalV2DispersalPortFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "internal-v2-dispersal-port"),
		Usage:    "Port at which node listens for v2 dispersal calls (used when node is behind NGINX)",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "INTERNAL_V2_DISPERSAL_PORT"),
	}
	InternalV2RetrievalPortFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "internal-v2-retrieval-port"),
		Usage:    "Port at which node listens for v2 retrieval calls (used when node is behind NGINX)",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "INTERNAL_V2_RETRIEVAL_PORT"),
	}
	EnableNodeApiFlag = cli.BoolFlag{
		Name:     common.PrefixFlag(FlagPrefix, "enable-node-api"),
		Usage:    "enable node-api to serve eigenlayer-cli node-api calls",
		Required: true,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "ENABLE_NODE_API"),
	}
	NodeApiPortFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "node-api-port"),
		Usage:    "Port at which node listens for eigenlayer-cli node-api calls",
		Required: false,
		Value:    "9091",
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "API_PORT"),
	}
	EnableMetricsFlag = cli.BoolFlag{
		Name:     common.PrefixFlag(FlagPrefix, "enable-metrics"),
		Usage:    "enable prometheus to serve metrics collection",
		Required: true,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "ENABLE_METRICS"),
	}
	MetricsPortFlag = cli.IntFlag{
		Name:     common.PrefixFlag(FlagPrefix, "metrics-port"),
		Usage:    "Port at which node listens for metrics calls",
		Required: false,
		Value:    9091,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "METRICS_PORT"),
	}
	OnchainMetricsIntervalFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "onchain-metrics-interval"),
		Usage:    "The interval in seconds at which the node polls the onchain state of the operator and update metrics. <=0 means no poll",
		Required: false,
		Value:    "180",
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "ONCHAIN_METRICS_INTERVAL"),
	}
	TimeoutFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "timeout"),
		Usage:    "Amount of time to wait for GPRC",
		Required: true,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "TIMEOUT"),
	}
	QuorumIDListFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "quorum-id-list"),
		Usage:    "Comma separated list of quorum IDs that the node will participate in. There should be at least one quorum ID. This list must not contain quorums node is already registered with.",
		Required: true,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "QUORUM_ID_LIST"),
	}
	DbPathFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "db-path"),
		Usage:    "Path for level db. This is only used for V1, and will eventually be removed.",
		Required: true,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "DB_PATH"),
	}
	// The files for encrypted private keys.
	BlsKeyFileFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "bls-key-file"),
		Required: false,
		Usage:    "Path to the encrypted bls private key",
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "BLS_KEY_FILE"),
	}
	EcdsaKeyFileFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "ecdsa-key-file"),
		Required: false,
		Usage:    "Path to the encrypted ecdsa private key",
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "ECDSA_KEY_FILE"),
	}
	// Passwords to decrypt the private keys.
	BlsKeyPasswordFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "bls-key-password"),
		Required: false,
		Usage:    "Password to decrypt bls private key",
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "BLS_KEY_PASSWORD"),
	}
	EcdsaKeyPasswordFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "ecdsa-key-password"),
		Required: false,
		Usage:    "Password to decrypt ecdsa private key",
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "ECDSA_KEY_PASSWORD"),
	}
	EigenDADirectoryFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "eigenda-directory"),
		Usage:    "Address of the EigenDA Address Directory",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "EIGENDA_DIRECTORY"),
	}
	BlsOperatorStateRetrieverFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "bls-operator-state-retriever"),
		Usage:    "[Deprecated: use EigenDADirectory instead] Address of the BLS operator state Retriever",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "BLS_OPERATOR_STATE_RETRIVER"),
	}
	EigenDAServiceManagerFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "eigenda-service-manager"),
		Usage:    "[Deprecated: use EigenDADirectory instead] Address of the EigenDA Service Manager",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "EIGENDA_SERVICE_MANAGER"),
	}
	ChurnerUrlFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "churner-url"),
		Usage:    "URL of the Churner",
		Required: true,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "CHURNER_URL"),
	}
	ChurnerUseSecureGRPC = cli.BoolTFlag{
		Name:     common.PrefixFlag(FlagPrefix, "churner-use-secure-grpc"),
		Usage:    "Whether to use secure GRPC connection to Churner",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "CHURNER_USE_SECURE_GRPC"),
	}
	RelayUseSecureGRPC = cli.BoolTFlag{
		Name:     common.PrefixFlag(FlagPrefix, "relay-use-secure-grpc"),
		Usage:    "Whether to use secure GRPC connection to Relay (defaults to true)",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "RELAY_USE_SECURE_GRPC"),
	}
	PubIPProviderFlag = cli.StringSliceFlag{
		Name:     common.PrefixFlag(FlagPrefix, "public-ip-provider"),
		Usage:    "The ip provider service(s) used to obtain a node's public IP. Valid options: 'seeip', 'ipify'",
		Required: true,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "PUBLIC_IP_PROVIDER"),
	}
	PubIPCheckIntervalFlag = cli.DurationFlag{
		Name:     common.PrefixFlag(FlagPrefix, "public-ip-check-interval"),
		Usage:    "Interval at which to check for changes in the node's public IP (Ex: 10s). If set to 0, the check will be disabled.",
		Required: false,
		Value:    10 * time.Second,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "PUBLIC_IP_CHECK_INTERVAL"),
	}

	/* Optional Flags */

	// This flag is used to control if the DA Node registers itself when it starts.
	// This is useful for testing and for hosted node where we don't want to have
	// mannual operation with CLI to register.
	// By default, it will not register itself at start.
	RegisterAtNodeStartFlag = cli.BoolFlag{
		Name:     common.PrefixFlag(FlagPrefix, "register-at-node-start"),
		Usage:    "Whether to register the node for EigenDA when it starts",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "REGISTER_AT_NODE_START"),
	}
	ExpirationPollIntervalSecFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "expiration-poll-interval"),
		Usage:    "How often (in second) to poll status and expire outdated blobs",
		Required: false,
		Value:    "180",
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "EXPIRATION_POLL_INTERVAL"),
	}
	ReachabilityPollIntervalSecFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "reachability-poll-interval"),
		Usage:    "How often (in second) to check if node is reachabile from Disperser",
		Required: false,
		Value:    "60",
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "REACHABILITY_POLL_INTERVAL"),
	}
	// Optional DataAPI URL. If not set, reachability checks are disabled
	DataApiUrlFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "dataapi-url"),
		Usage:    "URL of the DataAPI",
		Required: false,
		Value:    "",
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "DATAAPI_URL"),
	}
	// NumBatchValidators is the maximum number of parallel workers used to
	// validate a batch (defaults to 128).
	NumBatchValidatorsFlag = cli.IntFlag{
		Name:     "num-batch-validators",
		Usage:    "maximum number of parallel workers used to validate a batch (defaults to 128)",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "NUM_BATCH_VALIDATORS"),
		Value:    128,
	}
	NumBatchDeserializationWorkersFlag = cli.IntFlag{
		Name:     "num-batch-deserialization-workers",
		Usage:    "maximum number of parallel workers used to deserialize a batch (defaults to 128)",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "NUM_BATCH_DESERIALIZATION_WORKERS"),
		Value:    128,
	}
	EnableGnarkBundleEncodingFlag = cli.BoolFlag{
		Name:     "enable-gnark-bundle-encoding",
		Usage:    "Enable Gnark bundle encoding for chunks",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "ENABLE_GNARK_BUNDLE_ENCODING"),
	}
	OnchainStateRefreshIntervalFlag = cli.DurationFlag{
		Name:     common.PrefixFlag(FlagPrefix, "onchain-state-refresh-interval"),
		Usage:    "The interval at which to refresh the onchain state. This flag is only relevant in v2 (default: 1h)",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "ONCHAIN_STATE_REFRESH_INTERVAL"),
		Value:    1 * time.Hour,
	}
	ChunkDownloadTimeoutFlag = cli.DurationFlag{
		Name:     common.PrefixFlag(FlagPrefix, "chunk-download-timeout"),
		Usage:    "The timeout for downloading chunks from the relay (default: 30s)",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "CHUNK_DOWNLOAD_TIMEOUT"),
		Value:    20 * time.Second,
	}
	GRPCMsgSizeLimitV2Flag = cli.IntFlag{
		Name:     common.PrefixFlag(FlagPrefix, "grpc-msg-size-limit-v2"),
		Usage:    "The maximum message size in bytes the V2 dispersal endpoint can receive from the client. This flag is only relevant in v2 (default: 1MB)",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "GRPC_MSG_SIZE_LIMIT_V2"),
		Value:    units.MiB,
	}
	DisableDispersalAuthenticationFlag = cli.BoolFlag{
		Name:     common.PrefixFlag(FlagPrefix, "disable-dispersal-authentication"),
		Usage:    "Disable authentication for StoreChunks() calls from the disperser",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "DISABLE_DISPERSAL_AUTHENTICATION"),
	}
	DispersalAuthenticationKeyCacheSizeFlag = cli.IntFlag{
		Name:     common.PrefixFlag(FlagPrefix, "dispersal-authentication-key-cache-size"),
		Usage:    "The size of the dispersal authentication key cache",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "DISPERSAL_AUTHENTICATION_KEY_CACHE_SIZE"),
		Value:    units.KiB,
	}
	DisperserKeyTimeoutFlag = cli.DurationFlag{
		Name:     common.PrefixFlag(FlagPrefix, "disperser-key-timeout"),
		Usage:    "The duration for which a disperser key is cached",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "DISPERSER_KEY_TIMEOUT"),
		Value:    1 * time.Hour,
	}
	DispersalAuthenticationTimeoutFlag = cli.DurationFlag{
		Name:     common.PrefixFlag(FlagPrefix, "dispersal-authentication-timeout"),
		Usage:    "The duration for which a disperser authentication is valid",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "DISPERSAL_AUTHENTICATION_TIMEOUT"),
		Value:    0, // TODO (cody-littley) remove this feature
	}
	RelayMaxGRPCMessageSizeFlag = cli.IntFlag{
		Name:     common.PrefixFlag(FlagPrefix, "relay-max-grpc-message-size"),
		Usage:    "The maximum message size in bytes for messages received from the relay",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "RELAY_MAX_GRPC_MESSAGE_SIZE"),
		Value:    units.GiB, // intentionally large for the time being
	}

	ClientIPHeaderFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "client-ip-header"),
		Usage:    "The name of the header used to get the client IP address. If set to empty string, the IP address will be taken from the connection. The rightmost value of the header will be used.",
		Required: false,
		Value:    "",
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "CLIENT_IP_HEADER"),
	}

	DisableNodeInfoResourcesFlag = cli.BoolFlag{
		Name:     common.PrefixFlag(FlagPrefix, "disable-node-info-resources"),
		Usage:    "Disable system resource information (OS, architecture, CPU, memory) on the NodeInfo API",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "DISABLE_NODE_INFO_RESOURCES"),
	}

	BLSRemoteSignerEnabledFlag = cli.BoolFlag{
		Name:     common.PrefixFlag(FlagPrefix, "bls-remote-signer-enabled"),
		Usage:    "Set to true to enable the BLS remote signer",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "BLS_REMOTE_SIGNER_ENABLED"),
	}

	BLSRemoteSignerUrlFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "bls-remote-signer-url"),
		Usage:    "The URL of the BLS remote signer",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "BLS_REMOTE_SIGNER_URL"),
	}

	BLSPublicKeyHexFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "bls-public-key-hex"),
		Usage:    "The hex-encoded public key of the BLS signer",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "BLS_PUBLIC_KEY_HEX"),
	}

	BLSSignerCertFileFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "bls-signer-cert-file"),
		Usage:    "The path to the BLS signer certificate file",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "BLS_SIGNER_CERT_FILE"),
	}

	BLSSignerAPIKeyFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "bls-signer-api-key"),
		Usage:    "The API key for the BLS signer. Only required if BLSRemoteSignerEnabled is true",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "BLS_SIGNER_API_KEY"),
	}

	PprofHttpPort = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "pprof-http-port"),
		Usage:    "the http port which the pprof server is listening",
		Required: false,
		Value:    "6060",
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "PPROF_HTTP_PORT"),
	}
	EnablePprof = cli.BoolFlag{
		Name:     common.PrefixFlag(FlagPrefix, "enable-pprof"),
		Usage:    "start prrof server",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "ENABLE_PPROF"),
	}

	RuntimeModeFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "runtime-mode"),
		Usage:    fmt.Sprintf("Node runtime mode (%s (default), %s, or %s)", ModeV1AndV2, ModeV1Only, ModeV2Only),
		Required: false,
		Value:    ModeV1AndV2,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "RUNTIME_MODE"),
	}
	StoreChunksRequestMaxPastAgeFlag = cli.DurationFlag{
		Name:     common.PrefixFlag(FlagPrefix, "store-chunks-request-max-past-age"),
		Usage:    "The maximum age of a StoreChunks request in the past that the node will accept.",
		Required: false,
		Value:    5 * time.Minute,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "STORE_CHUNKS_REQUEST_MAX_PAST_AGE"),
	}
	StoreChunksRequestMaxFutureAgeFlag = cli.DurationFlag{
		Name:     common.PrefixFlag(FlagPrefix, "store-chunks-request-max-future-age"),
		Usage:    "The maximum age of a StoreChunks request in the future that the node will accept.",
		Required: false,
		Value:    5 * time.Minute,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "STORE_CHUNKS_REQUEST_MAX_FUTURE_AGE"),
	}
	LevelDBDisableSeeksCompactionV1Flag = cli.BoolTFlag{
		Name:     common.PrefixFlag(FlagPrefix, "leveldb-disable-seeks-compaction-v1"),
		Usage:    "Disable seeks compaction for LevelDB for v1",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "LEVELDB_DISABLE_SEEKS_COMPACTION_V1"),
	}
	LevelDBEnableSyncWritesV1Flag = cli.BoolFlag{
		Name:     common.PrefixFlag(FlagPrefix, "leveldb-enable-sync-writes-v1"),
		Usage:    "Enable sync writes for LevelDB for v1",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "LEVELDB_ENABLE_SYNC_WRITES_V1"),
	}
	LittDBWriteCacheSizeGBFlag = cli.IntFlag{
		Name: common.PrefixFlag(FlagPrefix, "litt-db-write-cache-size-gb"),
		Usage: "The size of the LittDB write cache in gigabytes. Overrides " +
			"LITT_DB_WRITE_CACHE_SIZE_FRACTION if > 0, otherwise is ignored.",
		Required: false,
		Value:    0,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "LITT_DB_WRITE_CACHE_SIZE_GB"),
	}
	LittDBWriteCacheSizeFractionFlag = cli.Float64Flag{
		Name:     common.PrefixFlag(FlagPrefix, "litt-db-write-cache-size-fraction"),
		Usage:    "The fraction of the total memory to use for the LittDB write cache.",
		Required: false,
		Value:    0.45,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "LITT_DB_WRITE_CACHE_SIZE_FRACTION"),
	}
	LittDBReadCacheSizeGBFlag = cli.IntFlag{
		Name: common.PrefixFlag(FlagPrefix, "litt-db-read-cache-size-gb"),
		Usage: "The size of the LittDB read cache in gigabytes. Overrides " +
			"LITT_DB_READ_CACHE_SIZE_FRACTION if > 0, otherwise is ignored.",
		Required: false,
		Value:    0,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "LITT_DB_READ_CACHE_SIZE_GB"),
	}
	LittDBReadCacheSizeFractionFlag = cli.Float64Flag{
		Name:     common.PrefixFlag(FlagPrefix, "litt-db-read-cache-size-fraction"),
		Usage:    "The fraction of the total memory to use for the LittDB read cache.",
		Required: false,
		Value:    0.05,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "LITT_DB_READ_CACHE_SIZE_FRACTION"),
	}
	LittDBStoragePathsFlag = cli.StringSliceFlag{
		Name:     common.PrefixFlag(FlagPrefix, "litt-db-storage-paths"),
		Usage:    "Comma separated list of paths to store the LittDB data files. If not provided, falls back to NODE_DB_PATH with '/chunk_v2_litt' suffix.",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "LITT_DB_STORAGE_PATHS"),
	}
	DownloadPoolSizeFlag = cli.IntFlag{
		Name:     common.PrefixFlag(FlagPrefix, "download-pool-size"),
		Usage:    "The size of the download pool. The default value is 16.",
		Required: false,
		Value:    16,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "DOWNLOAD_POOL_SIZE"),
	}
	GetChunksHotCacheReadLimitMBFlag = cli.Float64Flag{
		Name:     common.PrefixFlag(FlagPrefix, "get-chunks-hot-cache-read-limit-mb"),
		Usage:    "The rate limit for GetChunks() calls that hit the cache, unit is MB/s.",
		Required: false,
		Value:    1024,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "GET_CHUNKS_HOT_CACHE_READ_LIMIT_MB"),
	}
	GetChunksHotBurstLimitMBFlag = cli.Float64Flag{
		Name:     common.PrefixFlag(FlagPrefix, "get-chunks-hot-burst-limit-mb"),
		Usage:    "The burst limit for GetChunks() calls that hit the cache, unit is MB.",
		Required: false,
		Value:    1024,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "GET_CHUNKS_HOT_BURST_LIMIT_MB"),
	}
	GetChunksColdCacheReadLimitMBFlag = cli.Float64Flag{
		Name:     common.PrefixFlag(FlagPrefix, "get-chunks-cold-cache-read-limit-mb"),
		Usage:    "The rate limit for GetChunks() calls that miss the cache, unit is MB/s.",
		Required: false,
		Value:    32,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "GET_CHUNKS_COLD_CACHE_READ_LIMIT_MB"),
	}
	GetChunksColdBurstLimitMBFlag = cli.Float64Flag{
		Name:     common.PrefixFlag(FlagPrefix, "get-chunks-cold-burst-limit-MB"),
		Usage:    "The burst limit for GetChunks() calls that miss the cache, unit is MB.",
		Required: false,
		Value:    32,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "GET_CHUNKS_COLD_BURST_LIMIT_MB"),
	}
	GCSafetyBufferSizeGBFlag = cli.IntFlag{
		Name:     common.PrefixFlag(FlagPrefix, "gc-safety-buffer-size-gb"),
		Usage:    "The size of the safety buffer for garbage collection in gigabytes.",
		Required: false,
		Value:    1,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "GC_SAFETY_BUFFER_SIZE_GB"),
	}

	/////////////////////////////////////////////////////////////////////////////
	// TEST FLAGS SECTION
	//
	// WARNING: These flags are for testing purposes only.
	// They must be disabled in production environments as they may:
	//   - Break protocol requirements
	//   - Expose sensitive information
	//   - Bypass security checks
	//   - Degrade performance
	/////////////////////////////////////////////////////////////////////////////

	// This flag controls whether other test flags can take effect.
	// By default, it is not test mode.
	EnableTestModeFlag = cli.BoolFlag{
		Name:     common.PrefixFlag(FlagPrefix, "enable-test-mode"),
		Usage:    "Whether to run as test mode. This flag needs to be enabled for other test flags to take effect",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "ENABLE_TEST_MODE"),
	}

	// Corresponding to the BLOCK_STALE_MEASURE defined onchain in
	// contracts/src/core/EigenDAServiceManagerStorage.sol
	// This flag is used to override the value from the chain. The target use case is testing.
	OverrideBlockStaleMeasureFlag = cli.Uint64Flag{
		Name:     common.PrefixFlag(FlagPrefix, "override-block-stale-measure"),
		Usage:    "The maximum amount of blocks in the past that the service will consider stake amounts to still be valid. This is used to override the value set on chain. 0 means no override",
		Required: false,
		Value:    0,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "OVERRIDE_BLOCK_STALE_MEASURE"),
	}
	// Corresponding to the STORE_DURATION_BLOCKS defined onchain in
	// contracts/src/core/EigenDAServiceManagerStorage.sol
	// This flag is used to override the value from the chain. The target use case is testing.
	OverrideStoreDurationBlocksFlag = cli.Uint64Flag{
		Name:     common.PrefixFlag(FlagPrefix, "override-store-duration-blocks"),
		Usage:    "Unit of measure (in blocks) for which data will be stored for after confirmation. This is used to override the value set on chain. 0 means no override",
		Required: false,
		Value:    0,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "OVERRIDE_STORE_DURATION_BLOCKS"),
	}
	// DO NOT set plain private key in flag in production.
	// When test mode is enabled, the DA Node will take private BLS key from this flag.
	TestPrivateBlsFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "test-private-bls"),
		Usage:    "Test BLS private key for node operator",
		Required: false,
		EnvVar:   common.PrefixEnvVar(EnvVarPrefix, "TEST_PRIVATE_BLS"),
	}

	/////////////////////////////////////////////////////////////////////////////
	// END TEST FLAGS SECTION
	//
	// If you need to add new test flags:
	// 1. Place them within this section above
	// 2. Document their purpose and impact
	/////////////////////////////////////////////////////////////////////////////

)

var requiredFlags = []cli.Flag{
	HostnameFlag,
	DispersalPortFlag,
	RetrievalPortFlag,
	EnableMetricsFlag,
	MetricsPortFlag,
	OnchainMetricsIntervalFlag,
	EnableNodeApiFlag,
	NodeApiPortFlag,
	TimeoutFlag,
	QuorumIDListFlag,
	DbPathFlag,
	BlsKeyFileFlag,
	BlsKeyPasswordFlag,
	PubIPProviderFlag,
	PubIPCheckIntervalFlag,
	ChurnerUrlFlag,
}

var optionalFlags = []cli.Flag{
	RegisterAtNodeStartFlag,
	ExpirationPollIntervalSecFlag,
	ReachabilityPollIntervalSecFlag,
	EnableTestModeFlag,
	OverrideBlockStaleMeasureFlag,
	OverrideStoreDurationBlocksFlag,
	TestPrivateBlsFlag,
	NumBatchValidatorsFlag,
	NumBatchDeserializationWorkersFlag,
	InternalDispersalPortFlag,
	InternalRetrievalPortFlag,
	InternalV2DispersalPortFlag,
	InternalV2RetrievalPortFlag,
	ClientIPHeaderFlag,
	ChurnerUseSecureGRPC,
	RelayUseSecureGRPC,
	EcdsaKeyFileFlag,
	EcdsaKeyPasswordFlag,
	DataApiUrlFlag,
	DisableNodeInfoResourcesFlag,
	EnableGnarkBundleEncodingFlag,
	BLSRemoteSignerEnabledFlag,
	BLSRemoteSignerUrlFlag,
	BLSPublicKeyHexFlag,
	BLSSignerCertFileFlag,
	BLSSignerAPIKeyFlag,
	V2DispersalPortFlag,
	V2RetrievalPortFlag,
	OnchainStateRefreshIntervalFlag,
	ChunkDownloadTimeoutFlag,
	GRPCMsgSizeLimitV2Flag,
	PprofHttpPort,
	EnablePprof,
	DisableDispersalAuthenticationFlag,
	DispersalAuthenticationKeyCacheSizeFlag,
	DisperserKeyTimeoutFlag,
	DispersalAuthenticationTimeoutFlag,
	RelayMaxGRPCMessageSizeFlag,
	RuntimeModeFlag,
	StoreChunksRequestMaxPastAgeFlag,
	StoreChunksRequestMaxFutureAgeFlag,
	LevelDBDisableSeeksCompactionV1Flag,
	LevelDBEnableSyncWritesV1Flag,
	DownloadPoolSizeFlag,
	LittDBWriteCacheSizeGBFlag,
	LittDBReadCacheSizeGBFlag,
	LittDBWriteCacheSizeFractionFlag,
	LittDBReadCacheSizeFractionFlag,
	LittDBStoragePathsFlag,
	GetChunksHotCacheReadLimitMBFlag,
	GetChunksHotBurstLimitMBFlag,
	GetChunksColdCacheReadLimitMBFlag,
	GetChunksColdBurstLimitMBFlag,
	GCSafetyBufferSizeGBFlag,
	EigenDADirectoryFlag,
	BlsOperatorStateRetrieverFlag,
	EigenDAServiceManagerFlag,
}

func init() {
	Flags = append(requiredFlags, optionalFlags...)
	Flags = append(Flags, kzg.CLIFlags(EnvVarPrefix)...)
	Flags = append(Flags, geth.EthClientFlags(EnvVarPrefix)...)
	Flags = append(Flags, common.LoggerCLIFlags(EnvVarPrefix, FlagPrefix)...)
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag
