package store

type Repository interface {
	Get(id string) (any, error)
	Save(entity any) error
}
