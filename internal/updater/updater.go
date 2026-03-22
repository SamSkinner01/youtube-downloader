package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const repoOwner = "SamSkinner01"
const repoName = "youtube-downloader"

// Release holds the information returned from the GitHub Releases API.
type Release struct {
	TagName string  `json:"tag_name"` // e.g. "v1.2.0"
	HTMLURL string  `json:"html_url"` // release page URL
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// DMGUrl returns the download URL for the DMG asset, or falls back to the
// release page URL if no DMG asset is attached.
func (r Release) DMGUrl() string {
	for _, a := range r.Assets {
		if strings.HasSuffix(a.Name, ".dmg") {
			return a.BrowserDownloadURL
		}
	}
	return r.HTMLURL
}

// Check fetches the latest release from GitHub and returns it if the tag
// version is newer than current. Returns nil, nil if already up to date.
func Check(current string) (*Release, error) {
	if current == "dev" {
		return nil, nil // skip update checks for local dev builds
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil // no releases yet or private repo — silently skip
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	if isNewer(release.TagName, current) {
		return &release, nil
	}
	return nil, nil
}

// isNewer returns true if latest (e.g. "v1.2.0") is newer than current (e.g. "1.1.0").
func isNewer(latest, current string) bool {
	l := parseVersion(strings.TrimPrefix(latest, "v"))
	c := parseVersion(strings.TrimPrefix(current, "v"))
	for i := range l {
		if i >= len(c) {
			return true
		}
		if l[i] > c[i] {
			return true
		}
		if l[i] < c[i] {
			return false
		}
	}
	return false
}

func parseVersion(v string) []int {
	parts := strings.Split(v, ".")
	nums := make([]int, len(parts))
	for i, p := range parts {
		fmt.Sscanf(p, "%d", &nums[i])
	}
	return nums
}
