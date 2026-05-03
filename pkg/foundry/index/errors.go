package index

import (
	"errors"
	"fmt"
	"strings"
)

// ErrForbidden indicates a foundry source could not be reached because the
// remote required authentication that the local environment did not satisfy
// (private repo, missing credentials, expired token, 403/401, etc.).
//
// Callers should wrap this with a meaningful message and let the CLI render
// guidance about checking git credentials for the failing URL.
var ErrForbidden = errors.New("foundry forbidden")

// ErrNotFound indicates a remote returned 404 / "repository not found".
// Note that GitHub returns 404 for both missing repos and private repos that
// the caller cannot see, so callers should treat this as auth-adjacent.
var ErrNotFound = errors.New("foundry not found")

// classifyGitError inspects combined git stderr/stdout and the underlying
// error and returns a typed sentinel (ErrForbidden / ErrNotFound) when the
// output matches a known auth/visibility failure. Returns the original error
// unchanged when no pattern matches.
func classifyGitError(err error, gitOutput []byte) error {
	if err == nil {
		return nil
	}
	out := strings.ToLower(string(gitOutput))

	authPatterns := []string{
		"authentication failed",
		"could not read username",
		"could not read password",
		"permission denied",
		"403 forbidden",
		"the requested url returned error: 403",
		"the requested url returned error: 401",
		"access denied",
		"invalid username or password",
	}
	for _, p := range authPatterns {
		if strings.Contains(out, p) {
			return fmt.Errorf("%w: %v\n%s", ErrForbidden, err, gitOutput)
		}
	}

	notFoundPatterns := []string{
		"repository not found",
		"not found",
		"the requested url returned error: 404",
		"does not exist",
	}
	for _, p := range notFoundPatterns {
		if strings.Contains(out, p) {
			return fmt.Errorf("%w: %v\n%s", ErrNotFound, err, gitOutput)
		}
	}

	return fmt.Errorf("%v\n%s", err, gitOutput)
}
