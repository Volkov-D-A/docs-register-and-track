package coordination

import "context"

// StorageMutation protects an object storage mutation and its matching PostgreSQL
// aggregate update as one logical operation.
type StorageMutation interface {
	Context() context.Context
	Finish() error
}

type StorageMutationCoordinator interface {
	BeginStorageMutation(context.Context) (StorageMutation, error)
}
