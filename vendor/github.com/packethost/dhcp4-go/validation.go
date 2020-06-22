package dhcp4

import "fmt"

type Validation interface {
	Validate(p Packet) error
}

func Validate(p Packet, vs []Validation) error {
	for _, v := range vs {
		if err := v.Validate(p); err != nil {
			return err
		}
	}
	return nil
}

type ValidationError struct {
	Option
	MustHave bool
}

func (e *ValidationError) Error() string {
	if e.MustHave {
		return fmt.Sprintf("dhcp4: packet MUST have field %d", e.Option)
	}
	return fmt.Sprintf("dhcp4: packet MUST NOT have field %d", e.Option)
}

type validateMust struct {
	o    Option
	have bool
}

func (v validateMust) Validate(p Packet) error {
	if _, ok := p.GetOption(v.o); v.have != ok {
		return &ValidationError{Option: v.o, MustHave: v.have}
	}
	return nil
}

func ValidateMustNot(o Option) Validation {
	return validateMust{o, false}
}

func ValidateMust(o Option) Validation {
	return validateMust{o, true}
}

type validateAllowedOptions struct {
	allowed map[Option]bool
}

func (v validateAllowedOptions) Validate(p Packet) error {
	for k := range p.OptionMap {
		// If an option is not allowed, the packet MUST NOT have it.
		if !v.allowed[k] {
			return &ValidationError{Option: k, MustHave: false}
		}
	}
	return nil
}

func ValidateAllowedOptions(os []Option) Validation {
	allowed := make(map[Option]bool)
	for _, o := range os {
		allowed[o] = true
	}

	return validateAllowedOptions{allowed}
}
