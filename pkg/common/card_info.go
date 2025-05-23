package common

type CardInfo interface {
	GetFullName() string
	GetCount() int
	GetName() string
	GetSanitizedName() string
	GetSet() string
	GetCollectorNumber() string
}
