package cli

import (
	"fmt"
	"strings"

	"github.com/finetension/toss-openapi-cli/internal/apperr"
)

func allowedValue(flagName string, value string, allowed ...string) (string, error) {
	trimmed := strings.TrimSpace(value)
	for _, candidate := range allowed {
		if strings.EqualFold(trimmed, candidate) {
			return candidate, nil
		}
	}
	return "", apperr.Usage(fmt.Sprintf("%s must be one of %s", flagName, strings.Join(allowed, ", ")))
}
