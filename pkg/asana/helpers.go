package asana

import (
	"net/url"
)

func getPath(base string, elem ...string) (*url.URL, error) {
	fullPath, err := url.JoinPath(base, elem...)
	if err != nil {
		return nil, err
	}

	return url.Parse(fullPath)
}
