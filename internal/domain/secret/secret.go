package secret

type SecretKey interface {
	Format() string
	ID() string
}

type KeyGenerator interface {
	GenerateSecretKey() (SecretKey, error)
}
