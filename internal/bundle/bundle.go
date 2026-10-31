package bundle

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

//go:embed game-sounds/*
var files embed.FS

func Prepare(root string) (func([]byte, string, float64) error, error) {
	exe, err := exec.LookPath("ffmpeg")
	if err != nil {
		exe = ""
	}
	game := filepath.Join(root, "cache", "game-sounds")
	if err = os.MkdirAll(game, 0700); err != nil {
		return nil, err
	}
	entries, err := files.ReadDir("game-sounds")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		b, err := files.ReadFile("game-sounds/" + e.Name())
		if err != nil {
			return nil, err
		}
		if err = os.WriteFile(filepath.Join(game, e.Name()), b, 0600); err != nil {
			return nil, err
		}
	}
	if exe == "" {
		return nil, nil
	}
	return func(input []byte, target string, speed float64) error {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, exe, "-y", "-hide_banner", "-loglevel", "error", "-i", "pipe:0", "-acodec", "libmp3lame", "-af", fmt.Sprintf("atempo=%g,treble=g=6,apad=pad_len=22050,loudnorm=I=-14:TP=-2:LRA=7,alimiter=limit=0.794328:level=false", speed), "-f", "mp3", target)
		cmd.Stdin = bytes.NewReader(input)
		hideWindow(cmd)
		output, err := cmd.CombinedOutput()
		if err != nil {
			if len(output) > 4096 {
				output = output[len(output)-4096:]
			}
			return fmt.Errorf("FFmpeg: %w: %s", err, output)
		}
		return nil
	}, nil
}
