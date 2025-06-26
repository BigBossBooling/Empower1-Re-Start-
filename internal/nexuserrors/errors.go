package nexuserrors

import "errors"

// Validation Error Variables for DigiSocialBlock / Nexus Protocol
var (
	// General Field Validation
	ErrMissingField      = errors.New("required field is missing")
	ErrUnexpectedField   = errors.New("field is not expected for this type or context")
	ErrInvalidFormat     = errors.New("field has an invalid format")
	ErrValueOutOfRange   = errors.New("field value is out of the allowed range")
	ErrInvalidReference  = errors.New("field references an invalid or disallowed entity")

	// String Specific Validation
	ErrStringTooLong     = errors.New("string length exceeds maximum limit")
	ErrStringTooShort    = errors.New("string length is below minimum limit")
	ErrInvalidCharacters = errors.New("string contains invalid characters")

	// Identifier Validation
	ErrInvalidUUID = errors.New("invalid UUID format")

	// URL Validation
	ErrInvalidURL = errors.New("invalid URL format")

	// Timestamp Validation
	ErrInvalidTimestamp = errors.New("invalid timestamp value or range") // General, can be wrapped for specifics like "before epoch" or "updated_at < created_at"

	// Enum Validation
	ErrUnknownEnumValue = errors.New("unknown or invalid enum value")

	// List/Array/Repeated Field Validation
	ErrTooManyItems  = errors.New("number of items in list exceeds maximum limit")
	ErrEmptyListItem = errors.New("list item cannot be empty or zero value where required")

	// Map Validation
	ErrInvalidMetadataKey   = errors.New("invalid metadata key")
	ErrInvalidMetadataValue = errors.New("invalid metadata value")
)

// Note: More specific errors can be created by wrapping these base errors with fmt.Errorf,
// e.g., fmt.Errorf("username: %w", ErrStringTooShort)
// Or, by defining more granular error variables if frequently needed.
```
