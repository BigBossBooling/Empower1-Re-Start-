package core_types

import (
	"fmt"
	"time"
	"strings" // For Rules validation if needed

	"empower1/internal/nexuserrors"
	"empower1/internal/validationutils"
	"github.com/google/uuid"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// Constants for CommunityGroup validation
const (
	MinGroupNameLengthCommunityGroup        = 3
	MaxGroupNameLengthCommunityGroup        = 100
	MaxGroupDescriptionLengthCommunityGroup = 500
	MaxGroupAvatarURLLengthCommunityGroup   = 2048
	MaxGroupBannerURLLengthCommunityGroup  = 2048
	MaxGroupRulesCommunityGroup             = 10
	MaxRuleLengthCommunityGroup             = 500
	MaxMetadataItemsCommunityGroup          = 20
	MaxMetadataKeyLengthCommunityGroup      = 50
	MaxMetadataValueLengthCommunityGroup    = 256
	// EpochStartUnixCG is now validationutils.ProjectEpochStartUnix
)

// Validate checks the CommunityGroup fields for correctness.
func (m *CommunityGroup) Validate() error {
	if m == nil {
		return fmt.Errorf("CommunityGroup is nil: %w", nexuserrors.ErrMissingField)
	}

	// GroupId
	if m.GroupId == "" {
		return fmt.Errorf("GroupId: %w", nexuserrors.ErrMissingField)
	}
	if !validationutils.IsValidUUID(m.GroupId) {
		return fmt.Errorf("GroupId '%s' is not a valid UUID: %w", m.GroupId, nexuserrors.ErrInvalidUUID)
	}

	// Name
	if m.Name == "" {
		return fmt.Errorf("Name: %w", nexuserrors.ErrMissingField)
	}
	if err := validationutils.CheckStringLength(m.Name, "Name", MinGroupNameLengthCommunityGroup, MaxGroupNameLengthCommunityGroup); err != nil {
		return err
	}

	// Description (optional)
	if m.Description != "" {
		if err := validationutils.CheckStringLength(m.Description, "Description", 0, MaxGroupDescriptionLengthCommunityGroup); err != nil {
			return err
		}
	}

	// CreatorId
	if m.CreatorId == "" {
		return fmt.Errorf("CreatorId: %w", nexuserrors.ErrMissingField)
	}
	if !validationutils.IsValidUUID(m.CreatorId) {
		return fmt.Errorf("CreatorId '%s' is not a valid UUID: %w", m.CreatorId, nexuserrors.ErrInvalidUUID)
	}

	// PrivacySetting
	if err := validationutils.CheckEnumValue(m.PrivacySetting, GroupPrivacy_name, "PrivacySetting", GroupPrivacy_GROUP_PRIVACY_UNSPECIFIED, "GroupPrivacy"); err != nil {
		return err
	}

	// CreatedAt & UpdatedAt
	var valCreatedAt, valUpdatedAt *validationutils.Timestamp
	if m.CreatedAt != nil {
		if m.CreatedAt.GetSeconds() == 0 && m.CreatedAt.GetNanos() == 0 {
             return fmt.Errorf("CreatedAt: timestamp is zero: %w", nexuserrors.ErrInvalidTimestamp)
        }
		valCreatedAt = &validationutils.Timestamp{Seconds: m.CreatedAt.GetSeconds(), Nanos: m.CreatedAt.GetNanos()}
	} else {
		 return fmt.Errorf("CreatedAt: %w", nexuserrors.ErrMissingField)
	}

	if m.UpdatedAt != nil {
		if m.UpdatedAt.GetSeconds() == 0 && m.UpdatedAt.GetNanos() == 0 {
            return fmt.Errorf("UpdatedAt: timestamp is zero: %w", nexuserrors.ErrInvalidTimestamp)
        }
		valUpdatedAt = &validationutils.Timestamp{Seconds: m.UpdatedAt.GetSeconds(), Nanos: m.UpdatedAt.GetNanos()}
	} else {
		return fmt.Errorf("UpdatedAt: %w", nexuserrors.ErrMissingField)
	}

	if err := validationutils.CheckTimestamp(valCreatedAt, "CreatedAt", validationutils.ProjectEpochStartUnix, false, 0); err != nil {
		return err
	}
	if err := validationutils.CheckTimestamp(valUpdatedAt, "UpdatedAt", validationutils.ProjectEpochStartUnix, false, 0); err != nil {
		return err
	}
	if err := validationutils.CheckLogicalTimestampOrder(valCreatedAt, valUpdatedAt, "CreatedAt", "UpdatedAt"); err != nil {
		return err
	}

	// MemberCount (likely server-managed, but validate if provided on creation/update)
	if m.MemberCount < 0 {
		return fmt.Errorf("MemberCount %d cannot be negative: %w", m.MemberCount, nexuserrors.ErrValueOutOfRange)
	}

	// GroupAvatarUrl (optional)
	if m.GroupAvatarUrl != "" {
		if err := validationutils.CheckStringLength(m.GroupAvatarUrl, "GroupAvatarUrl", 0, MaxGroupAvatarURLLengthCommunityGroup); err != nil {
			return err
		}
		if !validationutils.IsValidURL(m.GroupAvatarUrl, []string{"http", "https"}) {
			return fmt.Errorf("GroupAvatarUrl '%s' is not a valid HTTP/HTTPS URL: %w", m.GroupAvatarUrl, nexuserrors.ErrInvalidURL)
		}
	}

	// GroupBannerUrl (optional)
	if m.GroupBannerUrl != "" {
		if err := validationutils.CheckStringLength(m.GroupBannerUrl, "GroupBannerUrl", 0, MaxGroupBannerURLLengthCommunityGroup); err != nil {
			return err
		}
		if !validationutils.IsValidURL(m.GroupBannerUrl, []string{"http", "https"}) {
			return fmt.Errorf("GroupBannerUrl '%s' is not a valid HTTP/HTTPS URL: %w", m.GroupBannerUrl, nexuserrors.ErrInvalidURL)
		}
	}

	// Rules (repeated string)
	if m.Rules != nil {
		if len(m.Rules) > MaxGroupRulesCommunityGroup {
			return fmt.Errorf("Rules: %w", nexuserrors.ErrTooManyItems)
		}
		for i, rule := range m.Rules {
			fieldName := fmt.Sprintf("Rules[%d]", i)
			if rule == "" {
				return fmt.Errorf("%s: %w", fieldName, nexuserrors.ErrEmptyListItem)
			}
			if err := validationutils.CheckStringLength(rule, fieldName, 1, MaxRuleLengthCommunityGroup); err != nil {
				return err
			}
		}
	}

	// Metadata (map<string, string>)
	if m.Metadata != nil {
		if len(m.Metadata) > MaxMetadataItemsCommunityGroup {
			return fmt.Errorf("Metadata: %w", nexuserrors.ErrTooManyItems)
		}
		for key, value := range m.Metadata {
			if key == "" {return fmt.Errorf("Metadata key: %w", nexuserrors.ErrInvalidMetadataKey)}
			if err:= validationutils.CheckStringLength(key, fmt.Sprintf("Metadata key '%s'", key), 1, MaxMetadataKeyLengthCommunityGroup); err != nil {return err}
			if err:= validationutils.CheckStringLength(value, fmt.Sprintf("Metadata value for key '%s'", key), 0, MaxMetadataValueLengthCommunityGroup); err != nil {return err}
		}
	}

	return nil
}
```
