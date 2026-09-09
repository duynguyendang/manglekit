package genes

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/sdk"
)

// ApplyTo compiles the given genes and loads them into the client's policy
// program via the official source channel (Client.LoadPolicy — additive,
// auto-merges std.dl Decls where needed). Nothing is applied unless the
// whole compiled program passes every check; on error the client's policy
// is unchanged for genes that failed (load happens after full compilation).
//
// Note: compiled genes belong to the USER program — a later
// ReloadPolicy/ReloadPolicySource replaces them (see the runtime's persistent
// units for engine builtins). Persist the gene manifest as the source of
// truth and re-apply after reloads, or bake the genes into the policy file.
func ApplyTo(ctx context.Context, client *sdk.Client, genes []Gene, opts ...CompileOption) error {
	src, err := Compile(genes, opts...)
	if err != nil {
		return err
	}
	if client == nil {
		return fmt.Errorf("x/genes: nil client")
	}
	if err := client.LoadPolicy(ctx, src); err != nil {
		return fmt.Errorf("x/genes: compiled genes failed to load as policy: %w", err)
	}
	return nil
}
