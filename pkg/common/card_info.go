package common

type CardInfo interface {
	GetFullName() string
	GetCount() int
	GetNameFront() string
	GetNameBack() string
	GetSanitizedNameFront() string
	GetSanitizedNameBack() string
	GetSet() string
	GetCollectorNumber() string
}
