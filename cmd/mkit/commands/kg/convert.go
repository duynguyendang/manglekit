package kg

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cayleygraph/quad"
	"github.com/cayleygraph/quad/jsonld"
	"github.com/cayleygraph/quad/nquads"
	"github.com/duynguyendang/manglekit/adapters/knowledge"
	"github.com/spf13/cobra"
)

var (
	inFile  string
	outFile string
	inFmt   string
	outFmt  string
)

var ConvertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert between knowledge graph formats (e.g., TTL to NQ)",
	Long: `Convert between knowledge graph formats.

Supported inputs: .ttl (Turtle, via the knowledge-rdf parser), .nq
(N-Quads), .nt (N-Triples), .jsonld (JSON-LD). Supported outputs:
nquads (default), ntriples, jsonld.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Open Input
		f, err := os.Open(inFile)
		if err != nil {
			return fmt.Errorf("failed to open input file: %w", err)
		}
		defer f.Close()

		// 2. Decoder
		format := inFmt
		if format == "" {
			ext := filepath.Ext(inFile)
			switch ext {
			case ".ttl":
				// Turtle is handled by the knowledge-rdf parser below.
				format = "turtle"
			case ".nq":
				format = "nquads"
			case ".nt":
				format = "ntriples"
			case ".jsonld":
				format = "jsonld"
			default:
				return fmt.Errorf("unknown input extension %s, please specify --from (supported: .ttl, .nq, .nt, .jsonld)", ext)
			}
		}

		// 3. Open Output
		outF, err := os.Create(outFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer outF.Close()

		// 4. Encoder
		var w quad.Writer
		switch outFmt {
		case "nquads", "ntriples":
			// nquads writer is used for both
			w = nquads.NewWriter(outF)
		case "jsonld":
			w = jsonld.NewWriter(outF)
		default:
			return fmt.Errorf("unsupported output format: %s", outFmt)
		}

		// Close writer if it needs closing
		defer func() {
			if c, ok := w.(io.Closer); ok {
				c.Close()
			}
		}()

		// 5. Copy
		var n int
		if format == "turtle" || format == "ttl" {
			n, err = copyTurtle(w)
		} else {
			var r quad.Reader
			switch format {
			case "nquads", "ntriples":
				// nquads.NewReader takes (io.Reader, bool) for strict mode.
				r = nquads.NewReader(f, false)
			case "jsonld":
				r = jsonld.NewReader(f)
			default:
				return fmt.Errorf("unsupported input format: %s (supported: turtle, nquads, ntriples, jsonld)", format)
			}
			n, err = quad.Copy(w, r)
		}
		if err != nil {
			return fmt.Errorf("conversion failed: %w", err)
		}

		// 6. UX
		cmd.Printf("Converted %d quads to %s\n", n, outFile)
		return nil
	},
}

// copyTurtle parses a Turtle file through the knowledge-rdf parser (the
// same path `mkit eval --knowledge` uses) and writes the triples as quads.
func copyTurtle(w quad.Writer) (int, error) {
	triples, err := knowledge.ParseGraphFile(inFile)
	if err != nil {
		return 0, fmt.Errorf("failed to parse turtle input: %w", err)
	}
	n := 0
	for _, t := range triples {
		if err := w.WriteQuad(quad.Quad{
			Subject:   toIRI(t.Subject),
			Predicate: toIRI(t.Predicate),
			Object:    toTerm(t.Object),
		}); err != nil {
			return n, fmt.Errorf("failed to write quad: %w", err)
		}
		n++
	}
	return n, nil
}

// toIRI converts a raw term to an IRI, stripping <> delimiters if present.
func toIRI(v string) quad.IRI {
	return quad.IRI(strings.Trim(strings.TrimSpace(v), "<>"))
}

// toTerm converts a raw object term to an IRI or typed string, mirroring
// the knowledge adapter's smart-casting (datatype suffixes stripped).
func toTerm(v string) quad.Value {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "<") && strings.HasSuffix(v, ">") {
		return quad.IRI(strings.Trim(v, "<>"))
	}
	if idx := strings.Index(v, "^^"); idx != -1 {
		v = v[:idx]
	}
	return quad.String(strings.Trim(strings.TrimSpace(v), `"`))
}

func init() {
	ConvertCmd.Flags().StringVarP(&inFile, "input", "i", "", "Input file path (.ttl, .nq, .nt, .jsonld)")
	ConvertCmd.MarkFlagRequired("input")

	ConvertCmd.Flags().StringVarP(&outFile, "output", "o", "", "Output file path")
	ConvertCmd.MarkFlagRequired("output")

	ConvertCmd.Flags().StringVar(&inFmt, "from", "", `Input format (auto-detect if empty; options: "turtle", "nquads", "ntriples", "jsonld")`)
	ConvertCmd.Flags().StringVar(&outFmt, "to", "nquads", `Output format (options: "nquads", "ntriples", "jsonld")`)
}
