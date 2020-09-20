package main

import "fmt"
import "oikotie/scraper"
import "database/sql"
import "log"

func main() {
	db, err := sql.Open("postgres", "postgres://johan:password@localhost:5432/oikotie?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
	search := scraper.CreateSearch(db).SetAreaCodes([]string{"00200", "00340"})
	res, err := search.Run()
    if err != nil {
        log.Fatal(err)
    }
	fmt.Println(res)
}
