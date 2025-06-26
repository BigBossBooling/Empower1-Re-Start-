package core_types_test

import (
	"errors"
	// "regexp" // No longer needed here if regex is in validation file
	"testing"
	"time"

	"empower1/internal/nexuserrors"
	pb "empower1/pkg/core_types" // pb alias for generated types
	"github.com/google/uuid"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// checkUserProfileError helper (already defined and seems fine)
func checkUserProfileError(t *testing.T, gotErr, wantErrType error, wantErr bool) {
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


func TestUserProfile_Validate(t *testing.T) {
	validUUID := uuid.New().String()
	validUsername := "testuser123"
	now := time.Now().UTC()
	// epochTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC) // Now uses validationutils.ProjectEpochStartUnix

	// Actual enum values from generated Go code (adjust if names differ)
	// Assuming these are generated in the pb package (pkg/core_types)
	// These constants should ideally be defined using the generated enum type for type safety,
	// e.g., const UserStatus_Active = pb.UserStatus_USER_STATUS_ACTIVE
	// For test simplicity, if pb.UserStatus_USER_STATUS_ACTIVE is an int32, direct use is okay.
	// We assume pb.UserStatus is the generated enum type.

	validProfile := func() *pb.UserProfile {
		return &pb.UserProfile{
			UserId:            validUUID,
			Username:          validUsername,
			DisplayName:       "Test User",
			Bio:               "This is a test bio.",
			ProfilePictureUrl: "http://example.com/pic.jpg",
			CreatedAt:         timestamppb.New(now.Add(-time.Hour)),
			UpdatedAt:         timestamppb.New(now),
			ReputationScore:   100,
			Status:            pb.UserStatus_USER_STATUS_ACTIVE, // Use generated enum
			Interests:         []string{"golang", "crypto"},
			ExternalLinks:     map[string]string{"github": "http://github.com/testuser"},
		}
	}

	tests := []struct {
		name            string
		profile         *pb.UserProfile
		wantErr         bool
		expectedErrType error
	}{
		{
			name: "valid user profile",
			profile: validProfile(),
			wantErr: false,
		},
		// UserId Tests
		{
			name: "missing UserId",
			profile: func() *pb.UserProfile { p := validProfile(); p.UserId = ""; return p }(),
			wantErr:         true,
			expectedErrType: nexuserrors.ErrMissingField,
		},
		{
			name: "invalid UserId format",
			profile: func() *pb.UserProfile { p := validProfile(); p.UserId = "not-a-uuid"; return p }(),
			wantErr:         true,
			expectedErrType: nexuserrors.ErrInvalidUUID,
		},
		// Username Tests
		{
			name: "missing Username",
			profile: func() *pb.UserProfile { p := validProfile(); p.Username = ""; return p }(),
			wantErr:         true,
			expectedErrType: nexuserrors.ErrMissingField,
		},
		{
			name: "Username too short",
			profile: func() *pb.UserProfile { p := validProfile(); p.Username = "a"; return p }(),
			wantErr:         true,
			expectedErrType: nexuserrors.ErrStringTooShort,
		},
		{
			name: "Username too long",
			profile: func() *pb.UserProfile { p := validProfile(); p.Username = string(make([]byte, pb.MaxUsernameLengthUserProfile + 1)); return p }(),
			wantErr:         true,
			expectedErrType: nexuserrors.ErrStringTooLong,
		},
		{
			name: "Username invalid characters",
			profile: func() *pb.UserProfile { p := validProfile(); p.Username = "test user!"; return p }(),
			wantErr:         true,
			expectedErrType: nexuserrors.ErrInvalidCharacters,
		},
		// DisplayName Tests
		{
			name: "DisplayName too long",
			profile: func() *pb.UserProfile { p := validProfile(); p.DisplayName = string(make([]byte, pb.MaxDisplayNameLengthUserProfile + 1)); return p }(),
			wantErr:         true,
			expectedErrType: nexuserrors.ErrStringTooLong,
		},
		// Bio Tests
		{
			name: "Bio too long",
			profile: func() *pb.UserProfile { p := validProfile(); p.Bio = string(make([]byte, pb.MaxBioLengthUserProfile + 1)); return p }(),
			wantErr:         true,
			expectedErrType: nexuserrors.ErrStringTooLong,
		},
		// ProfilePictureUrl Tests
		{
			name: "ProfilePictureUrl invalid format",
			profile: func() *pb.UserProfile { p := validProfile(); p.ProfilePictureUrl = "not a valid url"; return p }(),
			wantErr:         true,
			expectedErrType: nexuserrors.ErrInvalidURL,
		},
		{
			name: "ProfilePictureUrl too long",
			profile: func() *pb.UserProfile { p := validProfile(); p.ProfilePictureUrl = "http://" + string(make([]byte, pb.MaxProfilePictureURLLengthUserProfile)) + ".com"; return p }(), // exactly 2048 + http:// .com
			wantErr:         true,
			expectedErrType: nexuserrors.ErrStringTooLong,
		},
		// Timestamp Tests
		{
			name: "missing CreatedAt",
			profile: func() *pb.UserProfile { p := validProfile(); p.CreatedAt = nil; return p }(),
			wantErr:         true,
			expectedErrType: nexuserrors.ErrMissingField,
		},
		{
            name: "CreatedAt is zero protobuf timestamp",
            profile: func() *pb.UserProfile { p := validProfile(); p.CreatedAt = &timestamppb.Timestamp{Seconds:0, Nanos:0}; return p }(),
            wantErr:         true,
            expectedErrType: nexuserrors.ErrInvalidTimestamp,
        },
		{
			name: "CreatedAt before epoch",
			profile: func() *pb.UserProfile { p := validProfile(); p.CreatedAt = timestamppb.New(time.Unix(pb.ProjectEpochStartUnix-1, 0)); return p }(),
			wantErr:         true,
			expectedErrType: nexuserrors.ErrInvalidTimestamp,
		},
		{
			name: "missing UpdatedAt",
			profile: func() *pb.UserProfile { p := validProfile(); p.UpdatedAt = nil; return p }(),
			wantErr:         true,
			expectedErrType: nexuserrors.ErrMissingField,
		},
		{
            name: "UpdatedAt is zero protobuf timestamp",
            profile: func() *pb.UserProfile { p := validProfile(); p.UpdatedAt = &timestamppb.Timestamp{Seconds:0, Nanos:0}; return p }(),
            wantErr:         true,
            expectedErrType: nexuserrors.ErrInvalidTimestamp,
        },
		{
			name: "UpdatedAt before CreatedAt",
			profile: func() *pb.UserProfile {
				p := validProfile()
				p.CreatedAt = timestamppb.New(now)
				p.UpdatedAt = timestamppb.New(now.Add(-time.Hour))
				return p
			}(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidTimestamp,
		},
		// ReputationScore Tests
		{
			name: "ReputationScore below min",
			profile: func() *pb.UserProfile { p := validProfile(); p.ReputationScore = pb.MinReputationScoreUserProfile - 1; return p }(),
			wantErr:         true,
			expectedErrType: nexuserrors.ErrValueOutOfRange,
		},
		{
			name: "ReputationScore above max",
			profile: func() *pb.UserProfile { p := validProfile(); p.ReputationScore = pb.MaxReputationScoreUserProfile + 1; return p }(),
			wantErr:         true,
			expectedErrType: nexuserrors.ErrValueOutOfRange,
		},
		// Status Tests
		{
			name: "invalid status enum value",
			profile: func() *pb.UserProfile { p := validProfile(); p.Status = pb.UserStatus(99); return p }(),
			wantErr:         true,
			expectedErrType: nexuserrors.ErrUnknownEnumValue,
		},
		{
			name: "status unspecified",
			profile: func() *pb.UserProfile { p := validProfile(); p.Status = pb.UserStatus_USER_STATUS_UNSPECIFIED; return p }(),
			wantErr:         true,
			expectedErrType: nexuserrors.ErrUnknownEnumValue,
		},
		// Interests tests
		{
			name: "too many interests",
			profile: func() *pb.UserProfile { p := validProfile(); p.Interests = make([]string, 21); return p}(), // Assuming max 20
			wantErr: true, expectedErrType: nexuserrors.ErrTooManyItems,
		},
		{
			name: "empty interest string",
			profile: func() *pb.UserProfile { p := validProfile(); p.Interests = []string{"valid", ""}; return p}(),
			wantErr: true, expectedErrType: nexuserrors.ErrEmptyListItem,
		},
		{
			name: "interest string too long",
			profile: func() *pb.UserProfile { p := validProfile(); p.Interests = []string{string(make([]byte, 51))}; return p}(), // Assuming max 50
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong,
		},
		// ExternalLinks tests
		{
			name: "too many external links",
			profile: func() *pb.UserProfile {
				p := validProfile()
				p.ExternalLinks = make(map[string]string)
				for i := 0; i < 6; i++ { p.ExternalLinks[uuid.NewString()] = "http://example.com"} // Assuming max 5
				return p
			}(),
			wantErr: true, expectedErrType: nexuserrors.ErrTooManyItems,
		},
		{
			name: "external links key empty",
			profile: func() *pb.UserProfile { p := validProfile(); p.ExternalLinks = map[string]string{"": "http://example.com"}; return p}(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidMetadataKey,
		},
		{
			name: "external links key too long",
			profile: func() *pb.UserProfile { p := validProfile(); p.ExternalLinks = map[string]string{string(make([]byte, 31)): "http://example.com"}; return p}(), // Assuming max 30 for key
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong,
		},
		{
			name: "external links value empty",
			profile: func() *pb.UserProfile { p := validProfile(); p.ExternalLinks = map[string]string{"website": ""}; return p}(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		{
			name: "external links value too long",
			profile: func() *pb.UserProfile { p := validProfile(); p.ExternalLinks = map[string]string{"website": "http://" + string(make([]byte, 2040)) + ".com"}; return p}(), // Assuming max 2048 total
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong,
		},
		{
			name: "external links value invalid URL",
			profile: func() *pb.UserProfile { p := validProfile(); p.ExternalLinks = map[string]string{"website": "not a url"}; return p}(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidURL,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.profile.Validate()
			checkUserProfileError(t, err, tc.expectedErrType, tc.wantErr)
		})
	}
}
```
