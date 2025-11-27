package inspect

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit/v1/core/reflection"
	"github.com/spf13/cobra"
)

var jsonFlag string

var StructCmd = &cobra.Command{
	Use:   "struct",
	Short: "Inspect a struct and see the generated Mangle facts.",
	Long:  `Parses a JSON string or file representing a struct and uses the core reflection engine to generate Mangle Datalog facts. This helps users understand how Manglekit perceives their data structures.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var jsonData []byte
		var err error

		// Check if the input is a file path or a raw JSON string
		if _, err := os.Stat(jsonFlag); err == nil {
			jsonData, err = os.ReadFile(jsonFlag)
			if err != nil {
				return fmt.Errorf("failed to read JSON file: %w", err)
			}
		} else {
			jsonData = []byte(jsonFlag)
		}

		var data map[string]any
		if err := json.Unmarshal(jsonData, &data); err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		facts, err := reflection.ToFacts("request", data)
		if err != nil {
			return fmt.Errorf("failed to generate facts: %w", err)
		}

		for _, fact := range facts {
			fmt.Println(fact.String())
		}

		return nil
	},
}

func init() {
	StructCmd.Flags().StringVar(&jsonFlag, "json", "", "JSON string or file path representing a struct")
	StructCmd.MarkFlagRequired("json")
	InspectCmd.AddCommand(StructCmd)
}
