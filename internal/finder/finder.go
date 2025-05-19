package finder

import (
	"github.com/ktr0731/go-fuzzyfinder"
)

func Select(hosts []string) (string, error) {
	idx, err := fuzzyfinder.Find(
		hosts,
		func(i int) string {
			return hosts[i]
		},
		fuzzyfinder.WithHotReload(),
	)
	if err != nil {
		return "", err
	}
	return hosts[idx], nil
}
