package validate

import "fmt"

type Registry struct {
	validators map[string]Validator
}

func NewRegistry() *Registry {
	return &Registry{validators: make(map[string]Validator)}
}

func (r *Registry) Register(v Validator) {
	r.validators[v.Name()] = v
}

func (r *Registry) Get(name string) (Validator, error) {
	v, ok := r.validators[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
	return v, nil
}
