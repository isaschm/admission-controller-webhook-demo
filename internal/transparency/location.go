package transparency

import (
	"errors"
	"strings"

	"golang.org/x/exp/slices"
)

var (
	euError = errors.New("resource cannot be deployed outside of EU")
)

func ProcessLocation(labels map[string]string, locations []string) error {
	// Since zone is first in locations, we need to only check the first location
	if !strings.HasPrefix(locations[0], "europe") {
		return euError
	}

	if slices.ContainsFunc(locations, func(s string) bool {
		return strings.HasPrefix(s, "europe-west2")
	}) {
		return euError
	}
	return nil
}
