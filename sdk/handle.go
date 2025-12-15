package sdk

// TypedAction serves as a strongly-typed handle for a registered action.
// It creates a contract between the caller and the engine without code generation.
//
// TIn:  The Go struct type required for Input.
// TOut: The Go struct type guaranteed for Output.
type TypedAction[TIn any, TOut any] struct {
	Name string
}

// DefineAction creates a new typed handle.
// This should be used in a shared "definitions" package or at the top of main.
//
// Example:
//
//	var ActionTransfer = sdk.DefineAction[TransferReq, TransferResp]("transfer_money")
func DefineAction[TIn any, TOut any](name string) TypedAction[TIn, TOut] {
	return TypedAction[TIn, TOut]{Name: name}
}
