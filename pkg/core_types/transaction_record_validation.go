package core_types

import (
	"fmt"
	"strconv"
	"time"

	"empower1/internal/nexuserrors"
	"empower1/internal/validationutils"
	"github.com/google/uuid"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// Constants for TransactionRecord validation
const (
	MaxMemoLengthTransactionRecord              = 256
	MinCurrencyOrTokenIdLengthTransactionRecord = 1
	MaxCurrencyOrTokenIdLengthTransactionRecord = 50
	// EpochStartUnixTR is now validationutils.ProjectEpochStartUnix
)

// Validate checks the TransactionRecord fields for correctness.
func (m *TransactionRecord) Validate() error {
	if m == nil {
		return fmt.Errorf("TransactionRecord is nil: %w", nexuserrors.ErrMissingField)
	}

	// TransactionId
	if m.TransactionId == "" {
		return fmt.Errorf("TransactionId: %w", nexuserrors.ErrMissingField)
	}
	if !validationutils.IsValidUUID(m.TransactionId) {
		return fmt.Errorf("TransactionId '%s' is not a valid UUID: %w", m.TransactionId, nexuserrors.ErrInvalidUUID)
	}

	// FromUserId
	if m.FromUserId == "" {
		return fmt.Errorf("FromUserId: %w", nexuserrors.ErrMissingField)
	}
	if !validationutils.IsValidUUID(m.FromUserId) {
		// Add exceptions for system IDs like "SYSTEM_MINT" if needed
		// if m.FromUserId != "SYSTEM_MINT" { ... }
		return fmt.Errorf("FromUserId '%s' is not a valid UUID: %w", m.FromUserId, nexuserrors.ErrInvalidUUID)
	}

	// ToUserId
	if m.ToUserId == "" {
		return fmt.Errorf("ToUserId: %w", nexuserrors.ErrMissingField)
	}
	if !validationutils.IsValidUUID(m.ToUserId) {
		// Add exceptions for system IDs like "SYSTEM_BURN" if needed
		// if m.ToUserId != "SYSTEM_BURN" { ... }
		return fmt.Errorf("ToUserId '%s' is not a valid UUID: %w", m.ToUserId, nexuserrors.ErrInvalidUUID)
	}

	// AssetId (optional)
	if m.AssetId != "" && !validationutils.IsValidUUID(m.AssetId) {
		return fmt.Errorf("AssetId '%s' (if present) is not a valid UUID: %w", m.AssetId, nexuserrors.ErrInvalidUUID)
	}

	// TransactionType
	if err := validationutils.CheckEnumValue(m.TransactionType, DigiSocialTransactionType_name, "TransactionType", DigiSocialTransactionType_DIGISOCIAL_TRANSACTION_TYPE_UNSPECIFIED, "DigiSocialTransactionType"); err != nil {
		return err
	}

	isFungibleTransfer := m.TransactionType == DigiSocialTransactionType_DIGISOCIAL_TRANSACTION_TYPE_TRANSFER_FT ||
	                      m.TransactionType == DigiSocialTransactionType_DIGISOCIAL_TRANSACTION_TYPE_MINT_FT ||
	                      m.TransactionType == DigiSocialTransactionType_DIGISOCIAL_TRANSACTION_TYPE_BURN_FT ||
	                      m.TransactionType == DigiSocialTransactionType_DIGISOCIAL_TRANSACTION_TYPE_TIP
	isAssetOperation := m.TransactionType == DigiSocialTransactionType_DIGISOCIAL_TRANSACTION_TYPE_TRANSFER_ASSET ||
					   m.TransactionType == DigiSocialTransactionType_DIGISOCIAL_TRANSACTION_TYPE_MINT_ASSET ||
					   m.TransactionType == DigiSocialTransactionType_DIGISOCIAL_TRANSACTION_TYPE_BURN_ASSET

	requiresValueOrAmount := isFungibleTransfer

	if requiresValueOrAmount {
		if m.ValueOrAmount == "" {
			return fmt.Errorf("ValueOrAmount is required for transaction type %v: %w", m.TransactionType, nexuserrors.ErrMissingField)
		}
		valFloat, err := strconv.ParseFloat(m.ValueOrAmount, 64) // Assuming ValueOrAmount is string decimal
		if err != nil {
			return fmt.Errorf("ValueOrAmount '%s' is not a valid number: %w", m.ValueOrAmount, nexuserrors.ErrInvalidFormat)
		}
		if valFloat < 0 { // Allow 0 for specific FT operations like querying balance, but not for transfer/mint/burn
			if valFloat == 0 && (m.TransactionType == DigiSocialTransactionType_DIGISOCIAL_TRANSACTION_TYPE_TRANSFER_FT || m.TransactionType == DigiSocialTransactionType_DIGISOCIAL_TRANSACTION_TYPE_MINT_FT) {
				return fmt.Errorf("ValueOrAmount for FT transfer/mint cannot be zero: %w", nexuserrors.ErrValueOutOfRange)
			} else if valFloat <0 {
				return fmt.Errorf("ValueOrAmount '%s' cannot be negative: %w", m.ValueOrAmount, nexuserrors.ErrValueOutOfRange)
			}
		}


		if m.CurrencyOrTokenId == "" {
			return fmt.Errorf("CurrencyOrTokenId is required when ValueOrAmount is present for fungible transaction: %w", nexuserrors.ErrMissingField)
		}
		if err := validationutils.CheckStringLength(m.CurrencyOrTokenId, "CurrencyOrTokenId", MinCurrencyOrTokenIdLengthTransactionRecord, MaxCurrencyOrTokenIdLengthTransactionRecord); err != nil {
			return err
		}

	} else if isAssetOperation {
		// For asset operations, ValueOrAmount might be "1" or empty (implicit 1)
		if m.ValueOrAmount != "" && m.ValueOrAmount != "1" {
			return fmt.Errorf("ValueOrAmount for asset transaction type should be '1' or empty, got '%s': %w", m.ValueOrAmount, nexuserrors.ErrInvalidFormat)
		}
		if m.AssetId == "" { // AssetId is mandatory for asset operations
			return fmt.Errorf("AssetId is required for asset transaction type %v: %w", m.TransactionType, nexuserrors.ErrMissingField)
		}
		// CurrencyOrTokenId might be redundant if AssetId is present, or could specify a payment currency for a sale.
		// For simple transfers/mints/burns of an asset, CurrencyOrTokenId might be optional or match AssetId.
		// This logic can be refined based on specific protocol rules.
	}


	// Timestamp
	var valTimestamp *validationutils.Timestamp
	if m.Timestamp != nil {
		if m.Timestamp.GetSeconds() == 0 && m.Timestamp.GetNanos() == 0 {
            return fmt.Errorf("Timestamp: timestamp is zero: %w", nexuserrors.ErrInvalidTimestamp)
        }
		valTimestamp = &validationutils.Timestamp{Seconds: m.Timestamp.GetSeconds(), Nanos: m.Timestamp.GetNanos()}
	} else {
		return fmt.Errorf("Timestamp: %w", nexuserrors.ErrMissingField)
	}
	if err := validationutils.CheckTimestamp(valTimestamp, "Timestamp", validationutils.ProjectEpochStartUnix, false, 0); err != nil {
		return err
	}

	// Status
	if err := validationutils.CheckEnumValue(m.Status, DigiSocialTransactionStatus_name, "Status", DigiSocialTransactionStatus_DIGISOCIAL_TRANSACTION_STATUS_UNSPECIFIED, "DigiSocialTransactionStatus"); err != nil {
		return err
	}

	// RelatedInteractionId (optional)
	if m.RelatedInteractionId != "" && !validationutils.IsValidUUID(m.RelatedInteractionId) {
		return fmt.Errorf("RelatedInteractionId '%s' is not a valid UUID: %w", m.RelatedInteractionId, nexuserrors.ErrInvalidUUID)
	}

	// MemoOrDescription (optional)
	if m.MemoOrDescription != "" {
		if err := validationutils.CheckStringLength(m.MemoOrDescription, "MemoOrDescription", 0, MaxMemoLengthTransactionRecord); err != nil {
			return err
		}
	}

	// Details map<string, string>
	if m.Details != nil {
		if len(m.Details) > 10 { // Max 10 detail entries
			return fmt.Errorf("Details map: %w", nexuserrors.ErrTooManyItems)
		}
		for key, value := range m.Details {
			if key == "" {return fmt.Errorf("Details map key: %w", nexuserrors.ErrInvalidMetadataKey)}
			if err:= validationutils.CheckStringLength(key, fmt.Sprintf("Details map key '%s'", key), 1, 50); err != nil {return err}
			if err:= validationutils.CheckStringLength(value, fmt.Sprintf("Details map value for key '%s'", key), 0, 256); err != nil {return err}
		}
	}


	return nil
}
```
