package v2

import (
	"fmt"

	pb "github.com/Layr-Labs/eigenda/api/grpc/disperser/v2"
	corev2 "github.com/Layr-Labs/eigenda/core/v2"
	"github.com/Layr-Labs/eigenda/encoding"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

type BlobStatus uint

const (
	Queued BlobStatus = iota
	Encoded
	GatheringSignatures
	Complete
	Failed
)

func (s BlobStatus) String() string {
	switch s {
	case Queued:
		return "Queued"
	case Encoded:
		return "Encoded"
	case GatheringSignatures:
		return "Gathering Signatures"
	case Complete:
		return "Complete"
	case Failed:
		return "Failed"
	default:
		return "Unknown"
	}
}

func (s BlobStatus) ToProfobuf() pb.BlobStatus {
	switch s {
	case Queued:
		return pb.BlobStatus_QUEUED
	case Encoded:
		return pb.BlobStatus_ENCODED
	case GatheringSignatures:
		return pb.BlobStatus_GATHERING_SIGNATURES
	case Complete:
		return pb.BlobStatus_COMPLETE
	case Failed:
		return pb.BlobStatus_FAILED
	default:
		return pb.BlobStatus_UNKNOWN
	}
}

func BlobStatusFromProtobuf(s pb.BlobStatus) (BlobStatus, error) {
	switch s {
	case pb.BlobStatus_QUEUED:
		return Queued, nil
	case pb.BlobStatus_ENCODED:
		return Encoded, nil
	case pb.BlobStatus_GATHERING_SIGNATURES:
		return GatheringSignatures, nil
	case pb.BlobStatus_COMPLETE:
		return Complete, nil
	case pb.BlobStatus_FAILED:
		return Failed, nil
	default:
		return 0, fmt.Errorf("unknown blob status: %v", s)
	}
}

// BlobMetadata is an internal representation of a blob's metadata.
type BlobMetadata struct {
	BlobHeader *corev2.BlobHeader
	Signature  []byte

	// BlobStatus indicates the current status of the blob
	BlobStatus BlobStatus
	// Expiry is Unix timestamp of the blob expiry in seconds from epoch
	Expiry uint64
	// NumRetries is the number of times the blob has been retried
	NumRetries uint
	// BlobSize is the size of the blob in bytes
	BlobSize uint64
	// RequestedAt is the Unix timestamp of when the blob was requested in nanoseconds
	RequestedAt uint64
	// UpdatedAt is the Unix timestamp of when the blob was last updated in _nanoseconds_
	UpdatedAt uint64

	*encoding.FragmentInfo
}

// BlobAttestationInfo describes the attestation information for a blob regarding to the batch
// that the blob belongs to and the validators' attestation to that batch.
//
// Note: for a blob, there will be at most one attested/signed batch that contains the blob.
type BlobAttestationInfo struct {
	InclusionInfo *corev2.BlobInclusionInfo
	Attestation   *corev2.Attestation
}

// Account represents account information from the Account table
type Account struct {
	Address   gethcommon.Address `json:"address"`
	UpdatedAt uint64             `json:"updated_at"` // unix timestamp in seconds
}
