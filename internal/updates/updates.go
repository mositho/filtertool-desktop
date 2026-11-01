// Package updates connects Wails' updater to Filtertool's public releases.
package updates

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

const Repository = "mositho/filtertool-desktop"

type Release struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
	Major   bool   `json:"major"`
}

type Status struct {
	Enabled bool     `json:"enabled"`
	Version string   `json:"version"`
	Ready   bool     `json:"ready"`
	Release *Release `json:"release,omitempty"`
}

type Manager struct {
	mu      sync.Mutex
	updater *updater.Updater
	status  Status
}

func New(u *updater.Updater, version string, enabled bool) (*Manager, error) {
	m := &Manager{updater: u, status: Status{Version: version, Enabled: enabled && version != "dev"}}
	if !m.status.Enabled {
		return m, nil
	}
	p, err := github.New(github.Config{
		Repository: Repository, ChecksumAsset: "SHA256SUMS.txt", AssetMatcher: matchExecutable,
		HTTPClient: &http.Client{Timeout: 5 * time.Minute},
	})
	if err != nil {
		return nil, err
	}
	err = u.Init(updater.Config{CurrentVersion: version, Providers: []updater.Provider{verifiedProvider{p}}, Window: updater.WindowNone})
	return m, err
}

// Never select the similarly named installer: Wails swaps the running EXE.
func matchExecutable(req updater.CheckRequest, assets []github.ReleaseAsset) int {
	if req.Platform == "windows" && req.Arch == "amd64" {
		for i, asset := range assets {
			if asset.Name == "Filtertool.exe" {
				return i
			}
		}
	}
	return -1
}

// Wails' GitHub provider permits missing checksum files. Filtertool does not.
type verifiedProvider struct{ updater.Provider }

func (p verifiedProvider) Check(ctx context.Context, req updater.CheckRequest) (*updater.Release, error) {
	r, err := p.Provider.Check(ctx, req)
	if err == nil && r != nil && (r.Verification == nil || r.Verification.DigestAlgo != "sha256" || len(r.Verification.Digest) != 32) {
		return nil, errors.New("this release has no valid SHA-256 checksum; update manually from GitHub")
	}
	return r, err
}

func majorUpgrade(current, next string) bool {
	major := func(v string) int {
		n, _ := strconv.Atoi(strings.SplitN(strings.TrimPrefix(v, "v"), ".", 2)[0])
		return n
	}
	return major(next) > major(current)
}

func (m *Manager) Check() (Status, error) {
	if !m.mu.TryLock() {
		return Status{}, errors.New("an update operation is already in progress")
	}
	defer m.mu.Unlock()
	if !m.status.Enabled || m.status.Ready {
		return m.status, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r, err := m.updater.Check(ctx)
	if err != nil {
		return Status{}, err
	}
	m.status.Release = nil
	if r != nil {
		m.status.Release = &Release{Version: r.Version, Notes: r.Notes, Major: majorUpgrade(m.status.Version, r.Version)}
	}
	return m.status, nil
}

func (m *Manager) Download(version string, acceptMajor bool) (Status, error) {
	if !m.mu.TryLock() {
		return Status{}, errors.New("an update operation is already in progress")
	}
	defer m.mu.Unlock()
	r := m.status.Release
	if !m.status.Enabled || r == nil || r.Version != version {
		return Status{}, errors.New("check for updates again before downloading")
	}
	if r.Major && !acceptMajor {
		return Status{}, errors.New("confirm the breaking-change warning before updating")
	}
	if !m.status.Ready {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := m.updater.DownloadAndInstall(ctx); err != nil {
			return Status{}, err
		}
		m.status.Ready = true
	}
	return m.status, nil
}

func (m *Manager) Restart() error {
	if !m.mu.TryLock() {
		return errors.New("an update operation is already in progress")
	}
	defer m.mu.Unlock()
	if !m.status.Ready {
		return errors.New("download the update before restarting")
	}
	return m.updater.Restart(context.Background())
}
