package cardconjurer

type CardInfo interface {
	GetFullName() string
	GetCount() int
	GetName() string
	GetSet() string
	GetCollectorNumber() string
}
