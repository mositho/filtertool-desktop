package desktop

import (
	"errors"
	"testing"
)

func TestDialogSelection(t *testing.T) {
	path, err := DialogSelection("ignored", errors.New("cancelled by user"))
	if path != "" || err != nil {
		t.Fatalf("cancellation should be empty and successful: %q, %v", path, err)
	}
	paths, err := DialogSelection([]string{"ignored"}, errors.New("cancelled by user"))
	if paths != nil || err != nil {
		t.Fatalf("multiple-selection cancellation should be empty and successful: %v, %v", paths, err)
	}
	failure := errors.New("dialog failed")
	if _, err := DialogSelection("", failure); !errors.Is(err, failure) {
		t.Fatalf("real errors must be preserved: %v", err)
	}
	if path, err := DialogSelection("chosen", nil); path != "chosen" || err != nil {
		t.Fatalf("selection must be preserved: %q, %v", path, err)
	}
}
