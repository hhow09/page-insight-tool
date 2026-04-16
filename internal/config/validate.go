package config

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

// structValidator is safe for concurrent use; see [validator.Validate.Struct].
var structValidator = validator.New()

// Validate runs [github.com/go-playground/validator/v10] rules defined on [Config]
// and nested [FetchConfig] / [LinkConfig] struct tags.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config: nil Config")
	}
	if err := structValidator.Struct(c); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	return nil
}
