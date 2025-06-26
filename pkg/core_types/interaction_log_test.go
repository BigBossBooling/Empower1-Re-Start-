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

// checkInteractionLogError helper
func checkInteractionLogError(t *testing.T, gotErr, wantErrType error, wantErr bool) {
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

func TestInteractionLog_Validate(t *testing.T) {
	validInteractionUUID := uuid.New().String()
	validUserUUID := uuid.New().String()
	validTargetContentUUID := uuid.New().String()
	now := time.Now().UTC()

	// Use generated enum values

	validLikeInteraction := func() *pb.InteractionLog {
		return &pb.InteractionLog{
			InteractionId:     validInteractionUUID,
			UserId:            validUserUUID,
			TargetContentId:   validTargetContentUUID,
			TargetContentType: pb.InteractableContentType_INTERACTABLE_CONTENT_TYPE_POST,
			InteractionType:   pb.InteractionType_INTERACTION_TYPE_LIKE,
			Timestamp:         timestamppb.New(now),
			AdditionalData:    map[string]string{"source": "feed"},
		}
	}

	validCommentInteraction := func() *pb.InteractionLog {
		return &pb.InteractionLog{
			InteractionId:     uuid.New().String(),
			UserId:            validUserUUID,
			TargetContentId:   validTargetContentUUID,
			TargetContentType: pb.InteractableContentType_INTERACTABLE_CONTENT_TYPE_POST,
			InteractionType:   pb.InteractionType_INTERACTION_TYPE_COMMENT,
			CommentText:       "This is a valid comment.",
			Timestamp:         timestamppb.New(now),
		}
	}

	tests := []struct {
		name            string
		log             *pb.InteractionLog
		wantErr         bool
		expectedErrType error
	}{
		{name: "valid LIKE interaction", log: validLikeInteraction(), wantErr: false},
		{name: "valid COMMENT interaction", log: validCommentInteraction(), wantErr: false},
		// InteractionId Tests
		{
			name: "missing InteractionId",
			log:  func() *pb.InteractionLog { l := validLikeInteraction(); l.InteractionId = ""; return l }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		{
			name: "invalid InteractionId format",
			log:  func() *pb.InteractionLog { l := validLikeInteraction(); l.InteractionId = "not-a-uuid"; return l }(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidUUID,
		},
		// UserId Tests
		{
			name: "missing UserId",
			log:  func() *pb.InteractionLog { l := validLikeInteraction(); l.UserId = ""; return l }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		// TargetContentId Tests
		{
			name: "missing TargetContentId",
			log:  func() *pb.InteractionLog { l := validLikeInteraction(); l.TargetContentId = ""; return l }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		// TargetContentType Tests
		{
			name: "unspecified TargetContentType",
			log:  func() *pb.InteractionLog { l := validLikeInteraction(); l.TargetContentType = pb.InteractableContentType_INTERACTABLE_CONTENT_TYPE_UNSPECIFIED; return l }(),
			wantErr: true, expectedErrType: nexuserrors.ErrUnknownEnumValue,
		},
		// InteractionType Tests
		{
			name: "unspecified InteractionType",
			log:  func() *pb.InteractionLog { l := validLikeInteraction(); l.InteractionType = pb.InteractionType_INTERACTION_TYPE_UNSPECIFIED; return l }(),
			wantErr: true, expectedErrType: nexuserrors.ErrUnknownEnumValue,
		},
		// CommentText Tests
		{
			name: "missing CommentText for COMMENT type",
			log:  func() *pb.InteractionLog { l := validCommentInteraction(); l.CommentText = ""; return l }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		{
			name: "CommentText too long",
			log:  func() *pb.InteractionLog { l := validCommentInteraction(); l.CommentText = string(make([]byte, pb.MaxCommentTextLengthInteractionLog + 1)); return l }(),
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong,
		},
		{
			name: "unexpected CommentText for LIKE type",
			log:  func() *pb.InteractionLog { l := validLikeInteraction(); l.CommentText = "this shouldn't be here"; return l }(),
			wantErr: true, expectedErrType: nexuserrors.ErrUnexpectedField,
		},
		// Timestamp Tests
		{
			name: "missing Timestamp",
			log:  func() *pb.InteractionLog { l := validLikeInteraction(); l.Timestamp = nil; return l }(),
			wantErr: true, expectedErrType: nexuserrors.ErrMissingField,
		},
		{
            name: "Timestamp is zero",
            log:  func() *pb.InteractionLog { l := validLikeInteraction(); l.Timestamp = &timestamppb.Timestamp{Seconds:0, Nanos:0}; return l }(),
            wantErr: true, expectedErrType: nexuserrors.ErrInvalidTimestamp,
        },
		// ParentInteractionId Tests
		{
			name: "invalid ParentInteractionId format",
			log:  func() *pb.InteractionLog { l := validCommentInteraction(); l.ParentInteractionId = "not-a-uuid"; return l }(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidUUID,
		},
		{
			name: "ParentInteractionId same as InteractionId",
			log:  func() *pb.InteractionLog { l := validCommentInteraction(); l.ParentInteractionId = l.InteractionId; return l }(),
			wantErr: true, expectedErrType: nexuserrors.ErrInvalidReference,
		},
		// AdditionalData tests
		{
			name: "too many additional_data items",
			log:  func() *pb.InteractionLog {
				l := validLikeInteraction()
				l.AdditionalData = make(map[string]string)
				for i := 0; i < 11; i++ { l.AdditionalData[uuid.NewString()] = "value"}
				return l
			}(),
			wantErr: true, expectedErrType: nexuserrors.ErrTooManyItems,
		},
		{
			name: "additional_data key too long",
			log:  func() *pb.InteractionLog {
				l := validLikeInteraction()
				l.AdditionalData = map[string]string{string(make([]byte, 51)): "value"}
				return l
			}(),
			wantErr: true, expectedErrType: nexuserrors.ErrStringTooLong, // Assuming CheckStringLength used for key
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.log.Validate()
			checkInteractionLogError(t, err, tc.expectedErrType, tc.wantErr)
		})
	}
}
```
