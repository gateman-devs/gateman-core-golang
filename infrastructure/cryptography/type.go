package cryptography

type Hasher interface {
	HashString(data string, salt []byte) ([]byte, error)
	VerifyHashData(hash string, data string) bool
}
