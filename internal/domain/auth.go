package domain

type Auth struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthRepository interface {
	CreateAuth(auth *Auth) error
	GetAuthByUsername(username string) (*Auth, error)
}

type AuthService interface {
	Register(auth *Auth) error
	Login(username, password string) (*Auth, error)
}
