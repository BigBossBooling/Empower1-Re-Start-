package validationutils

// ProjectEpochStartUnix defines the earliest acceptable timestamp for records (e.g., Jan 1, 2023 UTC).
// Timestamps before this are considered invalid.
const ProjectEpochStartUnix int64 = 1672531200 // Jan 1, 2023 00:00:00 UTC
```

Now, update `pkg/core_types/user_profile_validation.go` to use this:
