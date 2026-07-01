package browser

import "fmt"

func Open(url string) error {
	if url == "" {
		return fmt.Errorf("empty url")
	}
	return openURL(url)
}
