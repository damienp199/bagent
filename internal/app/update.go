package app

import (
	"strconv"
	"strings"
)

// version est injectée au build via ldflags
// (-X github.com/damienp199/bagent/internal/app.version=vX.Y.Z).
// "dev" pour un build local : désactive la notif de mise à jour.
var version = "dev"

// parseSemver découpe "vMAJOR.MINOR.PATCH" ; ok=false si non parsable.
func parseSemver(tag string) ([3]int, bool) {
	parts := strings.Split(strings.TrimPrefix(tag, "v"), ".")
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var out [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}, false
		}
		out[i] = n
	}
	return out, true
}

// isNewer indique si latest est strictement plus récent que current.
// Renvoie false pour un build "dev" ou un tag non parsable (jamais de fausse notif).
func isNewer(latest, current string) bool {
	if current == "dev" {
		return false
	}
	l, ok1 := parseSemver(latest)
	c, ok2 := parseSemver(current)
	if !ok1 || !ok2 {
		return false
	}
	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}
