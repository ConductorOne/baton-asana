package asana

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func getPath(base string, elem ...string) (*url.URL, error) {
	fullPath, err := url.JoinPath(base, elem...)
	if err != nil {
		return nil, err
	}

	return url.Parse(fullPath)
}

// FormatAsanaError formats and returns a detailed error message from an AsanaError.
// This helper extracts all available error information from Asana API responses.
// If originalErr is not nil and asanaError has errors, it joins the errors using errors.Join.
func FormatAsanaError(asanaError *AsanaError, originalErr error) error {
	if asanaError == nil || len(asanaError.Errors) == 0 {
		if originalErr != nil {
			return originalErr
		}
		return fmt.Errorf("unknown error from Asana API: %w", originalErr)
	}

	var sb strings.Builder
	var writeErr error

	// For single errors, format a detailed message with all available fields
	if len(asanaError.Errors) == 1 {
		err := asanaError.Errors[0]
		if _, writeErr = sb.WriteString(fmt.Sprintf("Asana API error: %s", err.Message)); writeErr != nil {
			if originalErr != nil {
				return errors.Join(fmt.Errorf("failed to format Asana error: %w", writeErr), originalErr)
			}
			return errors.Join(fmt.Errorf("failed to format Asana error: %w", writeErr), originalErr)
		}

		// Include all available fields
		if err.Help != "" {
			if _, writeErr = sb.WriteString(fmt.Sprintf(" (Help: %s)", err.Help)); writeErr != nil {
				if originalErr != nil {
					return errors.Join(fmt.Errorf("failed to format Asana error: %w", writeErr), originalErr)
				}
				return errors.Join(fmt.Errorf("failed to format Asana error: %w", writeErr), originalErr)
			}
		}
		if err.Phrase != "" {
			if _, writeErr = sb.WriteString(fmt.Sprintf(" (Reference: %s)", err.Phrase)); writeErr != nil {
				if originalErr != nil {
					return errors.Join(fmt.Errorf("failed to format Asana error: %w", writeErr), originalErr)
				}
				return errors.Join(fmt.Errorf("failed to format Asana error: %w", writeErr), originalErr)
			}
		}

		// Join the original error if it exists
		if originalErr != nil {
			asanaErr := fmt.Errorf("%s", sb.String())
			return errors.Join(asanaErr, originalErr)
		}
		return errors.Join(fmt.Errorf("%s", sb.String()), originalErr)
	}

	// For multiple errors, format them all with all available fields
	if _, writeErr = sb.WriteString("Multiple Asana API errors:"); writeErr != nil {
		if originalErr != nil {
			return errors.Join(fmt.Errorf("failed to format Asana error: %w", writeErr), originalErr)
		}
		return errors.Join(fmt.Errorf("failed to format Asana error: %w", writeErr), originalErr)
	}

	for i, err := range asanaError.Errors {
		if _, writeErr = sb.WriteString(fmt.Sprintf("\n  %d. %s", i+1, err.Message)); writeErr != nil {
			if originalErr != nil {
				return errors.Join(fmt.Errorf("failed to format Asana error: %w", writeErr), originalErr)
			}
			return errors.Join(fmt.Errorf("failed to format Asana error: %w", writeErr), originalErr)
		}

		// Include all available fields for each error
		if err.Help != "" {
			if _, writeErr = sb.WriteString(fmt.Sprintf(" (Help: %s)", err.Help)); writeErr != nil {
				if originalErr != nil {
					return errors.Join(fmt.Errorf("failed to format Asana error: %w", writeErr), originalErr)
				}
				return errors.Join(fmt.Errorf("failed to format Asana error: %w", writeErr), originalErr)
			}
		}
		if err.Phrase != "" {
			if _, writeErr = sb.WriteString(fmt.Sprintf(" (Reference: %s)", err.Phrase)); writeErr != nil {
				if originalErr != nil {
					return errors.Join(fmt.Errorf("failed to format Asana error: %w", writeErr), originalErr)
				}
				return errors.Join(fmt.Errorf("failed to format Asana error: %w", writeErr), originalErr)
			}
		}
	}

	// Join the original error if it exists
	if originalErr != nil {
		asanaErr := fmt.Errorf("%s", sb.String())
		return errors.Join(asanaErr, originalErr)
	}
	return errors.Join(fmt.Errorf("%s", sb.String()), originalErr)
}
