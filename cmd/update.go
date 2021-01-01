package cmd

import (
	"log"
	"oikotie/scraper"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Scrape Oikotie and update data",
	Run: func(cmd *cobra.Command, args []string) {
		di := setup()

		search := scraper.Create(di.db).SetAreaCodes([]string{"00200", "00340"})
		_, err := search.Run()
		if err != nil {
			log.Fatal(err)
		}
	},
}
