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

// checkContentPostError helper
func checkContentPostError(t *testing.T, gotErr, wantErrType error, wantErr bool) {
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


func TestContentPost_Validate(t *testing.T) {
	validPostUUID := uuid.New().String()
	validAuthorUUID := uuid.New().String()
	now := time.Now().UTC()

	// Use generated enum values directly from pb package

	validTextPost := func() *pb.ContentPost {
		return &pb.ContentPost{
			PostId:      validPostUUID,
			AuthorId:    validAuthorUUID,
			ContentType: pb.ContentType_CONTENT_TYPE_TEXT,
			TextContent: "This is valid text content.",
			MediaUrl:    "",
			Visibility:  pb.VisibilitySetting_VISIBILITY_SETTING_PUBLIC,
			Tags:        []string{"valid", "tag"},
			CreatedAt:   timestamppb.New(now.Add(-time.Hour)),
			UpdatedAt:   timestamppb.New(now),
		}
	}

	validImagePost := func() *pb.ContentPost {
		return &pb.ContentPost{
			PostId:      uuid.New().String(),
			AuthorId:    validAuthorUUID,
			ContentType: pb.ContentType_CONTENT_TYPE_IMAGE,
			TextContent: "",
			MediaUrl:    "http://example.com/image.png",
			Visibility:  pb.VisibilitySetting_VISIBILITY_SETTING_FRIENDS_ONLY,
			CreatedAt:   timestamppb.New(now.Add(-time.Hour)),
			UpdatedAt:   timestamppb.New(now),
		}
	}

	tests := []struct {
		name            string
		post            *pb.ContentPost
		wantErr         bool
		expectedErrType error
	}{
		{name: "valid text post", post: validTextPost(), wantErr: false},
		{name: "valid image post", post: validImagePost(), wantErr: false},
		// PostId Tests
		{
			name: "missing PostId",
			post: func() *pb.ContentPost { p := validTextPost(); p.PostId = ""; return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		{
			name: "invalid PostId format",
			post: func() *pb.ContentPost { p := validTextPost(); p.PostId = "not-a-uuid"; return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidUUID,
		},
		// AuthorId Tests
		{
			name: "missing AuthorId",
			post: func() *pb.ContentPost { p := validTextPost(); p.AuthorId = ""; return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		{
			name: "invalid AuthorId format",
			post: func() *pb.ContentPost { p := validTextPost(); p.AuthorId = "not-a-uuid"; return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidUUID,
		},
		// ContentType Tests
		{
			name: "invalid ContentType enum",
			post: func() *pb.ContentPost { p := validTextPost(); p.ContentType = pb.ContentType(99); return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrUnknownEnumValue,
		},
		{
			name: "unspecified ContentType",
			post: func() *pb.ContentPost { p := validTextPost(); p.ContentType = pb.ContentType_CONTENT_TYPE_UNSPECIFIED; return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrUnknownEnumValue,
		},
		// TextContent Tests
		{
			name: "missing TextContent for TEXT type",
			post: func() *pb.ContentPost { p := validTextPost(); p.TextContent = ""; return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		{
			name: "TextContent too long",
			post: func() *pb.ContentPost { p := validTextPost(); p.TextContent = string(make([]byte, pb.MaxTextContentLengthContentPost + 1)); return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong,
		},
		{
			name: "unexpected TextContent for IMAGE type",
			post: func() *pb.ContentPost { p := validImagePost(); p.TextContent = "some text"; return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrUnexpectedField,
		},
		// MediaUrl Tests
		{
			name: "missing MediaUrl for IMAGE type",
			post: func() *pb.ContentPost { p := validImagePost(); p.MediaUrl = ""; return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		{
			name: "MediaUrl invalid format",
			post: func() *pb.ContentPost { p := validImagePost(); p.MediaUrl = "not a url"; return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidURL,
		},
		{
			name: "MediaUrl too long",
			post: func() *pb.ContentPost { p := validImagePost(); p.MediaUrl = "http://" + string(make([]byte, pb.MaxMediaURLLengthContentPost - 6)) ; return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong,
		},
		{
			name: "unexpected MediaUrl for TEXT type",
			post: func() *pb.ContentPost { p := validTextPost(); p.MediaUrl = "http://example.com/image.png"; return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrUnexpectedField,
		},
		// Visibility Tests
		{
			name: "invalid visibility enum",
			post: func() *pb.ContentPost { p := validTextPost(); p.Visibility = pb.VisibilitySetting(99); return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrUnknownEnumValue,
		},
		// Tags Tests
		{
			name: "too many tags",
			post: func() *pb.ContentPost { p := validTextPost(); p.Tags = make([]string, pb.MaxTagsContentPost + 1); return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrTooManyItems,
		},
		{
			name: "empty tag in list",
			post: func() *pb.ContentPost { p := validTextPost(); p.Tags = []string{"valid", ""}; return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrEmptyListItem,
		},
		{
			name: "tag too long",
			post: func() *pb.ContentPost { p := validTextPost(); p.Tags = []string{string(make([]byte, pb.MaxTagLengthContentPost + 1))}; return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong,
		},
		{
			name: "tag invalid characters",
			post: func() *pb.ContentPost { p := validTextPost(); p.Tags = []string{"valid tag!"}; return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidCharacters,
		},
		// Timestamp tests
		{
			name: "missing CreatedAt",
			post: func() *pb.ContentPost { p := validTextPost(); p.CreatedAt = nil; return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		{
			name: "UpdatedAt before CreatedAt",
			post: func() *pb.ContentPost {
				p := validTextPost()
				p.CreatedAt = timestamppb.New(now)
				p.UpdatedAt = timestamppb.New(now.Add(-time.Hour))
				return p
			}(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidTimestamp,
		},
		// Optional Foreign Key Tests
		{
			name: "invalid CommunityGroupId format",
			post: func() *pb.ContentPost { p := validTextPost(); p.CommunityGroupId = "not-a-uuid"; return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidUUID,
		},
		{
			name: "invalid ParentPostId format",
			post: func() *pb.ContentPost { p := validTextPost(); p.ParentPostId = "not-a-uuid"; return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidUUID,
		},
		{
			name: "ParentPostId same as PostId",
			post: func() *pb.ContentPost { p := validTextPost(); p.ParentPostId = p.PostId; return p }(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidReference,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.post.Validate()
			checkContentPostError(t, err, tc.expectedErrType, tc.wantErr)
		})
	}
}
```
