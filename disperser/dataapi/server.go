package dataapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Layr-Labs/eigenda/core"
	"github.com/Layr-Labs/eigenda/encoding"
	"github.com/Layr-Labs/eigensdk-go/logging"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/Layr-Labs/eigenda/disperser"
	"github.com/Layr-Labs/eigenda/disperser/common/semver"
	docsv1 "github.com/Layr-Labs/eigenda/disperser/dataapi/docs/v1"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/logger"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"     // swagger embed files
	ginswagger "github.com/swaggo/gin-swagger" // gin-swagger middleware
)

const (
	maxWorkerPoolLimit   = 10
	maxQueryBatchesLimit = 2

	cacheControlParam = "Cache-Control"

	// Cache control for responses.
	// The time unit is second for max age.
	maxOperatorsNonsigningPercentageAge = 10
	maxOperatorPortCheckAge             = 60
	maxNonSignerAge                     = 10
	maxDeregisteredOperatorAge          = 10
	maxEjectedOperatorAge               = 10
	maxThroughputAge                    = 10
	maxMetricAage                       = 10
	maxFeedBlobsAge                     = 10
	maxFeedBlobAge                      = 300 // this is completely static
	maxDisperserAvailabilityAge         = 3
	maxChurnerAvailabilityAge           = 3
	maxBatcherAvailabilityAge           = 3
	maxOperatorsStakeAge                = 300 // not expect the stake change to happen frequently
)

var errNotFound = errors.New("not found")

type EigenDAGRPCServiceChecker interface {
	CheckHealth(ctx context.Context, serviceName string) (*grpc_health_v1.HealthCheckResponse, error)
	CloseConnections() error
}

type EigenDAHttpServiceChecker interface {
	CheckHealth(serviceName string) (string, error)
}

type (
	BlobMetadataResponse struct {
		BlobKey                 string                    `json:"blob_key"`
		BatchHeaderHash         string                    `json:"batch_header_hash"`
		BlobIndex               uint32                    `json:"blob_index"`
		SignatoryRecordHash     string                    `json:"signatory_record_hash"`
		ReferenceBlockNumber    uint32                    `json:"reference_block_number"`
		BatchRoot               string                    `json:"batch_root"`
		BlobInclusionProof      string                    `json:"blob_inclusion_proof"`
		BlobCommitment          *encoding.BlobCommitments `json:"blob_commitment"`
		BatchId                 uint32                    `json:"batch_id"`
		ConfirmationBlockNumber uint32                    `json:"confirmation_block_number"`
		ConfirmationTxnHash     string                    `json:"confirmation_txn_hash"`
		Fee                     string                    `json:"fee"`
		SecurityParams          []*core.SecurityParam     `json:"security_params"`
		RequestAt               uint64                    `json:"requested_at"`
		BlobStatus              disperser.BlobStatus      `json:"blob_status"`
	}

	Metric struct {
		Throughput float64 `json:"throughput"`
		CostInGas  float64 `json:"cost_in_gas"`
		// deprecated: use TotalStakePerQuorum instead. Remove when the frontend is updated.
		TotalStake          *big.Int                   `json:"total_stake"`
		TotalStakePerQuorum map[core.QuorumID]*big.Int `json:"total_stake_per_quorum"`
	}

	Throughput struct {
		Throughput float64 `json:"throughput"`
		Timestamp  uint64  `json:"timestamp"`
	}

	Meta struct {
		Size      int    `json:"size"`
		NextToken string `json:"next_token,omitempty"`
	}

	BlobsResponse struct {
		Meta Meta                    `json:"meta"`
		Data []*BlobMetadataResponse `json:"data"`
	}

	OperatorNonsigningPercentageMetrics struct {
		OperatorId           string  `json:"operator_id"`
		OperatorAddress      string  `json:"operator_address"`
		QuorumId             uint8   `json:"quorum_id"`
		TotalUnsignedBatches int     `json:"total_unsigned_batches"`
		TotalBatches         int     `json:"total_batches"`
		Percentage           float64 `json:"percentage"`
		StakePercentage      float64 `json:"stake_percentage"`
	}

	OperatorsNonsigningPercentage struct {
		Meta Meta                                   `json:"meta"`
		Data []*OperatorNonsigningPercentageMetrics `json:"data"`
	}

	OperatorStake struct {
		QuorumId        string  `json:"quorum_id"`
		OperatorId      string  `json:"operator_id"`
		OperatorAddress string  `json:"operator_address"`
		StakePercentage float64 `json:"stake_percentage"`
		Rank            int     `json:"rank"`
		StakeAmount     float64 `json:"stake_amount"`
	}

	OperatorsStakeResponse struct {
		CurrentBlock         uint32                      `json:"current_block"`
		StakeRankedOperators map[string][]*OperatorStake `json:"stake_ranked_operators"`
	}

	QueriedStateOperatorMetadata struct {
		OperatorId           string `json:"operator_id"`
		BlockNumber          uint   `json:"block_number"`
		Socket               string `json:"socket"`
		IsOnline             bool   `json:"is_online"`
		OperatorProcessError string `json:"operator_process_error"`
	}

	QueriedStateOperatorsResponse struct {
		Meta Meta                            `json:"meta"`
		Data []*QueriedStateOperatorMetadata `json:"data"`
	}

	QueriedOperatorEjections struct {
		OperatorId      string  `json:"operator_id"`
		OperatorAddress string  `json:"operator_address"`
		Quorum          uint8   `json:"quorum"`
		BlockNumber     uint64  `json:"block_number"`
		BlockTimestamp  string  `json:"block_timestamp"`
		TransactionHash string  `json:"transaction_hash"`
		StakePercentage float64 `json:"stake_percentage"`
	}
	QueriedOperatorEjectionsResponse struct {
		Ejections []*QueriedOperatorEjections `json:"ejections"`
	}

	ServiceAvailability struct {
		ServiceName   string `json:"service_name"`
		ServiceStatus string `json:"service_status"`
	}

	ServiceAvailabilityResponse struct {
		Meta Meta                   `json:"meta"`
		Data []*ServiceAvailability `json:"data"`
	}

	OperatorPortCheckRequest struct {
		OperatorId string `json:"operator_id"`
	}

	OperatorLiveness struct {
		OperatorId      string `json:"operator_id"`
		DispersalSocket string `json:"dispersal_socket"`
		DispersalOnline bool   `json:"dispersal_online"`
		DispersalStatus string `json:"dispersal_status"`
		RetrievalSocket string `json:"retrieval_socket"`
		RetrievalOnline bool   `json:"retrieval_online"`
		RetrievalStatus string `json:"retrieval_status"`
	}

	OperatorPortCheckResponse struct {
		OperatorId      string `json:"operator_id"`
		DispersalSocket string `json:"dispersal_socket"`
		DispersalOnline bool   `json:"dispersal_online"`
		DispersalStatus string `json:"dispersal_status"`
		RetrievalSocket string `json:"retrieval_socket"`
		RetrievalOnline bool   `json:"retrieval_online"`
		RetrievalStatus string `json:"retrieval_status"`
	}
	SemverReportResponse struct {
		Semver map[string]*semver.SemverMetrics `json:"semver"`
	}

	ErrorResponse struct {
		Error string `json:"error"`
	}

	server struct {
		serverMode        string
		socketAddr        string
		allowOrigins      []string
		logger            logging.Logger
		blobstore         disperser.BlobStore
		promClient        PrometheusClient
		subgraphClient    SubgraphClient
		transactor        core.Reader
		chainState        core.ChainState
		indexedChainState core.IndexedChainState

		metrics                   *Metrics
		disperserHostName         string
		churnerHostName           string
		batcherHealthEndpt        string
		eigenDAGRPCServiceChecker EigenDAGRPCServiceChecker
		eigenDAHttpServiceChecker EigenDAHttpServiceChecker

		operatorHandler *OperatorHandler
		metricsHandler  *MetricsHandler
	}
)

type ServerInterface interface {
	Start() error
	Shutdown() error
}

func NewServer(
	config Config,
	blobstore disperser.BlobStore,
	promClient PrometheusClient,
	subgraphClient SubgraphClient,
	transactor core.Reader,
	chainState core.ChainState,
	indexedChainState core.IndexedChainState,
	logger logging.Logger,
	metrics *Metrics,
	grpcConn GRPCConn,
	eigenDAGRPCServiceChecker EigenDAGRPCServiceChecker,
	eigenDAHttpServiceChecker EigenDAHttpServiceChecker,

) (*server, error) {
	// Initialize the health checker service for EigenDA services
	if grpcConn == nil {
		grpcConn = &GRPCDialerSkipTLS{}
	}

	if eigenDAGRPCServiceChecker == nil {
		eigenDAGRPCServiceChecker = NewEigenDAServiceHealthCheck(grpcConn, config.DisperserHostname, config.ChurnerHostname)
	}

	if eigenDAHttpServiceChecker == nil {
		eigenDAHttpServiceChecker = &HttpServiceAvailability{}
	}

	l := logger.With("component", "DataAPIServer")

	operatorHandler, err := NewOperatorHandler(logger, metrics, transactor, chainState, indexedChainState, subgraphClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create operatorHandler: %w", err)
	}

	return &server{
		logger:                    l,
		serverMode:                config.ServerMode,
		socketAddr:                config.SocketAddr,
		allowOrigins:              config.AllowOrigins,
		blobstore:                 blobstore,
		promClient:                promClient,
		subgraphClient:            subgraphClient,
		transactor:                transactor,
		chainState:                chainState,
		indexedChainState:         indexedChainState,
		metrics:                   metrics,
		disperserHostName:         config.DisperserHostname,
		churnerHostName:           config.ChurnerHostname,
		batcherHealthEndpt:        config.BatcherHealthEndpt,
		eigenDAGRPCServiceChecker: eigenDAGRPCServiceChecker,
		eigenDAHttpServiceChecker: eigenDAHttpServiceChecker,
		operatorHandler:           operatorHandler,
		metricsHandler:            NewMetricsHandler(promClient, V1),
	}, nil
}

func (s *server) Start() error {
	if s.serverMode == gin.ReleaseMode {
		// optimize performance and disable debug features.
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	basePath := "/api/v1"
	docsv1.SwaggerInfoV1.BasePath = basePath
	docsv1.SwaggerInfoV1.Host = os.Getenv("SWAGGER_HOST")
	v1 := router.Group(basePath)
	{
		feed := v1.Group("/feed")
		{
			feed.GET("/blobs", s.FetchBlobsHandler)
			feed.GET("/blobs/:blob_key", s.FetchBlobHandler)
			feed.GET("/batches/:batch_header_hash/blobs", s.FetchBlobsFromBatchHeaderHash)
		}
		operatorsInfo := v1.Group("/operators-info")
		{
			operatorsInfo.GET("/deregistered-operators", s.FetchDeregisteredOperators)
			operatorsInfo.GET("/operator-ejections", s.FetchOperatorEjections)
			operatorsInfo.GET("/registered-operators", s.FetchRegisteredOperators)
			operatorsInfo.GET("/port-check", s.OperatorPortCheck)
			operatorsInfo.GET("/semver-scan", s.SemverScan)
			operatorsInfo.GET("/operators-stake", s.OperatorsStake)
		}
		metrics := v1.Group("/metrics")
		{
			metrics.GET("/", s.FetchMetricsHandler)
			metrics.GET("/throughput", s.FetchMetricsThroughputHandler)
			metrics.GET("/non-signers", s.FetchNonSigners)
			metrics.GET("/operator-nonsigning-percentage", s.FetchOperatorsNonsigningPercentageHandler)
			metrics.GET("/disperser-service-availability", s.FetchDisperserServiceAvailability)
			metrics.GET("/churner-service-availability", s.FetchChurnerServiceAvailability)
			metrics.GET("/batcher-service-availability", s.FetchBatcherAvailability)
		}
		swagger := v1.Group("/swagger")
		{
			swagger.GET("/*any", ginswagger.WrapHandler(swaggerfiles.Handler, ginswagger.InstanceName("V1"), ginswagger.URL("/api/v1/swagger/doc.json")))
		}
	}

	router.GET("/", func(g *gin.Context) {
		g.JSON(http.StatusAccepted, gin.H{"status": "OK"})
	})

	router.Use(logger.SetLogger(
		logger.WithSkipPath([]string{"/"}),
	))

	config := cors.DefaultConfig()
	config.AllowOrigins = s.allowOrigins
	config.AllowCredentials = true
	config.AllowMethods = []string{"GET", "POST", "HEAD", "OPTIONS"}

	if s.serverMode != gin.ReleaseMode {
		config.AllowOrigins = []string{"*"}
	}
	router.Use(cors.New(config))

	srv := &http.Server{
		Addr:              s.socketAddr,
		Handler:           router,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errChan := run(s.logger, srv)
	return <-errChan
}

func (s *server) Shutdown() error {

	if s.eigenDAGRPCServiceChecker != nil {
		err := s.eigenDAGRPCServiceChecker.CloseConnections()

		if err != nil {
			s.logger.Error("Failed to close connections", "error", err)
			return err
		}
	}

	return nil
}

// FetchBlobHandler godoc
//
//	@Summary	Fetch blob metadata by blob key
//	@Tags		Feed
//	@Produce	json
//	@Param		blob_key	path		string	true	"Blob Key"
//	@Success	200			{object}	BlobMetadataResponse
//	@Failure	400			{object}	ErrorResponse	"error: Bad request"
//	@Failure	404			{object}	ErrorResponse	"error: Not found"
//	@Failure	500			{object}	ErrorResponse	"error: Server error"
//	@Router		/feed/blobs/{blob_key} [get]
func (s *server) FetchBlobHandler(c *gin.Context) {
	handlerStart := time.Now()
	defer func() {
		s.metrics.ObserveLatency("FetchBlob", time.Since(handlerStart))
	}()

	blobKey := c.Param("blob_key")

	metadata, err := s.getBlob(c.Request.Context(), blobKey)
	if err != nil {
		s.metrics.IncrementFailedRequestNum("FetchBlob")
		errorResponse(c, err)
		return
	}

	s.metrics.IncrementSuccessfulRequestNum("FetchBlob")
	c.Writer.Header().Set(cacheControlParam, fmt.Sprintf("max-age=%d", maxFeedBlobAge))
	c.JSON(http.StatusOK, metadata)
}

// FetchBlobsFromBatchHeaderHash godoc
//
//	@Summary	Fetch blob metadata by batch header hash
//	@Tags		Feed
//	@Produce	json
//	@Param		batch_header_hash	path		string	true	"Batch Header Hash"
//	@Param		limit				query		int		false	"Limit [default: 10]"
//	@Param		next_token			query		string	false	"Next page token"
//	@Success	200					{object}	BlobsResponse
//	@Failure	400					{object}	ErrorResponse	"error: Bad request"
//	@Failure	404					{object}	ErrorResponse	"error: Not found"
//	@Failure	500					{object}	ErrorResponse	"error: Server error"
//	@Router		/feed/batches/{batch_header_hash}/blobs [get]
func (s *server) FetchBlobsFromBatchHeaderHash(c *gin.Context) {
	handlerStart := time.Now()
	defer func() {
		s.metrics.ObserveLatency("FetchBlobsFromBatchHeaderHash", time.Since(handlerStart))
	}()

	batchHeaderHash := c.Param("batch_header_hash")
	batchHeaderHashBytes, err := ConvertHexadecimalToBytes([]byte(batchHeaderHash))
	if err != nil {
		s.metrics.IncrementFailedRequestNum("FetchBlobsFromBatchHeaderHash")
		errorResponse(c, fmt.Errorf("invalid batch header hash"))
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil {
		s.metrics.IncrementFailedRequestNum("FetchBlobsFromBatchHeaderHash")
		errorResponse(c, fmt.Errorf("invalid limit parameter"))
		return
	}
	if limit <= 0 || limit > 1000 {
		s.metrics.IncrementFailedRequestNum("FetchBlobsFromBatchHeaderHash")
		errorResponse(c, fmt.Errorf("limit must be between 0 and 1000"))
		return
	}

	var exclusiveStartKey *disperser.BatchIndexExclusiveStartKey
	nextToken := c.Query("next_token")
	if nextToken != "" {
		exclusiveStartKey, err = decodeNextToken(nextToken)
		if err != nil {
			s.metrics.IncrementFailedRequestNum("FetchBlobsFromBatchHeaderHash")
			errorResponse(c, fmt.Errorf("invalid next_token"))
			return
		}
	}

	metadatas, newExclusiveStartKey, err := s.getBlobsFromBatchHeaderHash(c.Request.Context(), batchHeaderHashBytes, limit, exclusiveStartKey)
	if err != nil {
		s.metrics.IncrementFailedRequestNum("FetchBlobsFromBatchHeaderHash")
		errorResponse(c, err)
		return
	}

	var nextPageToken string
	if newExclusiveStartKey != nil {
		nextPageToken, err = encodeNextToken(newExclusiveStartKey)
		if err != nil {
			s.metrics.IncrementFailedRequestNum("FetchBlobsFromBatchHeaderHash")
			errorResponse(c, fmt.Errorf("failed to generate next page token"))
			return
		}
	}

	s.metrics.IncrementSuccessfulRequestNum("FetchBlobsFromBatchHeaderHash")
	c.Writer.Header().Set(cacheControlParam, fmt.Sprintf("max-age=%d", maxFeedBlobAge))
	c.JSON(http.StatusOK, BlobsResponse{
		Meta: Meta{
			Size:      len(metadatas),
			NextToken: nextPageToken,
		},
		Data: metadatas,
	})
}

func decodeNextToken(token string) (*disperser.BatchIndexExclusiveStartKey, error) {
	// Decode the base64 string
	decodedBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}

	// Unmarshal the JSON into a BatchIndexExclusiveStartKey
	var key disperser.BatchIndexExclusiveStartKey
	err = json.Unmarshal(decodedBytes, &key)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &key, nil
}

func encodeNextToken(key *disperser.BatchIndexExclusiveStartKey) (string, error) {
	// Marshal the key to JSON
	jsonBytes, err := json.Marshal(key)
	if err != nil {
		return "", fmt.Errorf("failed to marshal key: %w", err)
	}

	// Encode the JSON as a base64 string
	token := base64.URLEncoding.EncodeToString(jsonBytes)

	return token, nil
}

// FetchBlobsHandler godoc
//
//	@Summary	Fetch blobs metadata list
//	@Tags		Feed
//	@Produce	json
//	@Param		limit	query		int	false	"Limit [default: 10]"
//	@Success	200		{object}	BlobsResponse
//	@Failure	400		{object}	ErrorResponse	"error: Bad request"
//	@Failure	404		{object}	ErrorResponse	"error: Not found"
//	@Failure	500		{object}	ErrorResponse	"error: Server error"
//	@Router		/feed/blobs [get]
func (s *server) FetchBlobsHandler(c *gin.Context) {
	handlerStart := time.Now()
	defer func() {
		s.metrics.ObserveLatency("FetchBlobs", time.Since(handlerStart))
	}()

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil {
		s.metrics.IncrementFailedRequestNum("FetchBlobsFromBatchHeaderHash")
		errorResponse(c, fmt.Errorf("invalid limit parameter"))
		return
	}
	if limit <= 0 {
		s.metrics.IncrementFailedRequestNum("FetchBlobsFromBatchHeaderHash")
		errorResponse(c, fmt.Errorf("limit must be greater than 0"))
		return
	}

	metadatas, err := s.getBlobs(c.Request.Context(), limit)
	if err != nil {
		s.metrics.IncrementFailedRequestNum("FetchBlobs")
		errorResponse(c, err)
		return
	}

	s.metrics.IncrementSuccessfulRequestNum("FetchBlobs")
	c.Writer.Header().Set(cacheControlParam, fmt.Sprintf("max-age=%d", maxFeedBlobsAge))
	c.JSON(http.StatusOK, BlobsResponse{
		Meta: Meta{
			Size: len(metadatas),
		},
		Data: metadatas,
	})
}

// FetchMetricsHandler godoc
//
//	@Summary	Fetch metrics
//	@Tags		Metrics
//	@Produce	json
//	@Param		start	query		int	false	"Start unix timestamp [default: 1 hour ago]"
//	@Param		end		query		int	false	"End unix timestamp [default: unix time now]"
//	@Param		limit	query		int	false	"Limit [default: 10]"
//	@Success	200		{object}	Metric
//	@Failure	400		{object}	ErrorResponse	"error: Bad request"
//	@Failure	404		{object}	ErrorResponse	"error: Not found"
//	@Failure	500		{object}	ErrorResponse	"error: Server error"
//	@Router		/metrics  [get]
func (s *server) FetchMetricsHandler(c *gin.Context) {
	handlerStart := time.Now()
	defer func() {
		s.metrics.ObserveLatency("FetchMetrics", time.Since(handlerStart))
	}()

	now := time.Now()
	start, err := strconv.ParseInt(c.DefaultQuery("start", "0"), 10, 64)
	if err != nil || start == 0 {
		start = now.Add(-time.Hour * 1).Unix()
	}

	end, err := strconv.ParseInt(c.DefaultQuery("end", "0"), 10, 64)
	if err != nil || end == 0 {
		end = now.Unix()
	}

	metric, err := s.getMetric(c.Request.Context(), start, end)
	if err != nil {
		s.metrics.IncrementFailedRequestNum("FetchMetrics")
		errorResponse(c, err)
		return
	}

	s.metrics.IncrementSuccessfulRequestNum("FetchMetrics")
	c.Writer.Header().Set(cacheControlParam, fmt.Sprintf("max-age=%d", maxMetricAage))
	c.JSON(http.StatusOK, metric)
}

// FetchMetricsThroughputHandler godoc
//
//	@Summary	Fetch throughput time series
//	@Tags		Metrics
//	@Produce	json
//	@Param		start	query		int	false	"Start unix timestamp [default: 1 hour ago]"
//	@Param		end		query		int	false	"End unix timestamp [default: unix time now]"
//	@Success	200		{object}	[]Throughput
//	@Failure	400		{object}	ErrorResponse	"error: Bad request"
//	@Failure	404		{object}	ErrorResponse	"error: Not found"
//	@Failure	500		{object}	ErrorResponse	"error: Server error"
//	@Router		/metrics/throughput  [get]
func (s *server) FetchMetricsThroughputHandler(c *gin.Context) {
	handlerStart := time.Now()
	defer func() {
		s.metrics.ObserveLatency("FetchMetricsTroughput", time.Since(handlerStart))
	}()

	now := time.Now()
	start, err := strconv.ParseInt(c.DefaultQuery("start", "0"), 10, 64)
	if err != nil || start == 0 {
		start = now.Add(-time.Hour * 1).Unix()
	}

	end, err := strconv.ParseInt(c.DefaultQuery("end", "0"), 10, 64)
	if err != nil || end == 0 {
		end = now.Unix()
	}

	ths, err := s.metricsHandler.GetThroughputTimeseries(c.Request.Context(), start, end)
	if err != nil {
		s.metrics.IncrementFailedRequestNum("FetchMetricsTroughput")
		errorResponse(c, err)
		return
	}

	s.metrics.IncrementSuccessfulRequestNum("FetchMetricsTroughput")
	c.Writer.Header().Set(cacheControlParam, fmt.Sprintf("max-age=%d", maxThroughputAge))
	c.JSON(http.StatusOK, ths)
}

// FetchNonSigners godoc
//
//	@Summary	Fetch non signers
//	@Tags		Metrics
//	@Produce	json
//	@Param		interval	query		int	false	"Interval to query for non signers in seconds [default: 3600]"
//	@Success	200			{object}	[]NonSigner
//	@Failure	400			{object}	ErrorResponse	"error: Bad request"
//	@Failure	404			{object}	ErrorResponse	"error: Not found"
//	@Failure	500			{object}	ErrorResponse	"error: Server error"
//	@Router		/metrics/non-signers  [get]
func (s *server) FetchNonSigners(c *gin.Context) {
	handlerStart := time.Now()
	defer func() {
		s.metrics.ObserveLatency("FetchNonSigners", time.Since(handlerStart))
	}()

	interval, err := strconv.ParseInt(c.DefaultQuery("interval", "3600"), 10, 64)
	if err != nil || interval == 0 {
		interval = 3600
	}
	metric, err := s.getNonSigners(c.Request.Context(), interval)
	if err != nil {
		s.metrics.IncrementFailedRequestNum("FetchNonSigners")
		errorResponse(c, err)
		return
	}

	s.metrics.IncrementSuccessfulRequestNum("FetchNonSigners")
	c.Writer.Header().Set(cacheControlParam, fmt.Sprintf("max-age=%d", maxNonSignerAge))
	c.JSON(http.StatusOK, metric)
}

// FetchOperatorsNonsigningPercentageHandler godoc
//
//	@Summary	Fetch operators non signing percentage
//	@Tags		Metrics
//	@Produce	json
//	@Param		interval	query		int		false	"Interval to query for operators nonsigning percentage [default: 3600]"
//	@Param		end			query		string	false	"End time (2006-01-02T15:04:05Z) to query for operators nonsigning percentage [default: now]"
//	@Param		live_only	query		string	false	"Whether return only live nonsigners [default: true]"
//	@Success	200			{object}	OperatorsNonsigningPercentage
//	@Failure	400			{object}	ErrorResponse	"error: Bad request"
//	@Failure	404			{object}	ErrorResponse	"error: Not found"
//	@Failure	500			{object}	ErrorResponse	"error: Server error"
//	@Router		/metrics/operator-nonsigning-percentage  [get]
func (s *server) FetchOperatorsNonsigningPercentageHandler(c *gin.Context) {
	handlerStart := time.Now()
	defer func() {
		s.metrics.ObserveLatency("FetchOperatorsNonsigningPercentageHandler", time.Since(handlerStart))
	}()

	endTime := time.Now()
	if c.Query("end") != "" {

		var err error
		endTime, err = time.Parse("2006-01-02T15:04:05Z", c.Query("end"))
		if err != nil {
			errorResponse(c, err)
			return
		}
	}

	interval, err := strconv.ParseInt(c.DefaultQuery("interval", "3600"), 10, 64)
	if err != nil || interval == 0 {
		interval = 3600
	}

	liveOnly := "true"
	if c.Query("live_only") != "" {
		liveOnly = c.Query("live_only")
		if liveOnly != "true" && liveOnly != "false" {
			errorResponse(c, errors.New("the live_only param must be \"true\" or \"false\""))
			return
		}
	}

	startTime := endTime.Add(-time.Duration(interval) * time.Second)

	metric, err := s.getOperatorNonsigningRate(c.Request.Context(), startTime.Unix(), endTime.Unix(), liveOnly == "true")
	if err != nil {
		s.metrics.IncrementFailedRequestNum("FetchOperatorsNonsigningPercentageHandler")
		errorResponse(c, err)
		return
	}

	s.metrics.IncrementSuccessfulRequestNum("FetchOperatorsNonsigningPercentageHandler")
	c.Writer.Header().Set(cacheControlParam, fmt.Sprintf("max-age=%d", maxOperatorsNonsigningPercentageAge))
	c.JSON(http.StatusOK, metric)
}

// OperatorsStake godoc
//
//	@Summary	Operator stake distribution query
//	@Tags		OperatorsStake
//	@Produce	json
//	@Param		operator_id	query		string	true	"Operator ID"
//	@Success	200			{object}	OperatorsStakeResponse
//	@Failure	400			{object}	ErrorResponse	"error: Bad request"
//	@Failure	404			{object}	ErrorResponse	"error: Not found"
//	@Failure	500			{object}	ErrorResponse	"error: Server error"
//	@Router		/operators-info/operators-stake [get]
func (s *server) OperatorsStake(c *gin.Context) {
	handlerStart := time.Now()
	defer func() {
		s.metrics.ObserveLatency("OperatorsStake", time.Since(handlerStart))
	}()

	operatorId := c.DefaultQuery("operator_id", "")
	s.logger.Info("getting operators stake distribution", "operatorId", operatorId)

	operatorsStakeResponse, err := s.operatorHandler.GetOperatorsStake(c.Request.Context(), operatorId)
	if err != nil {
		s.metrics.IncrementFailedRequestNum("OperatorsStake")
		errorResponse(c, fmt.Errorf("failed to get operator stake: %w", err))
		return
	}

	s.metrics.IncrementSuccessfulRequestNum("OperatorsStake")
	c.Writer.Header().Set(cacheControlParam, fmt.Sprintf("max-age=%d", maxOperatorsStakeAge))
	c.JSON(http.StatusOK, operatorsStakeResponse)
}

// FetchDeregisteredOperators godoc
//
//	@Summary	Fetch list of operators that have been deregistered for days. Days is a query parameter with a default value of 14 and max value of 30.
//	@Tags		OperatorsInfo
//	@Produce	json
//	@Success	200	{object}	QueriedStateOperatorsResponse
//	@Failure	400	{object}	ErrorResponse	"error: Bad request"
//	@Failure	404	{object}	ErrorResponse	"error: Not found"
//	@Failure	500	{object}	ErrorResponse	"error: Server error"
//	@Router		/operators-info/deregistered-operators [get]
func (s *server) FetchDeregisteredOperators(c *gin.Context) {
	handlerStart := time.Now()
	defer func() {
		s.metrics.ObserveLatency("FetchDeregisteredOperators", time.Since(handlerStart))
	}()

	// Get query parameters
	// Default Value 14 days
	days := c.DefaultQuery("days", "14") // If not specified, defaults to 14

	// Convert days to integer
	daysInt, err := strconv.Atoi(days)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'days' parameter"})
		return
	}

	if daysInt > 30 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'days' parameter. Max value is 30"})
		return
	}

	operatorMetadatas, err := s.getDeregisteredOperatorForDays(c.Request.Context(), int32(daysInt))
	if err != nil {
		s.logger.Error("Failed to fetch deregistered operators", "error", err)
		s.metrics.IncrementFailedRequestNum("FetchDeregisteredOperators")
		errorResponse(c, err)
		return
	}

	s.metrics.IncrementSuccessfulRequestNum("FetchDeregisteredOperators")
	c.Writer.Header().Set(cacheControlParam, fmt.Sprintf("max-age=%d", maxDeregisteredOperatorAge))
	c.JSON(http.StatusOK, QueriedStateOperatorsResponse{
		Meta: Meta{
			Size: len(operatorMetadatas),
		},
		Data: operatorMetadatas,
	})
}

// FetchRegisteredOperators godoc
//
//	@Summary	Fetch list of operators that have been registered for days. Days is a query parameter with a default value of 14 and max value of 30.
//	@Tags		OperatorsInfo
//	@Produce	json
//	@Success	200	{object}	QueriedStateOperatorsResponse
//	@Failure	400	{object}	ErrorResponse	"error: Bad request"
//	@Failure	404	{object}	ErrorResponse	"error: Not found"
//	@Failure	500	{object}	ErrorResponse	"error: Server error"
//	@Router		/operators-info/registered-operators [get]
func (s *server) FetchRegisteredOperators(c *gin.Context) {
	handlerStart := time.Now()
	defer func() {
		s.metrics.ObserveLatency("FetchRegisteredOperators", time.Since(handlerStart))
	}()

	// Get query parameters
	// Default Value 14 days
	days := c.DefaultQuery("days", "14") // If not specified, defaults to 14

	// Convert days to integer
	daysInt, err := strconv.Atoi(days)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'days' parameter"})
		return
	}

	if daysInt > 30 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'days' parameter. Max value is 30"})
		return
	}

	operatorMetadatas, err := s.getRegisteredOperatorForDays(c.Request.Context(), int32(daysInt))
	if err != nil {
		s.metrics.IncrementFailedRequestNum("FetchRegisteredOperators")
		errorResponse(c, err)
		return
	}

	s.metrics.IncrementSuccessfulRequestNum("FetchRegisteredOperators")
	c.Writer.Header().Set(cacheControlParam, fmt.Sprintf("max-age=%d", maxDeregisteredOperatorAge))
	c.JSON(http.StatusOK, QueriedStateOperatorsResponse{
		Meta: Meta{
			Size: len(operatorMetadatas),
		},
		Data: operatorMetadatas,
	})
}

// FetchOperatorEjections godoc
//
//	@Summary	Fetch list of operator ejections over last N days.
//	@Tags		OperatorsInfo
//	@Produce	json
//	@Param		days		query		int		false	"Lookback in days [default: 1]"
//	@Param		operator_id	query		string	false	"Operator ID filter [default: all operators]"
//	@Param		first		query		int		false	"Return first N ejections [default: 1000]"
//	@Param		skip		query		int		false	"Skip first N ejections [default: 0]"
//	@Success	200			{object}	QueriedOperatorEjectionsResponse
//	@Failure	400			{object}	ErrorResponse	"error: Bad request"
//	@Failure	404			{object}	ErrorResponse	"error: Not found"
//	@Failure	500			{object}	ErrorResponse	"error: Server error"
//	@Router		/operators-info/operator-ejections [get]
func (s *server) FetchOperatorEjections(c *gin.Context) {
	handlerStart := time.Now()
	defer func() {
		s.metrics.ObserveLatency("FetchOperatorEjections", time.Since(handlerStart))
	}()

	operatorId := c.DefaultQuery("operator_id", "") // If not specified, defaults to all operators

	days := c.DefaultQuery("days", "1") // If not specified, defaults to 1
	parsedDays, err := strconv.ParseInt(days, 10, 32)
	if err != nil || parsedDays < math.MinInt32 || parsedDays > math.MaxInt32 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'days' parameter"})
		return
	}
	daysInt := int32(parsedDays)

	first := c.DefaultQuery("first", "1000") // If not specified, defaults to 1000
	parsedFirst, err := strconv.ParseInt(first, 10, 32)
	if err != nil || parsedFirst < 1 || parsedFirst > 10000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'first' parameter. Value must be between 1..10000"})
		return
	}
	firstInt := int32(parsedFirst)

	skip := c.DefaultQuery("skip", "0") // If not specified, defaults to 0
	parsedSkip, err := strconv.ParseInt(skip, 10, 32)
	if err != nil || parsedSkip < 0 || parsedSkip > 1000000000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'skip' parameter. Value must be between 0..1000000000"})
		return
	}
	skipInt := int32(parsedSkip)

	operatorEjections, err := s.getOperatorEjections(c.Request.Context(), int32(daysInt), operatorId, uint(firstInt), uint(skipInt))
	if err != nil {
		s.logger.Error("Failed to fetch ejected operators", "error", err)
		s.metrics.IncrementFailedRequestNum("FetchOperatorEjections")
		errorResponse(c, err)
		return
	}

	s.metrics.IncrementSuccessfulRequestNum("FetchOperatorEjections")
	c.Writer.Header().Set(cacheControlParam, fmt.Sprintf("max-age=%d", maxEjectedOperatorAge))
	c.JSON(http.StatusOK, QueriedOperatorEjectionsResponse{
		Ejections: operatorEjections,
	})
}

// OperatorPortCheck godoc
//
//	@Summary	Operator v1 node reachability port check
//	@Tags		OperatorsInfo
//	@Produce	json
//	@Param		operator_id	query		string	true	"Operator ID"
//	@Success	200			{object}	OperatorPortCheckResponse
//	@Failure	400			{object}	ErrorResponse	"error: Bad request"
//	@Failure	404			{object}	ErrorResponse	"error: Not found"
//	@Failure	500			{object}	ErrorResponse	"error: Server error"
//	@Router		/operators-info/port-check [get]
func (s *server) OperatorPortCheck(c *gin.Context) {
	handlerStart := time.Now()
	defer func() {
		s.metrics.ObserveLatency("OperatorPortCheck", time.Since(handlerStart))
	}()

	operatorId := c.DefaultQuery("operator_id", "")
	s.logger.Info("checking operator ports", "operatorId", operatorId)
	portCheckResponse, err := s.operatorHandler.ProbeV1OperatorPorts(c.Request.Context(), operatorId)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			err = errNotFound
			s.logger.Warn("operator not found", "operatorId", operatorId)
			s.metrics.IncrementNotFoundRequestNum("OperatorPortCheck")
		} else {
			s.logger.Error("operator port check failed", "error", err)
			s.metrics.IncrementFailedRequestNum("OperatorPortCheck")
		}
		errorResponse(c, err)
		return
	}

	c.Writer.Header().Set(cacheControlParam, fmt.Sprintf("max-age=%d", maxOperatorPortCheckAge))
	c.JSON(http.StatusOK, portCheckResponse)
}

// Semver scan godoc
//
//	@Summary	Active operator semver scan
//	@Tags		OperatorsInfo
//	@Produce	json
//	@Success	200	{object}	SemverReportResponse
//	@Failure	500	{object}	ErrorResponse	"error: Server error"
//	@Router		/operators-info/semver-scan [get]
func (s *server) SemverScan(c *gin.Context) {
	handlerStart := time.Now()
	defer func() {
		s.metrics.ObserveLatency("SemverScan", time.Since(handlerStart))
	}()

	report, err := s.operatorHandler.ScanOperatorsHostInfo(c.Request.Context())
	if err != nil {
		s.logger.Error("failed to scan operators host info", "error", err)
		s.metrics.IncrementFailedRequestNum("SemverScan")
		errorResponse(c, err)
	}
	c.Writer.Header().Set(cacheControlParam, fmt.Sprintf("max-age=%d", maxOperatorPortCheckAge))
	c.JSON(http.StatusOK, report)
}

// FetchDisperserServiceAvailability godoc
//
//	@Summary	Get status of EigenDA Disperser service.
//	@Tags		ServiceAvailability
//	@Produce	json
//	@Success	200	{object}	ServiceAvailabilityResponse
//	@Failure	400	{object}	ErrorResponse	"error: Bad request"
//	@Failure	404	{object}	ErrorResponse	"error: Not found"
//	@Failure	500	{object}	ErrorResponse	"error: Server error"
//	@Router		/metrics/disperser-service-availability [get]
func (s *server) FetchDisperserServiceAvailability(c *gin.Context) {
	handlerStart := time.Now()
	defer func() {
		s.metrics.ObserveLatency("FetchDisperserServiceAvailability", time.Since(handlerStart))
	}()

	// Check Disperser
	services := []string{"Disperser"}

	s.logger.Info("Getting service availability for", "services", strings.Join(services, ", "))

	availabilityStatuses, err := s.getServiceAvailability(c.Request.Context(), services)
	if err != nil {
		s.metrics.IncrementFailedRequestNum("FetchDisperserServiceAvailability")
		errorResponse(c, err)
		return
	}

	s.metrics.IncrementSuccessfulRequestNum("FetchDisperserServiceAvailability")

	// Set the status code to 503 if any of the services are not serving
	availabilityStatus := http.StatusOK
	for _, status := range availabilityStatuses {
		if status.ServiceStatus == "NOT_SERVING" {
			availabilityStatus = http.StatusServiceUnavailable
			break
		}

		if status.ServiceStatus == "UNKNOWN" {
			availabilityStatus = http.StatusInternalServerError
			break
		}

	}

	c.Writer.Header().Set(cacheControlParam, fmt.Sprintf("max-age=%d", maxDisperserAvailabilityAge))
	c.JSON(availabilityStatus, ServiceAvailabilityResponse{
		Meta: Meta{
			Size: len(availabilityStatuses),
		},
		Data: availabilityStatuses,
	})
}

// FetchChurnerServiceAvailability godoc
//
//	@Summary	Get status of EigenDA churner service.
//	@Tags		Churner ServiceAvailability
//	@Produce	json
//	@Success	200	{object}	ServiceAvailabilityResponse
//	@Failure	400	{object}	ErrorResponse	"error: Bad request"
//	@Failure	404	{object}	ErrorResponse	"error: Not found"
//	@Failure	500	{object}	ErrorResponse	"error: Server error"
//	@Router		/metrics/churner-service-availability [get]
func (s *server) FetchChurnerServiceAvailability(c *gin.Context) {
	handlerStart := time.Now()
	defer func() {
		s.metrics.ObserveLatency("FetchChurnerServiceAvailability", time.Since(handlerStart))
	}()

	// Check Disperser
	services := []string{"Churner"}

	s.logger.Info("Getting service availability for", "services", strings.Join(services, ", "))

	availabilityStatuses, err := s.getServiceAvailability(c.Request.Context(), services)
	if err != nil {
		s.metrics.IncrementFailedRequestNum("FetchChurnerServiceAvailability")
		errorResponse(c, err)
		return
	}

	s.metrics.IncrementSuccessfulRequestNum("FetchChurnerServiceAvailability")

	// Set the status code to 503 if any of the services are not serving
	availabilityStatus := http.StatusOK
	for _, status := range availabilityStatuses {
		if status.ServiceStatus == "NOT_SERVING" {
			availabilityStatus = http.StatusServiceUnavailable
			break
		}

		if status.ServiceStatus == "UNKNOWN" {
			availabilityStatus = http.StatusInternalServerError
			break
		}

	}

	c.Writer.Header().Set(cacheControlParam, fmt.Sprintf("max-age=%d", maxChurnerAvailabilityAge))
	c.JSON(availabilityStatus, ServiceAvailabilityResponse{
		Meta: Meta{
			Size: len(availabilityStatuses),
		},
		Data: availabilityStatuses,
	})
}

// FetchBatcherAvailability godoc
//
//	@Summary	Get status of EigenDA batcher.
//	@Tags		Batcher Availability
//	@Produce	json
//	@Success	200	{object}	ServiceAvailabilityResponse
//	@Failure	400	{object}	ErrorResponse	"error: Bad request"
//	@Failure	404	{object}	ErrorResponse	"error: Not found"
//	@Failure	500	{object}	ErrorResponse	"error: Server error"
//	@Router		/metrics/batcher-service-availability [get]
func (s *server) FetchBatcherAvailability(c *gin.Context) {
	handlerStart := time.Now()
	defer func() {
		s.metrics.ObserveLatency("FetchBatcherAvailability", time.Since(handlerStart))
	}()

	// Check Batcher
	services := []HttpServiceAvailabilityCheck{{"Batcher", s.batcherHealthEndpt}}

	s.logger.Info("Getting service availability for", "service", services[0].ServiceName, "endpoint", services[0].HealthEndPt)

	availabilityStatuses, err := s.getServiceHealth(c.Request.Context(), services)
	if err != nil {
		s.metrics.IncrementFailedRequestNum("FetchBatcherAvailability")
		errorResponse(c, err)
		return
	}

	s.metrics.IncrementSuccessfulRequestNum("FetchBatcherAvailability")

	// Set the status code to 503 if any of the services are not serving
	availabilityStatus := http.StatusOK
	for _, status := range availabilityStatuses {
		if status.ServiceStatus == "NOT_SERVING" {
			availabilityStatus = http.StatusServiceUnavailable
			break
		}

		if status.ServiceStatus == "UNKNOWN" {
			availabilityStatus = http.StatusInternalServerError
			break
		}

	}

	c.Writer.Header().Set(cacheControlParam, fmt.Sprintf("max-age=%d", maxBatcherAvailabilityAge))
	c.JSON(availabilityStatus, ServiceAvailabilityResponse{
		Meta: Meta{
			Size: len(availabilityStatuses),
		},
		Data: availabilityStatuses,
	})
}

func errorResponse(c *gin.Context, err error) {
	_ = c.Error(err)
	var code int
	switch {
	case errors.Is(err, errNotFound):
		code = http.StatusNotFound
	default:
		code = http.StatusInternalServerError
	}
	c.JSON(code, ErrorResponse{
		Error: err.Error(),
	})
}

func run(logger logging.Logger, httpServer *http.Server) <-chan error {
	errChan := make(chan error, 1)
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	go func() {
		<-ctx.Done()

		logger.Info("shutdown signal received")

		defer func() {
			stop()
			close(errChan)
		}()

		if err := httpServer.Shutdown(context.Background()); err != nil {
			errChan <- err
		}
		logger.Info("shutdown completed")
	}()

	go func() {
		logger.Info("server running", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil {
			errChan <- err
		}
	}()

	return errChan
}
