package core_types_test

import (
	"errors"
	"testing"
	"time"

	"empower1/internal/nexuserrors"
	pb "empower1/pkg/core_types"
	"github.com/google/uuid"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// checkTransactionRecordError helper
func checkTransactionRecordError(t *testing.T, gotErr, wantErrType error, wantErr bool) {
	t.Helper()
	if wantErr {
		if gotErr == nil {
			t.Errorf("expected an error but got nil")
			return
		}
		if wantErrType == nil {
			t.Logf("warning: wantErr is true but wantErrType is nil. Error received: %v", gotErr)
			return
		}
		if !errors.Is(gotErr, wantErrType) {
			t.Errorf("got error '%v' (type %T), want error to wrap or be of type %T (base error: '%v')", gotErr, gotErr, wantErrType, wantErrType)
		}
	} else {
		if gotErr != nil {
			t.Errorf("did not expect an error but got: %v (type %T)", gotErr, gotErr)
		}
	}
}


func TestTransactionRecord_Validate(t *testing.T) {
	validTxUUID := uuid.New().String()
	validFromUserUUID := uuid.New().String()
	validToUserUUID := uuid.New().String()
	validAssetUUID := uuid.New().String()
	now := time.Now().UTC()

	validTxRecordAsset := func() *pb.TransactionRecord {
		return &pb.TransactionRecord{
			TransactionId:   validTxUUID,
			FromUserId:      validFromUserUUID,
			ToUserId:        validToUserUUID,
			AssetId:         validAssetUUID,
			TransactionType: pb.DigiSocialTransactionType_DIGISOCIAL_TRANSACTION_TYPE_TRANSFER_ASSET,
			ValueOrAmount:   "1",
			CurrencyOrTokenId: validAssetUUID,
			Timestamp:       timestamppb.New(now),
			Status:          pb.DigiSocialTransactionStatus_DIGISOCIAL_TRANSACTION_STATUS_COMPLETED,
			MemoOrDescription: "Valid asset transfer.",
			Details:          map[string]string{"network_fee": "0.01"},
		}
	}

	validFTTransferRecord := func() *pb.TransactionRecord {
		return &pb.TransactionRecord{
			TransactionId:   uuid.New().String(),
			FromUserId:      validFromUserUUID,
			ToUserId:        validToUserUUID,
			TransactionType: pb.DigiSocialTransactionType_DIGISOCIAL_TRANSACTION_TYPE_TRANSFER_FT,
			ValueOrAmount:   "100.50",
			CurrencyOrTokenId: "POINTS_TOKEN",
			Timestamp:       timestamppb.New(now),
			Status:          pb.DigiSocialTransactionStatus_DIGISOCIAL_TRANSACTION_STATUS_COMPLETED,
		}
	}

	tests := []struct {
		name            string
		record          *pb.TransactionRecord
		wantErr         bool
		expectedErrType error
	}{
		{name: "valid asset transfer record", record: validTxRecordAsset(), wantErr: false},
		{name: "valid FT transfer record", record: validFTTransferRecord(), wantErr: false},
		// TransactionId Tests
		{
			name: "missing TransactionId",
			record: func() *pb.TransactionRecord { r := validTxRecordAsset(); r.TransactionId = ""; return r }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		// FromUserId Tests
		{
			name: "missing FromUserId",
			record: func() *pb.TransactionRecord { r := validTxRecordAsset(); r.FromUserId = ""; return r }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		// ToUserId Tests
		{
			name: "missing ToUserId",
			record: func() *pb.TransactionRecord { r := validTxRecordAsset(); r.ToUserId = ""; return r }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		// AssetId Tests (optional, but if present, must be valid UUID)
		{
			name: "invalid AssetId format if present",
			record: func() *pb.TransactionRecord { r := validTxRecordAsset(); r.AssetId = "not-a-uuid"; return r }(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidUUID,
		},
		// TransactionType Tests
		{
			name: "invalid TransactionType enum",
			record: func() *pb.TransactionRecord { r := validTxRecordAsset(); r.TransactionType = pb.DigiSocialTransactionType(99); return r }(),
			wantErr: true, expectedErrType: nexuserrors.ErrUnknownEnumValue,
		},
		// ValueOrAmount Tests
		{
			name: "missing ValueOrAmount for TRANSFER_FT",
			record: func() *pb.TransactionRecord {
				r := validFTTransferRecord()
				r.ValueOrAmount = ""
				return r
			}(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		{
			name: "invalid ValueOrAmount (negative) for FT",
			record: func() *pb.TransactionRecord {
				r := validFTTransferRecord()
				r.ValueOrAmount = "-100"
				return r
			}(),
			wantErr: true, expectedErrType: nexuserrors.ErrValueOutOfRange,
		},
		{
            name: "invalid ValueOrAmount (not a number for FT)",
            record: func() *pb.TransactionRecord {
                r := validFTTransferRecord()
                r.ValueOrAmount = "abc"
                return r
            }(),
            wantErr: true, expectedErrType: nexuserrors.ErrInvalidFormat,
        },
        {
            name: "invalid ValueOrAmount for ASSET_TRANSFER (not '1' or empty)",
             record: func() *pb.TransactionRecord {
                r := validTxRecordAsset()
                r.ValueOrAmount = "2" // Should be "1" or ""
                return r
            }(),
            wantErr: true, expectedErrType: nexuserrors.ErrInvalidFormat,
        },
		// CurrencyOrTokenId Tests
		{
			name: "missing CurrencyOrTokenId when ValueOrAmount is present for FT",
			record: func() *pb.TransactionRecord {
				r := validFTTransferRecord()
				r.CurrencyOrTokenId = ""
				return r
			}(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		// Timestamp Tests
		{
			name: "missing Timestamp",
			record: func() *pb.TransactionRecord { r := validTxRecordAsset(); r.Timestamp = nil; return r }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		// Status Tests
		{
			name: "invalid Status enum",
			record: func() *pb.TransactionRecord { r := validTxRecordAsset(); r.Status = pb.DigiSocialTransactionStatus(99); return r }(),
			wantErr: true, expectedErrType: nexuserrors.ErrUnknownEnumValue,
		},
		// RelatedInteractionId (optional)
		{
			name: "invalid RelatedInteractionId format if present",
			record: func() *pb.TransactionRecord { r := validTxRecordAsset(); r.RelatedInteractionId = "not-a-uuid"; return r }(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidUUID,
		},
		// MemoOrDescription (optional)
		{
			name: "MemoOrDescription too long",
			record: func() *pb.TransactionRecord { r := validTxRecordAsset(); r.MemoOrDescription = string(make([]byte, pb.MaxMemoLengthTransactionRecord + 1)); return r }(),
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong,
		},
		// Details map tests
		{
			name: "too many Details items",
			record: func() *pb.TransactionRecord {
				r := validTxRecordAsset()
				r.Details = make(map[string]string)
				for i := 0; i < 11; i++ { r.Details[uuid.NewString()] = "detail_value"}
				return r
			}(),
			wantErr: true, expectedErrType: nexuserrors.ErrTooManyItems,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.record.Validate()
			checkTransactionRecordError(t, err, tc.expectedErrType, tc.wantErr)
		})
	}
}
```
