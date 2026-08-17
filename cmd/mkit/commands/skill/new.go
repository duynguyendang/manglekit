package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"github.com/spf13/cobra"
)

var (
	flagDir   string
	flagForce bool
)

var newCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Scaffold a new policy-gated skill (main.go + policy.dl + contract test)",
	Long: `Scaffold a new Manglekit skill using only the blessed APIs
(sdk.NewClient + sdk.WithPolicyPath + sdk.Define).

Emits into <dir>/<name>/:
  - main.go       typed skill wired through the zero-trust supervisor
  - policy.dl     starter Datalog policy with a sample halt rule
  - main_test.go  contract test (allowed path + fail-closed block)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNew(args[0])
	},
}

func init() {
	newCmd.Flags().StringVar(&flagDir, "dir", ".", "base directory to create the skill in")
	newCmd.Flags().BoolVar(&flagForce, "force", false, "overwrite existing files")
}

type skillData struct {
	Name   string
	Pascal string
	Snake  string
}

func runNew(name string) error {
	snake := toSnake(name)
	if snake == "" {
		return fmt.Errorf("invalid skill name %q: must contain at least one letter or digit", name)
	}
	data := skillData{Name: name, Pascal: toPascal(name), Snake: snake}

	dir := filepath.Join(flagDir, snake)
	if _, err := os.Stat(dir); err == nil && !flagForce {
		return fmt.Errorf("directory %s already exists (use --force to overwrite)", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	files := map[string]string{
		"main.go":      "templates/main.go.tmpl",
		"main_test.go": "templates/main_test.go.tmpl",
		"policy.dl":    "templates/policy.dl.tmpl",
	}
	for outName, tmplName := range files {
		if err := renderFile(dir, outName, tmplName, data); err != nil {
			return err
		}
	}

	fmt.Printf("Skill %q scaffolded at %s\n", snake, dir)
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", dir)
	fmt.Println("  go mod init <module> && go mod tidy")
	fmt.Println("  go test ./...   # contract test: allowed + fail-closed block")
	fmt.Println("  go run .")
	return nil
}

func renderFile(dir, outName, tmplName string, data skillData) error {
	raw, err := templateFS.ReadFile(tmplName)
	if err != nil {
		return fmt.Errorf("read template %s: %w", tmplName, err)
	}
	tmpl, err := template.New(outName).Parse(string(raw))
	if err != nil {
		return fmt.Errorf("parse template %s: %w", tmplName, err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template %s: %w", tmplName, err)
	}
	outPath := filepath.Join(dir, outName)
	if err := os.WriteFile(outPath, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	return nil
}

// toSnake converts an arbitrary name to a lower_snake_case identifier.
// Runs of non-identifier characters collapse into a single underscore.
func toSnake(name string) string {
	var b strings.Builder
	pendingSep := true // suppresses leading/duplicate underscores
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			pendingSep = false
			continue
		}
		if !pendingSep {
			b.WriteRune('_')
			pendingSep = true
		}
	}
	return strings.Trim(b.String(), "_")
}

// toPascal converts an arbitrary name to a PascalCase identifier.
func toPascal(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == ' ' || r == '_' || !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	return b.String()
}
