package cryptography

type Hasher interface{
	HashString(data string) ([]byte, error)
	VerifyHashData(hash string, data string) bool
}