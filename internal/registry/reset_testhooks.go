//go:build testhooks

package registry

func ResetForTest() { resetLocked() }
