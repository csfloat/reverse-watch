package secret

type SecretKey interface {
	Format() string
	Hash() string
}

type KeyGenerator interface {
	GenerateSecretKey() (SecretKey, error)
}
