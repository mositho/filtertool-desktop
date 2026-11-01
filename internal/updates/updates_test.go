package updates

import (
	"context"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

func TestMatchExecutable(t *testing.T) {
	assets := []github.ReleaseAsset{{Name: "Filtertool-Setup.exe"}, {Name: "SHA256SUMS.txt"}, {Name: "Filtertool.exe"}}
	if got := matchExecutable(updater.CheckRequest{Platform: "windows", Arch: "amd64"}, assets); got != 2 {
		t.Fatalf("selected asset %d, want executable at 2", got)
	}
	if got := matchExecutable(updater.CheckRequest{Platform: "windows", Arch: "arm64"}, assets); got != -1 {
		t.Fatalf("selected unsupported architecture: %d", got)
	}
	if got := matchExecutable(updater.CheckRequest{Platform: "windows", Arch: "amd64"}, assets[:2]); got != -1 {
		t.Fatalf("selected installer instead of missing executable: %d", got)
	}
}

func TestMajorUpgrade(t *testing.T) {
	for _, tc := range []struct {
		current, next string
		want          bool
	}{{"1.0.0", "1.1.0", false}, {"1.0.0", "1.0.1", false}, {"1.9.9", "2.0.0", true}, {"1.9.9", "3.1.0", true}, {"v1.0.0", "v2.0.0", true}} {
		if got := majorUpgrade(tc.current, tc.next); got != tc.want {
			t.Errorf("%s → %s: got %v, want %v", tc.current, tc.next, got, tc.want)
		}
	}
}

type releaseProvider struct {
	updater.Provider
	release *updater.Release
}

func (p releaseProvider) Check(context.Context, updater.CheckRequest) (*updater.Release, error) {
	return p.release, nil
}

func TestRequireChecksum(t *testing.T) {
	for _, verification := range []*updater.Verification{nil, {DigestAlgo: "sha256"}, {DigestAlgo: "sha256", Digest: make([]byte, 32)}} {
		p := verifiedProvider{releaseProvider{release: &updater.Release{Verification: verification}}}
		_, err := p.Check(context.Background(), updater.CheckRequest{})
		valid := verification != nil && len(verification.Digest) == 32
		if (err == nil) != valid {
			t.Errorf("verification %+v: unexpected error %v", verification, err)
		}
	}
}

func TestMajorDownloadRequiresConsent(t *testing.T) {
	m := &Manager{status: Status{Enabled: true, Release: &Release{Version: "2.0.0", Major: true}}}
	// A nil updater would panic if the request bypassed the consent/version guards.
	if _, err := m.Download("2.0.0", false); err == nil {
		t.Fatal("major update accepted without consent")
	}
	if _, err := m.Download("3.0.0", true); err == nil {
		t.Fatal("stale release accepted")
	}
}
