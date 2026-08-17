package version

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModuleVersionNonEmpty(t *testing.T) {
	v := ModuleVersion()
	assert.NotEmpty(t, v)
}

func TestVersionCmdPrints(t *testing.T) {
	var out bytes.Buffer
	VersionCmd.SetOut(&out)
	VersionCmd.SetErr(&out)
	VersionCmd.SetArgs([]string{})
	assert.NoError(t, VersionCmd.Execute())
	assert.Contains(t, out.String(), "mkit ")
}
