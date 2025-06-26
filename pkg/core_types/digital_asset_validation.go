package core_types

import (
	"fmt"
	"strings"
	"time"

	"empower1/internal/nexuserrors"
	"empower1/internal/validationutils"
	"github.com/google/uuid"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// Constants for DigitalAsset validation
const (
	MinAssetNameLengthDigitalAsset            = 1
	MaxAssetNameLengthDigitalAsset            = 100
	MaxAssetDescriptionLengthDigitalAsset     = 1000
	MaxAssetURLOrIdentifierLengthDigitalAsset = 2048
	MaxAssetMetadataItemsDigitalAsset         = 50
	MaxAssetMetadataKeyLengthDigitalAsset     = 64
	MaxAssetMetadataValueLengthDigitalAsset   = 256
	// EpochStartUnixDA is now validationutils.ProjectEpochStartUnix
)

// Validate checks the DigitalAsset fields for correctness.
func (m *DigitalAsset) Validate() error {
	if m == nil {
		return fmt.Errorf("DigitalAsset is nil: %w", nexuserrors.ErrMissingField)
	}

	// AssetId
	if m.AssetId == "" {
		return fmt.Errorf("AssetId: %w", nexuserrors.ErrMissingField)
	}
	if !validationutils.IsValidUUID(m.AssetId) {
		return fmt.Errorf("AssetId '%s' is not a valid UUID: %w", m.AssetId, nexuserrors.ErrInvalidUUID)
	}

	// OwnerId
	if m.OwnerId == "" {
		return fmt.Errorf("OwnerId: %w", nexuserrors.ErrMissingField)
	}
	if !validationutils.IsValidUUID(m.OwnerId) {
		return fmt.Errorf("OwnerId '%s' is not a valid UUID: %w", m.OwnerId, nexuserrors.ErrInvalidUUID)
	}

	// AssetType
	if err := validationutils.CheckEnumValue(m.AssetType, AssetType_name, "AssetType", AssetType_ASSET_TYPE_UNSPECIFIED, "AssetType"); err != nil {
		return err
	}

	// Name
	if m.Name == "" {
		return fmt.Errorf("Name: %w", nexuserrors.ErrMissingField)
	}
	if err := validationutils.CheckStringLength(m.Name, "Name", MinAssetNameLengthDigitalAsset, MaxAssetNameLengthDigitalAsset); err != nil {
		return err
	}

	// Description (optional)
	if m.Description != "" {
		if err := validationutils.CheckStringLength(m.Description, "Description", 0, MaxAssetDescriptionLengthDigitalAsset); err != nil {
			return err
		}
	}

	// AssetUrlOrIdentifier
	if m.AssetUrlOrIdentifier == "" {
		return fmt.Errorf("AssetUrlOrIdentifier: %w", nexuserrors.ErrMissingField)
	}
	if err := validationutils.CheckStringLength(m.AssetUrlOrIdentifier, "AssetUrlOrIdentifier", 0, MaxAssetURLOrIdentifierLengthDigitalAsset); err != nil {
		return err
	}
	if strings.Contains(m.AssetUrlOrIdentifier, "://") {
		if !validationutils.IsValidURL(m.AssetUrlOrIdentifier, []string{"http", "https", "ipfs", "ar"}) {
			return fmt.Errorf("AssetUrlOrIdentifier '%s' (if URL) is not valid: %w", m.AssetUrlOrIdentifier, nexuserrors.ErrInvalidURL)
		}
	}

	// Metadata (optional, map<string, string>)
	if m.Metadata != nil {
		if len(m.Metadata) > MaxAssetMetadataItemsDigitalAsset {
			return fmt.Errorf("Metadata: %w", nexuserrors.ErrTooManyItems)
		}
		for k, v := range m.Metadata {
			fieldNameKey := fmt.Sprintf("Metadata key '%s'", k)
			if k == "" {
				return fmt.Errorf("Metadata key cannot be empty: %w", nexuserrors.ErrInvalidMetadataKey)
			}
			if err := validationutils.CheckStringLength(k, fieldNameKey, 1, MaxAssetMetadataKeyLengthDigitalAsset); err != nil {
				return err
			}
			fieldNameValue := fmt.Sprintf("Metadata value for key '%s'", k)
			if err := validationutils.CheckStringLength(v, fieldNameValue, 0, MaxAssetMetadataValueLengthDigitalAsset); err != nil {
				return err
			}
		}
	}

	// CreatedAt
	var valCreatedAt *validationutils.Timestamp
	if m.CreatedAt != nil {
		if m.CreatedAt.GetSeconds() == 0 && m.CreatedAt.GetNanos() == 0 {
            return fmt.Errorf("CreatedAt: timestamp is zero: %w", nexuserrors.ErrInvalidTimestamp)
        }
		valCreatedAt = &validationutils.Timestamp{Seconds: m.CreatedAt.GetSeconds(), Nanos: m.CreatedAt.GetNanos()}
	} else {
		return fmt.Errorf("CreatedAt: %w", nexuserrors.ErrMissingField)
	}
	if err := validationutils.CheckTimestamp(valCreatedAt, "CreatedAt", validationutils.ProjectEpochStartUnix, false, 0); err != nil {
		return err
	}

	// Supply (optional)
	if m.Supply < 0 {
		return fmt.Errorf("Supply %d cannot be negative: %w", m.Supply, nexuserrors.ErrValueOutOfRange)
	}

	// CollectionId (optional)
	if m.CollectionId != "" && !validationutils.IsValidUUID(m.CollectionId) {
		return fmt.Errorf("CollectionId '%s' (if present) is not a valid UUID: %w", m.CollectionId, nexuserrors.ErrInvalidUUID)
	}

	// IsTransferable (bool) - no specific validation needed.

	return nil
}
```
