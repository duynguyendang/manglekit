package sdk

import "fmt"

// Must ensures that the client initialization succeeded.
// If err is not nil, it panics. This is useful for concise initialization in main() functions.
//
// Parameters:
//   - c: The client instance.
//   - err: The error returned by the constructor.
//
// Returns:
//   - The valid Client instance.
func Must(c *Client, err error) *Client {
	if err != nil {
		panic(fmt.Sprintf("Manglekit initialization failed: %v", err))
	}
	return c
}

// TryClient attempts client initialization and returns a non-panicking error.
// Prefer this over Must() in library code or tests where panics are inappropriate.
//
// Returns:
//   - The valid Client instance, or a zero-value Client and the initialization error.
func TryClient(c *Client, err error) (*Client, error) {
	return c, err
}
