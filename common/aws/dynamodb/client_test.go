package dynamodb_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Layr-Labs/eigenda/common"
	commonaws "github.com/Layr-Labs/eigenda/common/aws"
	commondynamodb "github.com/Layr-Labs/eigenda/common/aws/dynamodb"
	test_utils "github.com/Layr-Labs/eigenda/common/aws/dynamodb/utils"
	"github.com/Layr-Labs/eigenda/inabox/deploy"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"
)

var (
	dockertestPool     *dockertest.Pool
	dockertestResource *dockertest.Resource
	dynamoClient       commondynamodb.Client
	clientConfig       commonaws.ClientConfig

	deployLocalStack bool
	localStackPort   = "4567"
)

func TestMain(m *testing.M) {
	setup(m)
	code := m.Run()
	teardown()
	os.Exit(code)
}

func setup(m *testing.M) {

	deployLocalStack = (os.Getenv("DEPLOY_LOCALSTACK") != "false")
	if !deployLocalStack {
		localStackPort = os.Getenv("LOCALSTACK_PORT")
	}

	if deployLocalStack {
		var err error
		dockertestPool, dockertestResource, err = deploy.StartDockertestWithLocalstackContainer(localStackPort)
		if err != nil {
			teardown()
			panic("failed to start localstack container: " + err.Error())
		}
	}

	loggerConfig := common.DefaultLoggerConfig()
	logger, err := common.NewLogger(loggerConfig)
	if err != nil {
		teardown()
		panic("failed to create logger")
	}

	clientConfig = commonaws.ClientConfig{
		Region:          "us-east-1",
		AccessKey:       "localstack",
		SecretAccessKey: "localstack",
		EndpointURL:     fmt.Sprintf("http://0.0.0.0:%s", localStackPort),
	}

	dynamoClient, err = commondynamodb.NewClient(clientConfig, logger)
	if err != nil {
		teardown()
		panic("failed to create dynamodb client")
	}
}

func teardown() {
	if deployLocalStack {
		deploy.PurgeDockertestResources(dockertestPool, dockertestResource)
	}
}

func createTable(t *testing.T, tableName string) {
	ctx := context.Background()
	tableDescription, err := test_utils.CreateTable(ctx, clientConfig, tableName, &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("MetadataKey"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("BlobStatus"),
				AttributeType: types.ScalarAttributeTypeN, // Assuming BlobStatus is a numeric value
			},
			{
				AttributeName: aws.String("RequestedAt"),
				AttributeType: types.ScalarAttributeTypeN, // Assuming RequestedAt is a string representing a timestamp
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("MetadataKey"),
				KeyType:       types.KeyTypeHash,
			},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("StatusIndex"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("BlobStatus"),
						KeyType:       types.KeyTypeHash,
					},
					{
						AttributeName: aws.String("RequestedAt"),
						KeyType:       types.KeyTypeRange, // Using RequestedAt as sort key
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll, // ProjectionTypeAll means all attributes are projected into the index
				},
				ProvisionedThroughput: &types.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(10),
					WriteCapacityUnits: aws.Int64(10),
				},
			},
		},
		TableName: aws.String(tableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, tableDescription)
}

func TestBasicOperations(t *testing.T) {
	tableName := "Processing"
	createTable(t, tableName)

	ctx := context.Background()
	item := commondynamodb.Item{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key"},
		"RequestedAt": &types.AttributeValueMemberN{Value: "123"},
		"SecurityParams": &types.AttributeValueMemberL{
			Value: []types.AttributeValue{
				&types.AttributeValueMemberM{
					Value: map[string]types.AttributeValue{
						"QuorumID":           &types.AttributeValueMemberN{Value: "0"},
						"AdversaryThreshold": &types.AttributeValueMemberN{Value: "80"},
					},
				},
				&types.AttributeValueMemberM{
					Value: map[string]types.AttributeValue{
						"QuorumID":           &types.AttributeValueMemberN{Value: "1"},
						"AdversaryThreshold": &types.AttributeValueMemberN{Value: "70"},
					},
				},
			},
		},
		"BlobSize": &types.AttributeValueMemberN{Value: "123"},
		"BlobKey":  &types.AttributeValueMemberS{Value: "blob1"},
		"Status":   &types.AttributeValueMemberS{Value: "Processing"},
	}
	err := dynamoClient.PutItem(ctx, tableName, item)
	assert.NoError(t, err)

	fetchedItem, err := dynamoClient.GetItem(ctx, tableName, commondynamodb.Key{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key"},
	})
	assert.NoError(t, err)

	assert.Equal(t, "key", fetchedItem["MetadataKey"].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "123", fetchedItem["RequestedAt"].(*types.AttributeValueMemberN).Value)
	assert.Equal(t, "Processing", fetchedItem["Status"].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "blob1", fetchedItem["BlobKey"].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "123", fetchedItem["BlobSize"].(*types.AttributeValueMemberN).Value)
	assert.Equal(t, []types.AttributeValue{
		&types.AttributeValueMemberM{
			Value: map[string]types.AttributeValue{
				"QuorumID":           &types.AttributeValueMemberN{Value: "0"},
				"AdversaryThreshold": &types.AttributeValueMemberN{Value: "80"},
			},
		},
		&types.AttributeValueMemberM{
			Value: map[string]types.AttributeValue{
				"QuorumID":           &types.AttributeValueMemberN{Value: "1"},
				"AdversaryThreshold": &types.AttributeValueMemberN{Value: "70"},
			},
		},
	}, fetchedItem["SecurityParams"].(*types.AttributeValueMemberL).Value)

	// Attempt to put an item with the same key
	err = dynamoClient.PutItemWithCondition(ctx, tableName, commondynamodb.Item{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key"},
		"RequestedAt": &types.AttributeValueMemberN{Value: "456"},
	}, "attribute_not_exists(MetadataKey)", nil, nil)
	assert.ErrorIs(t, err, commondynamodb.ErrConditionFailed)
	fetchedItem, err = dynamoClient.GetItem(ctx, tableName, commondynamodb.Key{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key"},
	})
	assert.NoError(t, err)
	// Shouldn't have been updated
	assert.Equal(t, "123", fetchedItem["RequestedAt"].(*types.AttributeValueMemberN).Value)

	_, err = dynamoClient.UpdateItem(ctx, tableName, commondynamodb.Key{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key"},
	}, commondynamodb.Item{
		"Status": &types.AttributeValueMemberS{Value: "Confirmed"},
		"BatchHeaderHash": &types.AttributeValueMemberS{
			Value: "0x123",
		},
		"BlobIndex": &types.AttributeValueMemberN{
			Value: "0",
		},
	})
	assert.NoError(t, err)

	// Attempt to update the item with invalid condition
	_, err = dynamoClient.UpdateItemWithCondition(ctx, tableName, commondynamodb.Key{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key"},
	}, commondynamodb.Item{
		"RequestedAt": &types.AttributeValueMemberN{Value: "456"},
	}, expression.Name("Status").In(expression.Value("Dispersing")))
	assert.Error(t, err)

	// Attempt to update the item with valid condition
	_, err = dynamoClient.UpdateItemWithCondition(ctx, tableName, commondynamodb.Key{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key"},
	}, commondynamodb.Item{
		"RequestedAt": &types.AttributeValueMemberN{Value: "456"},
	}, expression.Name("Status").In(expression.Value("Confirmed")))
	assert.NoError(t, err)

	_, err = dynamoClient.IncrementBy(ctx, tableName, commondynamodb.Key{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key"},
	}, "BlobSize", 1000)
	assert.NoError(t, err)

	fetchedItem, err = dynamoClient.GetItem(ctx, tableName, commondynamodb.Key{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "key", fetchedItem["MetadataKey"].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "Confirmed", fetchedItem["Status"].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "0x123", fetchedItem["BatchHeaderHash"].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "0", fetchedItem["BlobIndex"].(*types.AttributeValueMemberN).Value)
	assert.Equal(t, "1123", fetchedItem["BlobSize"].(*types.AttributeValueMemberN).Value)
	assert.Equal(t, "456", fetchedItem["RequestedAt"].(*types.AttributeValueMemberN).Value)

	err = dynamoClient.DeleteTable(ctx, tableName)
	assert.NoError(t, err)
}

func TestBatchOperations(t *testing.T) {
	tableName := "Processing"
	createTable(t, tableName)

	ctx := context.Background()
	numItems := 33
	items := make([]commondynamodb.Item, numItems)
	expectedBlobKeys := make([]string, numItems)
	expectedMetadataKeys := make([]string, numItems)
	for i := 0; i < numItems; i += 1 {
		items[i] = commondynamodb.Item{
			"MetadataKey": &types.AttributeValueMemberS{Value: fmt.Sprintf("key%d", i)},
			"BlobKey":     &types.AttributeValueMemberS{Value: fmt.Sprintf("blob%d", i)},
		}
		expectedBlobKeys[i] = fmt.Sprintf("blob%d", i)
		expectedMetadataKeys[i] = fmt.Sprintf("key%d", i)
	}
	unprocessed, err := dynamoClient.PutItems(ctx, tableName, items)
	assert.NoError(t, err)
	assert.Len(t, unprocessed, 0)

	fetchedItem, err := dynamoClient.GetItem(ctx, tableName, commondynamodb.Key{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key0"},
	})
	assert.NoError(t, err)
	assert.NotNil(t, fetchedItem)
	assert.Equal(t, fetchedItem["BlobKey"].(*types.AttributeValueMemberS).Value, "blob0")

	fetchedItem, err = dynamoClient.GetItem(ctx, tableName, commondynamodb.Key{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key1"},
	})
	assert.NoError(t, err)
	assert.NotNil(t, fetchedItem)
	assert.Equal(t, fetchedItem["BlobKey"].(*types.AttributeValueMemberS).Value, "blob1")

	keys := make([]commondynamodb.Key, numItems)
	for i := 0; i < numItems; i += 1 {
		keys[i] = commondynamodb.Key{
			"MetadataKey": &types.AttributeValueMemberS{Value: fmt.Sprintf("key%d", i)},
		}
	}

	fetchedItems, err := dynamoClient.GetItems(ctx, tableName, keys, true)
	assert.NoError(t, err)
	assert.Len(t, fetchedItems, numItems)
	blobKeys := make([]string, numItems)
	metadataKeys := make([]string, numItems)
	for i := 0; i < numItems; i += 1 {
		blobKeys[i] = fetchedItems[i]["BlobKey"].(*types.AttributeValueMemberS).Value
		metadataKeys[i] = fetchedItems[i]["MetadataKey"].(*types.AttributeValueMemberS).Value
	}
	assert.ElementsMatch(t, expectedBlobKeys, blobKeys)
	assert.ElementsMatch(t, expectedMetadataKeys, metadataKeys)

	unprocessedKeys, err := dynamoClient.DeleteItems(ctx, tableName, keys)
	assert.NoError(t, err)
	assert.Len(t, unprocessedKeys, 0)

	fetchedItem, err = dynamoClient.GetItem(ctx, tableName, commondynamodb.Key{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key0"},
	})
	assert.NoError(t, err)
	assert.Len(t, fetchedItem, 0)

	fetchedItem, err = dynamoClient.GetItem(ctx, tableName, commondynamodb.Key{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key1"},
	})
	assert.NoError(t, err)
	assert.Len(t, fetchedItem, 0)
}

func TestQueryIndex(t *testing.T) {
	tableName := "ProcessingQueryIndex"
	createTable(t, tableName)
	indexName := "StatusIndex"

	ctx := context.Background()
	numItems := 30
	items := make([]commondynamodb.Item, numItems)
	for i := 0; i < numItems; i += 1 {
		items[i] = commondynamodb.Item{
			"MetadataKey": &types.AttributeValueMemberS{Value: fmt.Sprintf("key%d", i)},
			"BlobKey":     &types.AttributeValueMemberS{Value: fmt.Sprintf("blob%d", i)},
			"BlobSize":    &types.AttributeValueMemberN{Value: "123"},
			"BlobStatus":  &types.AttributeValueMemberN{Value: "0"},
			"RequestedAt": &types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
		}
	}
	unprocessed, err := dynamoClient.PutItems(ctx, tableName, items)
	assert.NoError(t, err)
	assert.Len(t, unprocessed, 0)

	queryResult, err := dynamoClient.QueryIndex(ctx, tableName, indexName, "BlobStatus = :status", commondynamodb.ExpressionValues{
		":status": &types.AttributeValueMemberN{
			Value: "0",
		}})
	assert.NoError(t, err)
	assert.Equal(t, len(queryResult), 30)
}

func TestQueryIndexCount(t *testing.T) {
	tableName := "ProcessingQueryIndexCount"
	createTable(t, tableName)
	indexName := "StatusIndex"

	ctx := context.Background()
	numItemsProcessing := 10
	items1 := make([]commondynamodb.Item, numItemsProcessing)
	for i := 0; i < numItemsProcessing; i += 1 {
		items1[i] = commondynamodb.Item{
			"MetadataKey": &types.AttributeValueMemberS{Value: fmt.Sprintf("key%d", i)},
			"BlobKey":     &types.AttributeValueMemberS{Value: fmt.Sprintf("blob%d", i)},
			"BlobSize":    &types.AttributeValueMemberN{Value: "123"},
			"BlobStatus":  &types.AttributeValueMemberN{Value: "0"},
			"RequestedAt": &types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
		}
	}

	numItemsConfirmed := 20
	items2 := make([]commondynamodb.Item, numItemsConfirmed)
	for i := 0; i < numItemsConfirmed; i += 1 {
		items2[i] = commondynamodb.Item{
			"MetadataKey": &types.AttributeValueMemberS{Value: fmt.Sprintf("key%d", i+numItemsProcessing)},
			"BlobKey":     &types.AttributeValueMemberS{Value: fmt.Sprintf("blob%d", i+numItemsProcessing)},
			"BlobSize":    &types.AttributeValueMemberN{Value: "123"},
			"BlobStatus":  &types.AttributeValueMemberN{Value: "1"},
			"RequestedAt": &types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
		}
	}

	unprocessed, err := dynamoClient.PutItems(ctx, tableName, items1)
	assert.NoError(t, err)
	assert.Len(t, unprocessed, 0)

	unprocessed, err = dynamoClient.PutItems(ctx, tableName, items2)
	assert.NoError(t, err)
	assert.Len(t, unprocessed, 0)

	count, err := dynamoClient.QueryIndexCount(ctx, tableName, indexName, "BlobStatus = :status", commondynamodb.ExpressionValues{
		":status": &types.AttributeValueMemberN{
			Value: "0",
		}})
	assert.NoError(t, err)
	assert.Equal(t, int(count), 10)

	count, err = dynamoClient.QueryIndexCount(ctx, tableName, indexName, "BlobStatus = :status", commondynamodb.ExpressionValues{
		":status": &types.AttributeValueMemberN{
			Value: "1",
		}})
	assert.NoError(t, err)
	assert.Equal(t, int(count), 20)
}

func TestQueryIndexPaginationSingleItem(t *testing.T) {
	tableName := "ProcessingWithPaginationSingleItem"
	createTable(t, tableName)
	indexName := "StatusIndex"

	ctx := context.Background()
	requestedAt := time.Now().Unix()
	item := commondynamodb.Item{
		"MetadataKey": &types.AttributeValueMemberS{Value: fmt.Sprintf("key%d", 0)},
		"BlobKey":     &types.AttributeValueMemberS{Value: fmt.Sprintf("blob%d", 0)},
		"BlobSize":    &types.AttributeValueMemberN{Value: "123"},
		"BlobStatus":  &types.AttributeValueMemberN{Value: "0"},
		"RequestedAt": &types.AttributeValueMemberN{Value: strconv.FormatInt(requestedAt, 10)},
	}
	err := dynamoClient.PutItem(ctx, tableName, item)
	assert.NoError(t, err)

	queryResult, err := dynamoClient.QueryIndexWithPagination(ctx, tableName, indexName, "BlobStatus = :status", commondynamodb.ExpressionValues{
		":status": &types.AttributeValueMemberN{
			Value: "0",
		}}, 1, nil, true)
	assert.NoError(t, err)
	assert.Len(t, queryResult.Items, 1)
	assert.Equal(t, "key0", queryResult.Items[0]["MetadataKey"].(*types.AttributeValueMemberS).Value)
	assert.NotNil(t, queryResult.LastEvaluatedKey)
	assert.Equal(t, "key0", queryResult.LastEvaluatedKey["MetadataKey"].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "0", queryResult.LastEvaluatedKey["BlobStatus"].(*types.AttributeValueMemberN).Value)

	// Save Last Evaluated Key
	lastEvaluatedKey := queryResult.LastEvaluatedKey

	// Get the next item using LastEvaluatedKey expect to be nil
	queryResult, err = dynamoClient.QueryIndexWithPagination(ctx, tableName, indexName, "BlobStatus = :status", commondynamodb.ExpressionValues{
		":status": &types.AttributeValueMemberN{
			Value: "0",
		}}, 1, lastEvaluatedKey, true)
	assert.NoError(t, err)
	assert.Nil(t, queryResult.Items)
	assert.Nil(t, queryResult.LastEvaluatedKey)
}

func TestQueryIndexPaginationItemNoLimit(t *testing.T) {
	tableName := "ProcessingWithNoPaginationLimit"
	createTable(t, tableName)
	indexName := "StatusIndex"

	ctx := context.Background()
	numItems := 30
	for i := 0; i < numItems; i += 1 {
		requestedAt := time.Now().Add(-time.Duration(3*i) * time.Second).Unix()

		// Create new item
		item := commondynamodb.Item{
			"MetadataKey": &types.AttributeValueMemberS{Value: fmt.Sprintf("key%d", i)},
			"BlobKey":     &types.AttributeValueMemberS{Value: fmt.Sprintf("blob%d", i)},
			"BlobSize":    &types.AttributeValueMemberN{Value: "123"},
			"BlobStatus":  &types.AttributeValueMemberN{Value: "0"},
			"RequestedAt": &types.AttributeValueMemberN{Value: strconv.FormatInt(requestedAt, 10)},
		}
		err := dynamoClient.PutItem(ctx, tableName, item)
		assert.NoError(t, err)
	}

	queryResult, err := dynamoClient.QueryIndexWithPagination(ctx, tableName, indexName, "BlobStatus = :status", commondynamodb.ExpressionValues{
		":status": &types.AttributeValueMemberN{
			Value: "0",
		}}, 0, nil, true)
	assert.NoError(t, err)
	assert.Len(t, queryResult.Items, 30)
	assert.Equal(t, "key29", queryResult.Items[0]["MetadataKey"].(*types.AttributeValueMemberS).Value)
	assert.Nil(t, queryResult.LastEvaluatedKey)

	// Save Last Evaluated Key
	lastEvaluatedKey := queryResult.LastEvaluatedKey

	// Get the next item using LastEvaluatedKey expect to be nil
	queryResult, err = dynamoClient.QueryIndexWithPagination(ctx, tableName, indexName, "BlobStatus = :status", commondynamodb.ExpressionValues{
		":status": &types.AttributeValueMemberN{
			Value: "0",
		}}, 2, lastEvaluatedKey, true)
	assert.NoError(t, err)
	assert.Len(t, queryResult.Items, 2)
	assert.Equal(t, "key29", queryResult.Items[0]["MetadataKey"].(*types.AttributeValueMemberS).Value)
	assert.NotNil(t, queryResult.LastEvaluatedKey)
}

func TestQueryIndexPaginationNoStoredItems(t *testing.T) {
	tableName := "ProcessingWithPaginationNoItem"
	createTable(t, tableName)
	indexName := "StatusIndex"

	ctx := context.Background()
	queryResult, err := dynamoClient.QueryIndexWithPagination(ctx, tableName, indexName, "BlobStatus = :status", commondynamodb.ExpressionValues{
		":status": &types.AttributeValueMemberN{
			Value: "0",
		}}, 1, nil, true)
	assert.NoError(t, err)
	assert.Nil(t, queryResult.Items)
	assert.Nil(t, queryResult.LastEvaluatedKey)
}

func TestQueryIndexPagination(t *testing.T) {
	tableName := "ProcessingWithPagination"
	createTable(t, tableName)
	indexName := "StatusIndex"

	ctx := context.Background()
	numItems := 30
	for i := 0; i < numItems; i += 1 {
		// Noticed same timestamp for multiple items which resulted in key28
		// being returned when 10 items were queried as first item,hence multiplying
		// by random number 3 here to avoid such a situation
		// requestedAt: 1705040877
		// metadataKey: key28
		// BlobKey: blob28
		// requestedAt: 1705040877
		// metadataKey: key29
		// BlobKey: blob29
		requestedAt := time.Now().Add(-time.Duration(3*i) * time.Second).Unix()

		// Create new item
		item := commondynamodb.Item{
			"MetadataKey": &types.AttributeValueMemberS{Value: fmt.Sprintf("key%d", i)},
			"BlobKey":     &types.AttributeValueMemberS{Value: fmt.Sprintf("blob%d", i)},
			"BlobSize":    &types.AttributeValueMemberN{Value: "123"},
			"BlobStatus":  &types.AttributeValueMemberN{Value: "0"},
			"RequestedAt": &types.AttributeValueMemberN{Value: strconv.FormatInt(requestedAt, 10)},
		}
		err := dynamoClient.PutItem(ctx, tableName, item)
		assert.NoError(t, err)
	}

	queryResult, err := dynamoClient.QueryIndexWithPagination(ctx, tableName, indexName, "BlobStatus = :status", commondynamodb.ExpressionValues{
		":status": &types.AttributeValueMemberN{
			Value: "0",
		}}, 10, nil, true)
	assert.NoError(t, err)
	assert.Len(t, queryResult.Items, 10)
	assert.Equal(t, "key29", queryResult.Items[0]["MetadataKey"].(*types.AttributeValueMemberS).Value)
	assert.NotNil(t, queryResult.LastEvaluatedKey)
	assert.Equal(t, "key20", queryResult.LastEvaluatedKey["MetadataKey"].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "0", queryResult.LastEvaluatedKey["BlobStatus"].(*types.AttributeValueMemberN).Value)

	// Get the next 10 items
	queryResult, err = dynamoClient.QueryIndexWithPagination(ctx, tableName, indexName, "BlobStatus = :status", commondynamodb.ExpressionValues{
		":status": &types.AttributeValueMemberN{
			Value: "0",
		}}, 10, queryResult.LastEvaluatedKey, true)
	assert.NoError(t, err)
	assert.Len(t, queryResult.Items, 10)
	assert.Equal(t, "key10", queryResult.LastEvaluatedKey["MetadataKey"].(*types.AttributeValueMemberS).Value)

	// Get the last 10 items
	queryResult, err = dynamoClient.QueryIndexWithPagination(ctx, tableName, indexName, "BlobStatus = :status", commondynamodb.ExpressionValues{
		":status": &types.AttributeValueMemberN{
			Value: "0",
		}}, 10, queryResult.LastEvaluatedKey, true)
	assert.NoError(t, err)
	assert.Len(t, queryResult.Items, 10)
	assert.Equal(t, "key0", queryResult.LastEvaluatedKey["MetadataKey"].(*types.AttributeValueMemberS).Value)

	// Empty result Since all items are processed
	queryResult, err = dynamoClient.QueryIndexWithPagination(ctx, tableName, indexName, "BlobStatus = :status", commondynamodb.ExpressionValues{
		":status": &types.AttributeValueMemberN{
			Value: "0",
		}}, 10, queryResult.LastEvaluatedKey, true)
	assert.NoError(t, err)
	assert.Len(t, queryResult.Items, 0)
	assert.Nil(t, queryResult.LastEvaluatedKey)
}

func TestQueryIndexWithPaginationForBatch(t *testing.T) {
	tableName := "ProcessingWithPaginationForBatch"
	createTable(t, tableName)
	indexName := "StatusIndex"

	ctx := context.Background()
	numItems := 30
	items := make([]commondynamodb.Item, numItems)
	for i := 0; i < numItems; i += 1 {
		items[i] = commondynamodb.Item{
			"MetadataKey": &types.AttributeValueMemberS{Value: fmt.Sprintf("key%d", i)},
			"BlobKey":     &types.AttributeValueMemberS{Value: fmt.Sprintf("blob%d", i)},
			"BlobSize":    &types.AttributeValueMemberN{Value: "123"},
			"BlobStatus":  &types.AttributeValueMemberN{Value: "0"},
			"RequestedAt": &types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
		}
	}
	unprocessed, err := dynamoClient.PutItems(ctx, tableName, items)
	assert.NoError(t, err)
	assert.Len(t, unprocessed, 0)

	// Get First 10 items
	queryResult, err := dynamoClient.QueryIndexWithPagination(ctx, tableName, indexName, "BlobStatus = :status", commondynamodb.ExpressionValues{
		":status": &types.AttributeValueMemberN{
			Value: "0",
		}}, 10, nil, true)
	assert.NoError(t, err)
	assert.Len(t, queryResult.Items, 10)

	// Get the next 10 items
	queryResult, err = dynamoClient.QueryIndexWithPagination(ctx, tableName, indexName, "BlobStatus = :status", commondynamodb.ExpressionValues{
		":status": &types.AttributeValueMemberN{
			Value: "0",
		}}, 10, queryResult.LastEvaluatedKey, true)
	assert.NoError(t, err)
	assert.Len(t, queryResult.Items, 10)

	// Get the last 10 items
	queryResult, err = dynamoClient.QueryIndexWithPagination(ctx, tableName, indexName, "BlobStatus = :status", commondynamodb.ExpressionValues{
		":status": &types.AttributeValueMemberN{
			Value: "0",
		}}, 10, queryResult.LastEvaluatedKey, true)
	assert.NoError(t, err)
	assert.Len(t, queryResult.Items, 10)

	// Empty result Since all items are processed
	queryResult, err = dynamoClient.QueryIndexWithPagination(ctx, tableName, indexName, "BlobStatus = :status", commondynamodb.ExpressionValues{
		":status": &types.AttributeValueMemberN{
			Value: "0",
		}}, 10, queryResult.LastEvaluatedKey, true)
	assert.NoError(t, err)
	assert.Len(t, queryResult.Items, 0)
	assert.Nil(t, queryResult.LastEvaluatedKey)
}

func TestQueryWithInput(t *testing.T) {
	tableName := "ProcessingQueryWithInput"
	createTable(t, tableName)

	ctx := context.Background()
	numItems := 30
	items := make([]commondynamodb.Item, numItems)
	for i := 0; i < numItems; i++ {
		requestedAt := time.Now().Add(-time.Duration(i) * time.Minute).Unix()
		items[i] = commondynamodb.Item{
			"MetadataKey": &types.AttributeValueMemberS{Value: fmt.Sprintf("key%d", i)},
			"BlobKey":     &types.AttributeValueMemberS{Value: fmt.Sprintf("blob%d", i)},
			"BlobSize":    &types.AttributeValueMemberN{Value: "123"},
			"BlobStatus":  &types.AttributeValueMemberN{Value: "0"},
			"RequestedAt": &types.AttributeValueMemberN{Value: strconv.FormatInt(requestedAt, 10)},
		}
	}
	unprocessed, err := dynamoClient.PutItems(ctx, tableName, items)
	assert.NoError(t, err)
	assert.Len(t, unprocessed, 0)

	// Test forward order with limit
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		IndexName:              aws.String("StatusIndex"),
		KeyConditionExpression: aws.String("BlobStatus = :status"),
		ExpressionAttributeValues: commondynamodb.ExpressionValues{
			":status": &types.AttributeValueMemberN{Value: "0"},
		},
		ScanIndexForward: aws.Bool(true),
		Limit:            aws.Int32(10),
	}
	queryResult, err := dynamoClient.QueryWithInput(ctx, queryInput)
	assert.NoError(t, err)
	assert.Len(t, queryResult, 10)
	// Check if the items are in ascending order
	for i := 0; i < len(queryResult)-1; i++ {
		assert.True(t, queryResult[i]["RequestedAt"].(*types.AttributeValueMemberN).Value <= queryResult[i+1]["RequestedAt"].(*types.AttributeValueMemberN).Value)
	}

	// Test reverse order with limit
	queryInput = &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		IndexName:              aws.String("StatusIndex"),
		KeyConditionExpression: aws.String("BlobStatus = :status"),
		ExpressionAttributeValues: commondynamodb.ExpressionValues{
			":status": &types.AttributeValueMemberN{Value: "0"},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int32(10),
	}
	queryResult, err = dynamoClient.QueryWithInput(ctx, queryInput)
	assert.NoError(t, err)
	assert.Len(t, queryResult, 10)
	// Check if the items are in descending order
	for i := 0; i < len(queryResult)-1; i++ {
		assert.True(t, queryResult[i]["RequestedAt"].(*types.AttributeValueMemberN).Value >= queryResult[i+1]["RequestedAt"].(*types.AttributeValueMemberN).Value)
	}

	// Test with a smaller limit
	queryInput = &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		IndexName:              aws.String("StatusIndex"),
		KeyConditionExpression: aws.String("BlobStatus = :status"),
		ExpressionAttributeValues: commondynamodb.ExpressionValues{
			":status": &types.AttributeValueMemberN{Value: "0"},
		},
		Limit: aws.Int32(5),
	}
	queryResult, err = dynamoClient.QueryWithInput(ctx, queryInput)
	assert.NoError(t, err)
	assert.Len(t, queryResult, 5)

	// Test with a limit larger than the number of items
	queryInput = &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		IndexName:              aws.String("StatusIndex"),
		KeyConditionExpression: aws.String("BlobStatus = :status"),
		ExpressionAttributeValues: commondynamodb.ExpressionValues{
			":status": &types.AttributeValueMemberN{Value: "0"},
		},
		Limit: aws.Int32(50),
	}
	queryResult, err = dynamoClient.QueryWithInput(ctx, queryInput)
	assert.NoError(t, err)
	assert.Len(t, queryResult, 30) // Should return all items
}

func TestPutItemWithConditionAndReturn(t *testing.T) {
	tableName := "PutItemWithConditionAndReturn"
	createTable(t, tableName)

	ctx := context.Background()

	// Create an initial item
	initialItem := commondynamodb.Item{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key1"},
		"BlobKey":     &types.AttributeValueMemberS{Value: "blob1"},
		"BlobSize":    &types.AttributeValueMemberN{Value: "123"},
		"BlobStatus":  &types.AttributeValueMemberN{Value: "0"},
		"Status":      &types.AttributeValueMemberS{Value: "Processing"},
	}
	err := dynamoClient.PutItem(ctx, tableName, initialItem)
	assert.NoError(t, err)

	// Case 1: Condition succeeds, should return old item
	updatedItem := commondynamodb.Item{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key1"},
		"BlobKey":     &types.AttributeValueMemberS{Value: "blob1-updated"},
		"BlobSize":    &types.AttributeValueMemberN{Value: "456"},
		"BlobStatus":  &types.AttributeValueMemberN{Value: "1"},
		"Status":      &types.AttributeValueMemberS{Value: "Updated"},
	}

	// Condition that should succeed (Status = Processing)
	oldItem, err := dynamoClient.PutItemWithConditionAndReturn(
		ctx,
		tableName,
		updatedItem,
		"#status = :status",
		map[string]string{"#status": "Status"},
		map[string]types.AttributeValue{":status": &types.AttributeValueMemberS{Value: "Processing"}},
	)
	assert.NoError(t, err)
	assert.NotNil(t, oldItem)
	assert.Equal(t, "key1", oldItem["MetadataKey"].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "blob1", oldItem["BlobKey"].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "123", oldItem["BlobSize"].(*types.AttributeValueMemberN).Value)
	assert.Equal(t, "Processing", oldItem["Status"].(*types.AttributeValueMemberS).Value)

	// Verify the update was applied
	fetchedItem, err := dynamoClient.GetItem(ctx, tableName, commondynamodb.Key{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key1"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "blob1-updated", fetchedItem["BlobKey"].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "456", fetchedItem["BlobSize"].(*types.AttributeValueMemberN).Value)
	assert.Equal(t, "Updated", fetchedItem["Status"].(*types.AttributeValueMemberS).Value)

	// Case 2: Condition fails, should return ErrConditionFailed
	newItem := commondynamodb.Item{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key1"},
		"BlobKey":     &types.AttributeValueMemberS{Value: "blob1-newer"},
		"Status":      &types.AttributeValueMemberS{Value: "Newer"},
	}

	// Condition that should fail (Status = Processing, but it's now "Updated")
	oldItem, err = dynamoClient.PutItemWithConditionAndReturn(
		ctx,
		tableName,
		newItem,
		"#status = :status",
		map[string]string{"#status": "Status"},
		map[string]types.AttributeValue{":status": &types.AttributeValueMemberS{Value: "Processing"}},
	)
	assert.ErrorIs(t, err, commondynamodb.ErrConditionFailed)
	assert.Nil(t, oldItem)

	// Case 3: Put item that doesn't exist yet, with condition
	nonExistingItem := commondynamodb.Item{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key2"},
		"BlobKey":     &types.AttributeValueMemberS{Value: "blob2"},
		"Status":      &types.AttributeValueMemberS{Value: "New"},
	}

	// Condition: attribute_not_exists(MetadataKey)
	oldItem, err = dynamoClient.PutItemWithConditionAndReturn(
		ctx,
		tableName,
		nonExistingItem,
		"attribute_not_exists(MetadataKey)",
		nil,
		nil,
	)
	assert.NoError(t, err)
	assert.Empty(t, oldItem)

	// Verify the new item was created
	fetchedItem, err = dynamoClient.GetItem(ctx, tableName, commondynamodb.Key{
		"MetadataKey": &types.AttributeValueMemberS{Value: "key2"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "blob2", fetchedItem["BlobKey"].(*types.AttributeValueMemberS).Value)
	assert.Equal(t, "New", fetchedItem["Status"].(*types.AttributeValueMemberS).Value)

	err = dynamoClient.DeleteTable(ctx, tableName)
	assert.NoError(t, err)
}
