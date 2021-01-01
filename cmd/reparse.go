package cmd

import (
	"log"
	"oikotie/database/models"
	"oikotie/scraper"

	"github.com/spf13/cobra"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

func init() {
	rootCmd.AddCommand(reparseCmd)
}

var reparseCmd = &cobra.Command{
	Use:   "reparse",
	Short: "Reparse raw data",
	Run: func(cmd *cobra.Command, args []string) {
		di := setup()

		listings, err := models.Listings().All(di.db)
		if err != nil {
			log.Fatal(err)
		}

		for _, listing := range listings {
			err = scraper.SetDerivedFields(listing)
			if err != nil {
				log.Printf("Error parsing listing (%d), skipping. %v", listing.ID, err)
			}

			_, err = listing.Update(di.db, boil.Infer())
			if err != nil {
				log.Fatal(err)
			}
		}

		log.Printf("Reparsed %d listings", len(listings))
	},
}
