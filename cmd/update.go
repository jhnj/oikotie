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

		search := scraper.Create(di.db).SetAreaCodes(di.cfg.SearchConfig().Areas)
		if p := di.cfg.SearchConfig().Price; p != nil {
			search.SetPrice(p.Min, p.Max)
		}
		if s := di.cfg.SearchConfig().Size; s != nil {
			search.SetSize(s.Min, s.Max)
		}

		_, err := search.Run()
		if err != nil {
			log.Fatal(err)
		}
	},
}
