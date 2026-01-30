package repository

type Factory interface {
	Private() PrivateRepository
	Public() PublicRepository
}
