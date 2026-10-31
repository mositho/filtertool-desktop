package desktop

// DialogSelection treats native dialog cancellation as an empty selection.
// Wails' Windows dialog cancellation error is internal, so it cannot be
// matched by identity here. Normalize only that exact error at the boundary.
func DialogSelection[T any](selection T, err error) (T, error) {
	if err != nil && err.Error() == "cancelled by user" {
		var empty T
		return empty, nil
	}
	return selection, err
}
