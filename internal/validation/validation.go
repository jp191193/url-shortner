package validation

import (
	"errors"
	"regexp"
)

var aliasRegex = regexp.MustCompile(`^[a-zA-Z0-9\-]{3,30}$`)

func ValidateAlias(alias string) (bool, error) {
	if alias == "" {
		return true, nil // alias is optional
	}

	if !aliasRegex.MatchString(alias) {
		return false, errors.New("alias must be 3-30 chars, letters/numbers/hyphens only")
	}

	return true, nil
}
