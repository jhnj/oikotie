# Oikotie scraper

## Commands
- Run migrations
  `migrate -database "postgres://johan:password@localhost:5432/oikotie?sslmode=disable" -path migrations up`
- Connect to DB 
  `psql postgres://johan:password@localhost:5432/oikotie`
- Generate SqlBoiler
  `sqlboiler psql`
