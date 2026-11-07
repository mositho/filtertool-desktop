package compiler

import (
	"strings"
	"testing"
)

func TestShieldSoundOnlyDuringEarlyProgression(t *testing.T) {
	c := New(nil, nil, "test-sounds")
	if c.manifest["shield"] != "Shield" {
		t.Fatal("shield TTS template must say Shield")
	}
	for _, mode := range []string{"none", "early", "full"} {
		t.Run(mode, func(t *testing.T) {
			output := c.early(Object{
				"shieldProgression": mode,
				"preferredArmour":   []any{"armour"},
			}, Object{"earlyMaxAreaLevel": float64(17)})
			early, late := 0, 0
			for _, block := range strings.Split(output, "\n\n") {
				if !strings.Contains(block, `Class "Shields"`) {
					continue
				}
				sound := strings.Contains(block, `CustomAlertSound "test-sounds/shield.mp3" 300`)
				if strings.Contains(block, "AreaLevel <= 17") {
					early++
					if !sound {
						t.Fatal("early shield missing sound", block)
					}
				} else {
					late++
					if !strings.Contains(block, "AreaLevel > 17") || sound {
						t.Fatal("later shield must preserve cutoff and stay silent", block)
					}
				}
			}
			wantEarly, wantLate := 3, 0
			if mode == "none" {
				wantEarly = 0
			}
			if mode == "full" {
				wantLate = 1
			}
			if early != wantEarly || late != wantLate {
				t.Fatalf("got %d early and %d late shield rules; want %d and %d", early, late, wantEarly, wantLate)
			}
		})
	}
}
