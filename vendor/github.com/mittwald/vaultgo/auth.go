package vault

type AuthProvider interface {
	Auth() (*AuthResponse, error)
}
