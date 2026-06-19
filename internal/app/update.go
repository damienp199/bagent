package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

const (
	releaseAPI     = "https://api.github.com/repos/damienp199/bagent/releases/latest"
	updateCheckTTL = 24 * 60 * 60 // secondes
)

func updateCacheFile() string { return filepath.Join(configDir(), ".update-check") }

type updateCache struct {
	CheckedAt int64  `json:"checked_at"`
	Tag       string `json:"tag"`
}

// httpFetch récupère une URL avec un timeout court. Échoue sur statut non-2xx.
func httpFetch(url string) ([]byte, error) {
	cl := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := cl.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("statut HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// latestRelease extrait le tag_name de la dernière release.
func latestRelease(fetch func(string) ([]byte, error)) (string, error) {
	b, err := fetch(releaseAPI)
	if err != nil {
		return "", err
	}
	var r struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return "", err
	}
	if r.TagName == "" {
		return "", errors.New("tag_name vide")
	}
	return r.TagName, nil
}

func readUpdateCache() (updateCache, bool) {
	b, err := os.ReadFile(updateCacheFile())
	if err != nil {
		return updateCache{}, false
	}
	var c updateCache
	if err := json.Unmarshal(b, &c); err != nil {
		return updateCache{}, false
	}
	return c, true
}

func writeUpdateCache(c updateCache) {
	b, err := json.Marshal(c)
	if err != nil {
		return
	}
	_ = os.MkdirAll(configDir(), 0o755)
	_ = os.WriteFile(updateCacheFile(), b, 0o644)
}

// checkForUpdate renvoie le tag à proposer ("" si rien). Sert le cache s'il a
// moins de 24h, sinon interroge l'API et le réécrit. Toute erreur → "".
func checkForUpdate(now int64, fetch func(string) ([]byte, error)) string {
	tag := ""
	if c, ok := readUpdateCache(); ok && now-c.CheckedAt < updateCheckTTL {
		tag = c.Tag
	} else {
		t, err := latestRelease(fetch)
		if err != nil {
			return ""
		}
		tag = t
		writeUpdateCache(updateCache{CheckedAt: now, Tag: tag})
	}
	if isNewer(tag, version) {
		return tag
	}
	return ""
}

// updateShellCmd renvoie la commande shell qui (ré)installe bagent depuis la
// dernière release, isolée pour être testable sans exécution.
func updateShellCmd() string {
	return "curl -fsSL https://raw.githubusercontent.com/damienp199/bagent/main/install.sh | sh"
}
