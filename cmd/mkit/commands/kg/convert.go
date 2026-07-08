package kg

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cayleygraph/quad"
	"github.com/cayleygraph/quad/jsonld"
	"github.com/cayleygraph/quad/nquads"
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
				// Turtle is not supported by the underlying quad reader.
				return fmt.Errorf("turtle (.ttl) is not supported; convert it to .nq/.nt first")
			case ".nq":
				format = "nquads"
			case ".nt":
				format = "ntriples"
			case ".jsonld":
				format = "jsonld"
			default:
				return fmt.Errorf("unknown input extension %s, please specify --from", ext)
			}
		}

		var r quad.Reader
		switch format {
		case "nquads", "ntriples":
			// nquads.NewReader takes (io.Reader, bool) for strict mode.
			r = nquads.NewReader(f, false)
		case "jsonld":
			r = jsonld.NewReader(f)
		default:
			return fmt.Errorf("unsupported input format: %s", format)
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
		n, err := quad.Copy(w, r)
		if err != nil {
			return fmt.Errorf("conversion failed: %w", err)
		}

		// 6. UX
		cmd.Printf("Converted %d quads to %s\n", n, outFile)
		return nil
	},
}

func init() {
	ConvertCmd.Flags().StringVarP(&inFile, "input", "i", "", "Input file path")
	ConvertCmd.MarkFlagRequired("input")

	ConvertCmd.Flags().StringVarP(&outFile, "output", "o", "", "Output file path")
	ConvertCmd.MarkFlagRequired("output")

	ConvertCmd.Flags().StringVar(&inFmt, "from", "", "Input format (auto-detect if empty)")
	ConvertCmd.Flags().StringVar(&outFmt, "to", "nquads", "Output format (options: \"nquads\", \"ntriples\", \"jsonld\")")
}
