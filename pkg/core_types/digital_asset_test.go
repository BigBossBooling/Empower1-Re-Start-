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

// checkDigitalAssetError helper
func checkDigitalAssetError(t *testing.T, gotErr, wantErrType error, wantErr bool) {
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

func TestDigitalAsset_Validate(t *testing.T) {
	validAssetUUID := uuid.New().String()
	validOwnerUUID := uuid.New().String()
	now := time.Now().UTC()

	validAsset := func() *pb.DigitalAsset {
		return &pb.DigitalAsset{
			AssetId:               validAssetUUID,
			OwnerId:               validOwnerUUID,
			AssetType:             pb.AssetType_ASSET_TYPE_NFT_IMAGE,
			Name:                  "Valid Asset Name",
			Description:           "A valid description of the asset.",
			AssetUrlOrIdentifier:  "ipfs://QmValidHash",
			Metadata:              map[string]string{"trait": "rare", "color": "blue"},
			CreatedAt:             timestamppb.New(now.Add(-time.Hour)),
			Supply:                1,
			IsTransferable:        true,
			CollectionId:          uuid.NewString(),
		}
	}

	tests := []struct {
		name            string
		asset           *pb.DigitalAsset
		wantErr         bool
		expectedErrType error
	}{
		{name: "valid digital asset (NFT)", asset: validAsset(), wantErr: false},
		// AssetId Tests
		{
			name: "missing AssetId",
			asset: func() *pb.DigitalAsset { da := validAsset(); da.AssetId = ""; return da }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		// OwnerId Tests
		{
			name: "missing OwnerId",
			asset: func() *pb.DigitalAsset { da := validAsset(); da.OwnerId = ""; return da }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		// AssetType Tests
		{
			name: "invalid AssetType enum",
			asset: func() *pb.DigitalAsset { da := validAsset(); da.AssetType = pb.AssetType(99); return da }(),
			wantErr: true, expectedErrType: nexuserrors.ErrUnknownEnumValue,
		},
		// Name Tests
		{
			name: "missing Name",
			asset: func() *pb.DigitalAsset { da := validAsset(); da.Name = ""; return da }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		{
			name: "Name too long",
			asset: func() *pb.DigitalAsset { da := validAsset(); da.Name = string(make([]byte, pb.MaxAssetNameLengthDA + 1)); return da }(),
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong,
		},
		// Description Tests
		{
			name: "Description too long",
			asset: func() *pb.DigitalAsset { da := validAsset(); da.Description = string(make([]byte, pb.MaxAssetDescriptionLengthDA + 1)); return da }(),
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong,
		},
		// AssetUrlOrIdentifier Tests
		{
			name: "missing AssetUrlOrIdentifier",
			asset: func() *pb.DigitalAsset { da := validAsset(); da.AssetUrlOrIdentifier = ""; return da }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		{
			name: "AssetUrlOrIdentifier too long",
			asset: func() *pb.DigitalAsset { da := validAsset(); da.AssetUrlOrIdentifier = string(make([]byte, pb.MaxAssetURLOrIdentifierLengthDA + 1)); return da }(),
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong,
		},
		{
            name: "AssetUrlOrIdentifier invalid URL format (if URL)",
            asset: func() *pb.DigitalAsset {
                da := validAsset()
                da.AssetUrlOrIdentifier = "http//not_a_url_at_all"
                return da
            }(),
            wantErr: true, expectedErrType: nexuserrors.ErrInvalidURL,
        },
		// Metadata Tests
		{
			name: "too many metadata items",
			asset: func() *pb.DigitalAsset {
				da := validAsset()
				da.Metadata = make(map[string]string)
				for i := 0; i < pb.MaxAssetMetadataItemsDA + 1; i++ {
					da.Metadata[uuid.New().String()] = "value"
				}
				return da
			}(),
			wantErr: true, expectedErrType: nexuserrors.ErrTooManyItems,
		},
		{
			name: "metadata key too long",
			asset: func() *pb.DigitalAsset {
				da := validAsset()
				da.Metadata = map[string]string{string(make([]byte, pb.MaxAssetMetadataKeyLengthDA + 1)): "value"}
				return da
			}(),
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong, // CheckStringLength used for key
		},
		{
			name: "metadata value too long",
			asset: func() *pb.DigitalAsset {
				da := validAsset()
				da.Metadata = map[string]string{"key": string(make([]byte, pb.MaxAssetMetadataValueLengthDA + 1))}
				return da
			}(),
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong,
		},
		// CreatedAt Tests
		{
			name: "missing CreatedAt",
			asset: func() *pb.DigitalAsset { da := validAsset(); da.CreatedAt = nil; return da }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		// Supply Tests
		{
			name: "invalid Supply (negative)",
			asset: func() *pb.DigitalAsset { da := validAsset(); da.Supply = -2; return da }(),
			wantErr: true, expectedErrType: nexuserrors.ErrValueOutOfRange,
		},
		// CollectionId Tests
		{
			name: "invalid CollectionId format if present",
			asset: func() *pb.DigitalAsset { da := validAsset(); da.CollectionId = "not-a-uuid"; return da}(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidUUID,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.asset.Validate()
			checkDigitalAssetError(t, err, tc.expectedErrType, tc.wantErr)
		})
	}
}
```
