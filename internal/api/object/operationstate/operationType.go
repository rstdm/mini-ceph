package operationstate

//go:generate enumer -type operationType -trimprefix operationType -text
type operationType int

const (
	operationTypeCreate operationType = iota + 1
	operationTypeRead
	operationTypeDelete
)
