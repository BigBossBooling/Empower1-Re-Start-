package validationutils_test

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"empower1/internal/nexuserrors"
	"empower1/internal/validationutils"
	"github.com/google/uuid"
)

// Helper for checking errors in utils tests
func checkUtilError(t *testing.T, gotErr, wantErrType error, wantErr bool) {
	t.Helper()
	if wantErr {
		if gotErr == nil {
			t.Errorf("expected an error but got nil")
			return
		}
		if wantErrType != nil && !errors.Is(gotErr, wantErrType) {
			t.Errorf("got error '%v', want error to wrap or be '%v'", gotErr, wantErrType)
		}
	} else if gotErr != nil {
		t.Errorf("did not expect an error but got: %v", gotErr)
	}
}

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{"valid uuid", uuid.New().String(), true},
		{"invalid uuid - empty", "", false},
		{"invalid uuid - short", "123", false},
		{"invalid uuid - malformed", "not-a-uuid-string", false},
		{"valid uuid - all zeros", "00000000-0000-0000-0000-000000000000", true}, // Zero UUID is valid format
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validationutils.IsValidUUID(tt.id); got != tt.want {
				t.Errorf("IsValidUUID() = %v, want %v for id '%s'", got, tt.want, tt.id)
			}
		})
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name           string
		urlStr         string
		allowedSchemes []string
		want           bool
	}{
		{"valid http", "http://example.com", nil, true},
		{"valid https", "https://example.com/path?query=1", nil, true},
		{"valid custom scheme", "custom://data", []string{"custom"}, true},
		{"invalid scheme", "ftp://example.com", nil, false},
		{"invalid scheme with custom list", "ftp://example.com", []string{"http", "https"}, false},
		{"malformed url", "http//example.com", nil, false},
		{"empty url (valid as optional)", "", nil, true}, // IsValidURL itself allows empty
		{"url without scheme", "example.com", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validationutils.IsValidURL(tt.urlStr, tt.allowedSchemes); got != tt.want {
				t.Errorf("IsValidURL() = %v, want %v for url '%s' with schemes %v", got, tt.want, tt.urlStr, tt.allowedSchemes)
			}
		})
	}
}

func TestCheckStringLength(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		fieldName string
		minLen    int
		maxLen    int
		wantErr   bool
		errType   error
	}{
		{"valid length", "test", "field", 3, 10, false, nil},
		{"exact min length", "abc", "field", 3, 10, false, nil},
		{"exact max length", "0123456789", "field", 0, 10, false, nil},
		{"too short", "hi", "field", 3, 10, true, nexuserrors.ErrStringTooShort},
		{"too long", "thisiswaytoolong", "field", 0, 10, true, nexuserrors.ErrStringTooLong},
		{"empty allowed (min 0)", "", "field", 0, 10, false, nil},
		{"empty not allowed (min 1)", "", "field", 1, 10, true, nexuserrors.ErrStringTooShort},
		{"no max len (max 0)", string(make([]byte, 100)), "field", 1, 0, false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validationutils.CheckStringLength(tt.s, tt.fieldName, tt.minLen, tt.maxLen)
			checkUtilError(t, err, tt.errType, tt.wantErr)
		})
	}
}

func TestCheckAllowedChars(t *testing.T) {
	alphaNum := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	tests := []struct {
		name      string
		s         string
		fieldName string
		pattern   *regexp.Regexp
		wantErr   bool
		errType   error
	}{
		{"valid chars", "AlphaNum123", "user", alphaNum, false, nil},
		{"invalid chars - space", "Alpha Num", "user", alphaNum, true, nexuserrors.ErrInvalidCharacters},
		{"invalid chars - symbol", "Alpha!", "user", alphaNum, true, nexuserrors.ErrInvalidCharacters},
		{"empty string valid for pattern", "", "user", alphaNum, false, nil}, // Pattern dependent
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validationutils.CheckAllowedChars(tt.s, tt.fieldName, tt.pattern)
			checkUtilError(t, err, tt.errType, tt.wantErr)
		})
	}
}

// Mock enum for testing CheckEnumValue
type MockStatus int32
const (
	MockStatus_MOCK_STATUS_UNSPECIFIED MockStatus = 0
	MockStatus_MOCK_STATUS_ACTIVE      MockStatus = 1
	MockStatus_MOCK_STATUS_INACTIVE    MockStatus = 2
)
var MockStatus_name = map[int32]string{0: "MOCK_STATUS_UNSPECIFIED", 1: "MOCK_STATUS_ACTIVE", 2: "MOCK_STATUS_INACTIVE"}

func TestCheckEnumValue(t *testing.T) {
	tests := []struct {
		name             string
		val              MockStatus
		enumNameMap      map[int32]string
		fieldName        string
		unspecifiedValue MockStatus
		enumTypeName     string
		wantErr          bool
		errType          error
	}{
		{"valid enum value", MockStatus_MOCK_STATUS_ACTIVE, MockStatus_name, "status", MockStatus_MOCK_STATUS_UNSPECIFIED, "MockStatus", false, nil},
		{"invalid enum value (unspecified)", MockStatus_MOCK_STATUS_UNSPECIFIED, MockStatus_name, "status", MockStatus_MOCK_STATUS_UNSPECIFIED, "MockStatus", true, nexuserrors.ErrUnknownEnumValue},
		{"invalid enum value (out of map)", MockStatus(99), MockStatus_name, "status", MockStatus_MOCK_STATUS_UNSPECIFIED, "MockStatus", true, nexuserrors.ErrUnknownEnumValue},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validationutils.CheckEnumValue(tt.val, tt.enumNameMap, tt.fieldName, tt.unspecifiedValue, tt.enumTypeName)
			checkUtilError(t, err, tt.errType, tt.wantErr)
		})
	}
}

func TestCheckTimestamp(t *testing.T) {
	epoch := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	now := time.Now()

	// Convert time.Time to validationutils.Timestamp for testing
	toValidationTimestamp := func(t time.Time) *validationutils.Timestamp {
		if t.IsZero() { return &validationutils.Timestamp{Seconds:0, Nanos:0}} // Protobuf zero
		return &validationutils.Timestamp{Seconds: t.Unix(), Nanos: int32(t.Nanosecond())}
	}
	nilTimestamp := (*validationutils.Timestamp)(nil)


	tests := []struct {
		name        string
		ts          *validationutils.Timestamp
		fieldName   string
		epoch       int64
		allowFuture bool
		futureLimit time.Duration
		wantErr     bool
		errType     error
	}{
		{"valid timestamp", toValidationTimestamp(now), "created_at", epoch, true, 2 * time.Hour, false, nil},
		{"nil timestamp", nilTimestamp, "created_at", epoch, false, 0, true, nexuserrors.ErrMissingField},
		{"zero timestamp", toValidationTimestamp(time.Time{}), "created_at", epoch, false, 0, true, nexuserrors.ErrInvalidTimestamp},
		{"before epoch", toValidationTimestamp(time.Unix(epoch-1, 0)), "created_at", epoch, false, 0, true, nexuserrors.ErrInvalidTimestamp},
		{"future not allowed - valid now", toValidationTimestamp(now.Add(-time.Minute)), "created_at", epoch, false, 0, false, nil},
		{"future not allowed - too far future", toValidationTimestamp(now.Add(5 * time.Minute)), "created_at", epoch, false, 0, true, nexuserrors.ErrInvalidTimestamp},
		{"future allowed - within limit", toValidationTimestamp(now.Add(1 * time.Hour)), "created_at", epoch, true, 2 * time.Hour, false, nil},
		{"future allowed - exceeds limit", toValidationTimestamp(now.Add(3 * time.Hour)), "created_at", epoch, true, 2 * time.Hour, true, nexuserrors.ErrInvalidTimestamp},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validationutils.CheckTimestamp(tt.ts, tt.fieldName, tt.epoch, tt.allowFuture, tt.futureLimit)
			checkUtilError(t, err, tt.errType, tt.wantErr)
		})
	}
}

func TestCheckLogicalTimestampOrder(t *testing.T) {
	now := time.Now()
	t1 := &validationutils.Timestamp{Seconds: now.Unix()}
	t2Later := &validationutils.Timestamp{Seconds: now.Add(time.Hour).Unix()}
	t2Earlier := &validationutils.Timestamp{Seconds: now.Add(-time.Hour).Unix()}

	tests := []struct {
		name    string
		t1      *validationutils.Timestamp
		t2      *validationutils.Timestamp
		fName1  string
		fName2  string
		wantErr bool
		errType error
	}{
		{"t2 after t1", t1, t2Later, "created", "updated", false, nil},
		{"t2 same as t1", t1, t1, "created", "updated", false, nil},
		{"t2 before t1", t1, t2Earlier, "created", "updated", true, nexuserrors.ErrInvalidTimestamp},
		{"t1 nil", nil, t2Later, "created", "updated", false, nil}, // Should not error if caught by individual checks
		{"t2 nil", t1, nil, "created", "updated", false, nil}, // Should not error
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validationutils.CheckLogicalTimestampOrder(tt.t1, tt.t2, tt.fName1, tt.fName2)
			checkUtilError(t, err, tt.errType, tt.wantErr)
		})
	}
}

```
