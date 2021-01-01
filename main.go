package main

import (
	"database/sql"
	"log"
	"oikotie/config"
	"oikotie/scraper"

	"oikotie/cmd"
)

func main() {
	cmd.Execute()
	return
	cfg, err := config.NewReader()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL())
	if err != nil {
		log.Fatal(err)
	}

	// search := scraper.Create(db).SetAreaCodes([]string{"00200", "00340"})
	// res, err := search.Run()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(res)

	err = scraper.UpdateListing(db, 4)
	if err != nil {
		log.Fatal(err)
	}
}
