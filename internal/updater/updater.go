package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const repoOwner = "SamSkinner01"
const repoName = "youtube-downloader"

// Release holds the information returned from the GitHub Releases API.
type Release struct {
	TagName string  `json:"tag_name"`
	HTMLURL string  `json:"html_url"`
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

// Check fetches the latest release from GitHub and returns it if newer than
// current. Returns nil, nil if already up to date or on a dev build.
func Check(current string) (*Release, error) {
	if current == "dev" {
		return nil, nil
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
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

// ApplyUpdate downloads the DMG for the given release, mounts it, copies the
// .app bundle to /Applications, unmounts, then relaunches the new version and
// exits the current process. progress is called with values 0.0–1.0.
func ApplyUpdate(release *Release, progress func(float64)) error {
	// 1. Download DMG to a temp file
	dmgPath, err := downloadDMG(release.DMGUrl(), progress)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(dmgPath)

	// 2. Mount the DMG
	mountPoint, err := mountDMG(dmgPath)
	if err != nil {
		return fmt.Errorf("mount failed: %w", err)
	}

	// 3. Find the .app inside the mounted volume
	srcApp, err := findApp(mountPoint)
	if err != nil {
		_ = unmountDMG(mountPoint)
		return err
	}

	// 4. Copy the .app to /Applications (overwrites the existing installation)
	destApp := "/Applications/" + filepath.Base(srcApp)
	if err := copyApp(srcApp, destApp); err != nil {
		_ = unmountDMG(mountPoint)
		return fmt.Errorf("install failed: %w", err)
	}

	// 5. Unmount
	_ = unmountDMG(mountPoint)

	// 6. Relaunch the new version and quit this process
	if err := exec.Command("open", destApp).Start(); err != nil {
		return fmt.Errorf("relaunch failed: %w", err)
	}
	os.Exit(0)
	return nil
}

func downloadDMG(url string, progress func(float64)) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	tmp, err := os.CreateTemp("", "yt-downloader-update-*.dmg")
	if err != nil {
		return "", err
	}

	total := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := tmp.Write(buf[:n]); werr != nil {
				tmp.Close()
				os.Remove(tmp.Name())
				return "", werr
			}
			downloaded += int64(n)
			if progress != nil && total > 0 {
				progress(float64(downloaded) / float64(total))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return "", err
		}
	}
	tmp.Close()
	return tmp.Name(), nil
}

func mountDMG(dmgPath string) (string, error) {
	out, err := exec.Command(
		"hdiutil", "attach", dmgPath,
		"-nobrowse", "-noautoopen", "-plist",
	).Output()
	if err != nil {
		return "", err
	}

	// Parse the plist output to find the mount point
	// Look for the last /Volumes/... path in the output
	lines := strings.Split(string(out), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "/Volumes/") {
			return line, nil
		}
	}
	return "", fmt.Errorf("could not find mount point in hdiutil output")
}

func unmountDMG(mountPoint string) error {
	return exec.Command("hdiutil", "detach", mountPoint, "-quiet").Run()
}

func findApp(mountPoint string) (string, error) {
	entries, err := os.ReadDir(mountPoint)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".app") {
			return filepath.Join(mountPoint, e.Name()), nil
		}
	}
	return "", fmt.Errorf("no .app found in mounted DMG")
}

func copyApp(src, dest string) error {
	// Remove the old app first so ditto starts fresh
	if err := os.RemoveAll(dest); err != nil && !os.IsNotExist(err) {
		return err
	}
	return exec.Command("ditto", src, dest).Run()
}

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
