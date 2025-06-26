package core_types

import (
	"fmt"
	"time"

	"empower1/internal/nexuserrors"
	"empower1/internal/validationutils"
	"github.com/google/uuid"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// Constants for InteractionLog validation
const (
	MaxCommentTextLengthInteractionLog = 2000
	// EpochStartUnixIL is now validationutils.ProjectEpochStartUnix
)

// Validate checks the InteractionLog fields for correctness.
func (m *InteractionLog) Validate() error {
	if m == nil {
		return fmt.Errorf("InteractionLog is nil: %w", nexuserrors.ErrMissingField)
	}

	// InteractionId
	if m.InteractionId == "" {
		return fmt.Errorf("InteractionId: %w", nexuserrors.ErrMissingField)
	}
	if !validationutils.IsValidUUID(m.InteractionId) {
		return fmt.Errorf("InteractionId '%s' is not a valid UUID: %w", m.InteractionId, nexuserrors.ErrInvalidUUID)
	}

	// UserId
	if m.UserId == "" {
		return fmt.Errorf("UserId: %w", nexuserrors.ErrMissingField)
	}
	if !validationutils.IsValidUUID(m.UserId) {
		return fmt.Errorf("UserId '%s' is not a valid UUID: %w", m.UserId, nexuserrors.ErrInvalidUUID)
	}

	// TargetContentId
	if m.TargetContentId == "" {
		return fmt.Errorf("TargetContentId: %w", nexuserrors.ErrMissingField)
	}
	if !validationutils.IsValidUUID(m.TargetContentId) {
		return fmt.Errorf("TargetContentId '%s' is not a valid UUID: %w", m.TargetContentId, nexuserrors.ErrInvalidUUID)
	}

	// TargetContentType
	if err := validationutils.CheckEnumValue(m.TargetContentType, InteractableContentType_name, "TargetContentType", InteractableContentType_INTERACTABLE_CONTENT_TYPE_UNSPECIFIED, "InteractableContentType"); err != nil {
		return err
	}

	// InteractionType
	if err := validationutils.CheckEnumValue(m.InteractionType, InteractionType_name, "InteractionType", InteractionType_INTERACTION_TYPE_UNSPECIFIED, "InteractionType"); err != nil {
		return err
	}

	// CommentText (conditional)
	isCommentExpected := m.InteractionType == InteractionType_INTERACTION_TYPE_COMMENT

	if isCommentExpected && m.CommentText == "" {
		return fmt.Errorf("CommentText is required for COMMENT interaction type: %w", nexuserrors.ErrMissingField)
	}
	if m.CommentText != "" {
		if err := validationutils.CheckStringLength(m.CommentText, "CommentText", 0, MaxCommentTextLengthInteractionLog); err != nil {
			return err
		}
		if !isCommentExpected {
			return fmt.Errorf("CommentText is unexpected for non-COMMENT interaction type %v: %w", m.InteractionType, nexuserrors.ErrUnexpectedField)
		}
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

	// ParentInteractionId (optional)
	if m.ParentInteractionId != "" {
		if !validationutils.IsValidUUID(m.ParentInteractionId) {
			return fmt.Errorf("ParentInteractionId '%s' is not a valid UUID: %w", m.ParentInteractionId, nexuserrors.ErrInvalidUUID)
		}
		if m.ParentInteractionId == m.InteractionId {
			return fmt.Errorf("ParentInteractionId cannot be the same as InteractionId: %w", nexuserrors.ErrInvalidReference)
		}
	}

	// AdditionalData map<string, string> - example validation
	if m.AdditionalData != nil {
		if len(m.AdditionalData) > 10 { // Max 10 additional data entries
			return fmt.Errorf("AdditionalData: %w", nexuserrors.ErrTooManyItems)
		}
		for key, value := range m.AdditionalData {
			if key == "" {return fmt.Errorf("AdditionalData key: %w", nexuserrors.ErrInvalidMetadataKey)}
			if err:= validationutils.CheckStringLength(key, fmt.Sprintf("AdditionalData key '%s'", key), 1, 50); err != nil {return err} // Max key length 50
			if err:= validationutils.CheckStringLength(value, fmt.Sprintf("AdditionalData value for key '%s'", key), 0, 256); err != nil {return err} // Max value length 256
		}
	}


	return nil
}
```
