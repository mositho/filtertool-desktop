package bundle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareWorksWithoutFFmpeg(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	root := t.TempDir()
	processor, err := Prepare(root)
	if err != nil {
		t.Fatal(err)
	}
	if processor != nil {
		t.Fatal("audio processor should be unavailable when FFmpeg is not on PATH")
	}
	if _, err = os.Stat(filepath.Join(root, "cache", "game-sounds", "AlertSound1.mp3")); err != nil {
		t.Fatal("built-in game sounds were not prepared", err)
	}
}
