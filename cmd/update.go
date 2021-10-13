package cmd

import (
	"log"
	"oikotie/scraper"
	"oikotie/tg"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Scrape Oikotie and update data",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Running Oikotie update")
		di := setup()

		search := scraper.Create(di.db).SetAreaCodes(di.cfg.SearchConfig().Areas)
		if p := di.cfg.SearchConfig().Price; p != nil {
			search.SetPrice(p.Min, p.Max)
		}
		if s := di.cfg.SearchConfig().Size; s != nil {
			search.SetSize(s.Min, s.Max)
		}

		l, err := search.Run()
		if err != nil {
			_ = tg.ReportError(err)
			log.Fatal(err)
		} else {
			log.Printf("Update successfull, created %d listings\n", len(l))
		}
	},
}
