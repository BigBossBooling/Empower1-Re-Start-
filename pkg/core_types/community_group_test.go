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

// checkCommunityGroupError helper
func checkCommunityGroupError(t *testing.T, gotErr, wantErrType error, wantErr bool) {
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

func TestCommunityGroup_Validate(t *testing.T) {
	validGroupUUID := uuid.New().String()
	validCreatorUUID := uuid.New().String()
	now := time.Now().UTC()

	// Use generated enum values

	validGroup := func() *pb.CommunityGroup {
		return &pb.CommunityGroup{
			GroupId:         validGroupUUID,
			Name:            "Valid Group Name",
			Description:     "This is a valid group description.",
			CreatorId:       validCreatorUUID,
			PrivacySetting:  pb.GroupPrivacy_GROUP_PRIVACY_PUBLIC,
			CreatedAt:       timestamppb.New(now.Add(-time.Hour)),
			UpdatedAt:       timestamppb.New(now),
			MemberCount:     10,
			GroupAvatarUrl: "http://example.com/group_avatar.png",
			GroupBannerUrl: "http://example.com/group_banner.png",
			Rules:           []string{"Rule 1", "Be kind"},
			Metadata:        map[string]string{"category": "gaming"},
		}
	}

	tests := []struct {
		name            string
		group           *pb.CommunityGroup
		wantErr         bool
		expectedErrType error
	}{
		{name: "valid community group", group: validGroup(), wantErr: false},
		// GroupId Tests
		{
			name: "missing GroupId",
			group: func() *pb.CommunityGroup { g := validGroup(); g.GroupId = ""; return g }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		{
			name: "invalid GroupId format",
			group: func() *pb.CommunityGroup { g := validGroup(); g.GroupId = "not-a-uuid"; return g }(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidUUID,
		},
		// Name Tests
		{
			name: "missing Name",
			group: func() *pb.CommunityGroup { g := validGroup(); g.Name = ""; return g }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		{
			name: "Name too short",
			group: func() *pb.CommunityGroup { g := validGroup(); g.Name = "a"; return g }(),
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooShort,
		},
		{
			name: "Name too long",
			group: func() *pb.CommunityGroup { g := validGroup(); g.Name = string(make([]byte, pb.MaxGroupNameLengthCommunityGroup + 1)); return g }(),
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong,
		},
		// Description Tests
		{
			name: "Description too long",
			group: func() *pb.CommunityGroup { g := validGroup(); g.Description = string(make([]byte, pb.MaxGroupDescriptionLengthCommunityGroup + 1)); return g }(),
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong,
		},
		// CreatorId Tests
		{
			name: "missing CreatorId",
			group: func() *pb.CommunityGroup { g := validGroup(); g.CreatorId = ""; return g }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		// PrivacySetting Tests
		{
			name: "invalid PrivacySetting enum",
			group: func() *pb.CommunityGroup { g := validGroup(); g.PrivacySetting = pb.GroupPrivacy(99); return g }(),
			wantErr: true, expectedErrType: nexuserrors.ErrUnknownEnumValue,
		},
		// Timestamp Tests
		{
			name: "UpdatedAt before CreatedAt",
			group: func() *pb.CommunityGroup {
				g := validGroup()
				g.CreatedAt = timestamppb.New(now)
				g.UpdatedAt = timestamppb.New(now.Add(-time.Hour))
				return g
			}(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidTimestamp,
		},
		// MemberCount Tests
		{
			name: "negative MemberCount",
			group: func() *pb.CommunityGroup { g := validGroup(); g.MemberCount = -1; return g }(),
			wantErr: true, expectedErrType: nexuserrors.ErrValueOutOfRange,
		},
		// GroupAvatarUrl Tests
		{
			name: "GroupAvatarUrl invalid format",
			group: func() *pb.CommunityGroup { g := validGroup(); g.GroupAvatarUrl = "not a url"; return g }(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidURL,
		},
		{
			name: "GroupAvatarUrl too long",
			group: func() *pb.CommunityGroup { g := validGroup(); g.GroupAvatarUrl = "http://" + string(make([]byte, pb.MaxGroupAvatarURLLengthCommunityGroup - 6)); return g }(),
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong,
		},
		// GroupBannerUrl Tests
		{
			name: "GroupBannerUrl too long",
			group: func() *pb.CommunityGroup { g := validGroup(); g.GroupBannerUrl = "http://" + string(make([]byte, pb.MaxGroupBannerURLLengthCommunityGroup - 6)); return g }(),
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong,
		},
		// Rules tests
		{
			name: "too many rules",
			group: func() *pb.CommunityGroup { g := validGroup(); g.Rules = make([]string, pb.MaxGroupRulesCommunityGroup + 1); return g}(),
			wantErr: true, expectedErrType: nexuserrors.ErrTooManyItems,
		},
		{
			name: "empty rule string",
			group: func() *pb.CommunityGroup { g := validGroup(); g.Rules = []string{"Rule 1", ""}; return g}(),
			wantErr: true, expectedErrType: nexuserrors.ErrEmptyListItem,
		},
		{
			name: "rule string too long",
			group: func() *pb.CommunityGroup { g := validGroup(); g.Rules = []string{string(make([]byte, pb.MaxRuleLengthCommunityGroup + 1))}; return g}(),
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong,
		},
		// Metadata tests
		{
			name: "too many metadata items for group",
			group: func() *pb.CommunityGroup {
				g := validGroup()
				g.Metadata = make(map[string]string)
				for i := 0; i < pb.MaxMetadataItemsCommunityGroup + 1; i++ { g.Metadata[uuid.NewString()] = "value"}
				return g
			}(),
			wantErr: true, expectedErrType: nexuserrors.ErrTooManyItems,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.group.Validate()
			checkCommunityGroupError(t, err, tc.expectedErrType, tc.wantErr)
		})
	}
}
```
