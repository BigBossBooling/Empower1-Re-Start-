package core_types

import (
	"fmt"
	"regexp"
	"time"

	"empower1/internal/nexuserrors"
	"empower1/internal/validationutils"
	"github.com/google/uuid"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb" // For pb.Timestamp type
)

// Constants for UserProfile validation (assuming these are still relevant)
const (
	MinUsernameLengthUserProfile          = 3 // Added suffix for clarity
	MaxUsernameLengthUserProfile          = 30
	MaxDisplayNameLengthUserProfile       = 50
	MaxBioLengthUserProfile               = 160
	MaxProfilePictureURLLengthUserProfile = 2048
	MinReputationScoreUserProfile         = -1000000
	MaxReputationScoreUserProfile         = 1000000
	// ProjectEpochStartUnix is now in validationutils
)

var usernameRegexUserProfile = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// Validate checks the UserProfile fields for correctness using validationutils.
// Assumes m is a pointer to a struct matching protoc-gen-go output for UserProfile.
func (m *UserProfile) Validate() error {
	if m == nil {
		return fmt.Errorf("UserProfile is nil: %w", nexuserrors.ErrMissingField)
	}

	// UserId (was user_id in proto)
	if m.UserId == "" { // Field name now UserId
		return fmt.Errorf("UserId: %w", nexuserrors.ErrMissingField)
	}
	if !validationutils.IsValidUUID(m.UserId) {
		return fmt.Errorf("UserId '%s' is not a valid UUID: %w", m.UserId, nexuserrors.ErrInvalidUUID)
	}

	// Username
	if m.Username == "" {
		return fmt.Errorf("Username: %w", nexuserrors.ErrMissingField)
	}
	if err := validationutils.CheckStringLength(m.Username, "Username", MinUsernameLengthUserProfile, MaxUsernameLengthUserProfile); err != nil {
		return err
	}
	if err := validationutils.CheckAllowedChars(m.Username, "Username", usernameRegexUserProfile); err != nil {
		return err
	}

	// DisplayName (was display_name)
	if m.DisplayName != "" {
		if err := validationutils.CheckStringLength(m.DisplayName, "DisplayName", 0, MaxDisplayNameLengthUserProfile); err != nil {
			return err
		}
	}

	// Bio
	if m.Bio != "" {
		if err := validationutils.CheckStringLength(m.Bio, "Bio", 0, MaxBioLengthUserProfile); err != nil {
			return err
		}
	}

	// ProfilePictureUrl (was profile_picture_url)
	if m.ProfilePictureUrl != "" {
		if err := validationutils.CheckStringLength(m.ProfilePictureUrl, "ProfilePictureUrl", 0, MaxProfilePictureURLLengthUserProfile); err != nil {
			return err
		}
		if !validationutils.IsValidURL(m.ProfilePictureUrl, []string{"http", "https"}) {
			return fmt.Errorf("ProfilePictureUrl '%s' is not a valid HTTP/HTTPS URL: %w", m.ProfilePictureUrl, nexuserrors.ErrInvalidURL)
		}
	}

	// CreatedAt & UpdatedAt (are *timestamppb.Timestamp)
	var valCreatedAt, valUpdatedAt *validationutils.Timestamp
	if m.CreatedAt != nil {
		// Ensure CreatedAt is not a zero value protobuf timestamp if it's required to be meaningful
        if m.CreatedAt.GetSeconds() == 0 && m.CreatedAt.GetNanos() == 0 {
             return fmt.Errorf("CreatedAt: timestamp is zero: %w", nexuserrors.ErrInvalidTimestamp)
        }
		valCreatedAt = &validationutils.Timestamp{Seconds: m.CreatedAt.GetSeconds(), Nanos: m.CreatedAt.GetNanos()}
	} else {
		return fmt.Errorf("CreatedAt: %w", nexuserrors.ErrMissingField) // If CreatedAt is required
	}

	if m.UpdatedAt != nil {
        if m.UpdatedAt.GetSeconds() == 0 && m.UpdatedAt.GetNanos() == 0 {
            return fmt.Errorf("UpdatedAt: timestamp is zero: %w", nexuserrors.ErrInvalidTimestamp)
        }
		valUpdatedAt = &validationutils.Timestamp{Seconds: m.UpdatedAt.GetSeconds(), Nanos: m.UpdatedAt.GetNanos()}
	} else {
		return fmt.Errorf("UpdatedAt: %w", nexuserrors.ErrMissingField) // If UpdatedAt is required
	}

	if err := validationutils.CheckTimestamp(valCreatedAt, "CreatedAt", validationutils.ProjectEpochStartUnix, false, 0); err != nil {
		return err // CheckTimestamp already checks for nil if passed directly, but here we made valCreatedAt from m.CreatedAt
	}
	if err := validationutils.CheckTimestamp(valUpdatedAt, "UpdatedAt", validationutils.ProjectEpochStartUnix, false, 0); err != nil {
		return err
	}
	if err := validationutils.CheckLogicalTimestampOrder(valCreatedAt, valUpdatedAt, "CreatedAt", "UpdatedAt"); err != nil {
		return err
	}

	// ReputationScore (was reputation_score)
	if m.ReputationScore < MinReputationScoreUserProfile || m.ReputationScore > MaxReputationScoreUserProfile {
		return fmt.Errorf("ReputationScore %d is out of range [%d, %d]: %w", m.ReputationScore, MinReputationScoreUserProfile, MaxReputationScoreUserProfile, nexuserrors.ErrValueOutOfRange)
	}

	// Status
	// Assumes UserStatus_name and UserStatus_USER_STATUS_UNSPECIFIED are available
	if err := validationutils.CheckEnumValue(m.Status, UserStatus_name, "Status", UserStatus_USER_STATUS_UNSPECIFIED, "UserStatus"); err != nil {
		return err
	}

	// Interests and ExternalLinks (maps/repeated fields)
	// Example for interests (repeated string)
	if m.Interests != nil {
		if len(m.Interests) > 20 { // Max 20 interests
			return fmt.Errorf("Interests: %w", nexuserrors.ErrTooManyItems)
		}
		for i, interest := range m.Interests {
			if interest == "" {
				return fmt.Errorf("Interests[%d]: %w", i, nexuserrors.ErrEmptyListItem)
			}
			if err := validationutils.CheckStringLength(interest, fmt.Sprintf("Interests[%d]", i), 1, 50); err != nil { // Max 50 chars per interest
				return err
			}
		}
	}

	// Example for external_links (map<string, string>)
	if m.ExternalLinks != nil {
		if len(m.ExternalLinks) > 5 { // Max 5 external links
			return fmt.Errorf("ExternalLinks: %w", nexuserrors.ErrTooManyItems)
		}
		for key, value := range m.ExternalLinks {
			if key == "" {return fmt.Errorf("ExternalLinks key: %w", nexuserrors.ErrInvalidMetadataKey)} // Re-use for map keys
			if err:= validationutils.CheckStringLength(key, fmt.Sprintf("ExternalLinks key '%s'", key), 1, 30); err != nil {return err}

			if value == "" {return fmt.Errorf("ExternalLinks value for key '%s': %w", key, nexuserrors.ErrMissingField)} // Assuming values must be non-empty
			if err:= validationutils.CheckStringLength(value, fmt.Sprintf("ExternalLinks value for key '%s'", key), 1, 2048); err != nil {return err}
			if !validationutils.IsValidURL(value, []string{"http", "https"}) { // Assuming links are URLs
				return fmt.Errorf("ExternalLinks value for key '%s' is not a valid URL: %w", key, nexuserrors.ErrInvalidURL)
			}
		}
	}


	return nil
}
```
