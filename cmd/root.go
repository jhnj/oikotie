package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"oikotie/config"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ot",
	Short: "Oikotie scraper",
	// Run: func(cmd *cobra.Command, args []string) {
	// 	// Do Stuff Here
	// },
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type DI struct {
	cfg *config.Reader
	db  *sql.DB
}

func setup() DI {
	cfg, err := config.NewReader()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL())
	if err != nil {
		log.Fatal(err)
	}

	return DI{
		cfg: cfg,
		db:  db,
	}
}

// func init() {
// 	rootCmd.AddCommand(updateCmd)
// }
