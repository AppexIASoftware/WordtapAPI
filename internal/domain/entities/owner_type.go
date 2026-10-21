package entities

// OwnerType identifies who owns and manages a resource.
type OwnerType string

const (
	OwnerTypePlatform OwnerType = "platform"
	OwnerTypeTeacher  OwnerType = "teacher"
)

// IsValid reports whether the owner type is a supported value.
func (o OwnerType) IsValid() bool {
	return o == OwnerTypePlatform || o == OwnerTypeTeacher
}
