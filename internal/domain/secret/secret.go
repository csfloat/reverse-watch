package secret

type SecretKey interface {
	Format() (string, error)
	ID() (string, error)
}

type KeyGenerator interface {
	GenerateSecretKey() (SecretKey, error)
}
