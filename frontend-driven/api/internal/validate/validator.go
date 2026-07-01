package validate

type ProviderUser struct {
	ProviderID string
	Email      string
	Name       string
	AvatarURL  string
}

type Validator interface {
	Name() string
	Validate(token string) (*ProviderUser, error)
}
