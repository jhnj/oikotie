package scraper

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"oikotie/database/models"
	"regexp"
	"strconv"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/lib/pq"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/types/pgeo"
)

const cardsURL = "https://asunnot.oikotie.fi/api/cards"

var acceptedCardIDs = map[int]interface{}{
	4: struct{}{}, // kaupunginosa
	5: struct{}{}, // postinumeroalue
}

type requestParams struct {
	token   string
	loaded  string
	cuid    string
	cookies []*http.Cookie
}

type apiArea struct {
	Card struct {
		Name     string `json:"name"`
		CardID   int    `json:"cardId"`
		CardType int    `json:"cardType"`
	} `json:"card"`
	Parent struct {
		Name string `json:"name"`
	} `json:"parent"`
}

type scraperOptions struct {
	MaxPrice  int
	MinPrice  int
	MaxSize   int
	MinSize   int
	AreaCodes []string
}

type Scraper struct {
	options       scraperOptions
	db            *sql.DB
	requestParams *requestParams
	client        *http.Client
}

// Create Initialize with default values
func Create(db *sql.DB) *Scraper {
	search := &Scraper{
		options: scraperOptions{
			MaxPrice:  800000,
			MinPrice:  1,
			MaxSize:   100,
			MinSize:   1,
			AreaCodes: []string{"00200"},
		},
		db:     db,
		client: &http.Client{},
	}

	return search
}

func (s *Scraper) SetAreaCodes(areaCodes []string) *Scraper {
	s.options.AreaCodes = areaCodes
	return s
}

func (s *Scraper) SetPrice(min int, max int) *Scraper {
	s.options.MinPrice = min
	s.options.MaxPrice = max
	return s
}

func (s *Scraper) SetSize(min int, max int) *Scraper {
	s.options.MinSize = min
	s.options.MaxSize = max
	return s
}

func (s *Scraper) Run() ([]*models.Listing, error) {
	params, err := getRequestParams()
	if err != nil {
		return nil, err
	}
	s.requestParams = &params

	areas, err := s.getAreas(s.options.AreaCodes)
	if err != nil {
		return nil, err
	}

	l := []*models.Listing{}
	for _, area := range areas {
		nl, err := s.getListings(area)
		l = append(l, nl...)
		if err != nil {
			return nil, err
		}
	}

	return l, nil
}

func (s *Scraper) getAreas(areaCodes []string) ([]*models.Area, error) {
	areasInDB, err := models.Areas(models.AreaWhere.Name.IN(areaCodes)).All(s.db)
	if err != nil {
		return nil, err
	}

	for _, areaCode := range areaCodes {
		inDB := false
		for _, area := range areasInDB {
			if area.Name == areaCode {
				inDB = true
				break
			}
		}

		if !inDB {
			area, err := s.getArea(areaCode)
			if err != nil {
				return nil, err
			}

			exists, err := models.Areas(models.AreaWhere.ExternalID.EQ(area.Card.CardID)).Exists(s.db)
			if err != nil {
				return nil, err
			}

			if exists {
				continue
			}

			dbArea := models.Area{
				ExternalID: area.Card.CardID,
				Name:       area.Card.Name,
				CardType:   area.Card.CardType,
				City:       area.Parent.Name,
			}

			err = dbArea.Insert(s.db, boil.Infer())
			if err != nil {
				return nil, err
			}
		}
	}

	return models.Areas(models.AreaWhere.Name.IN(areaCodes)).All(s.db)
}

func (s *Scraper) apiCall(endpoint string) *http.Request {
	url := "https://asunnot.oikotie.fi/api/3.0/" + endpoint
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("ota-token", s.requestParams.token)
	req.Header.Set("ota-cuid", s.requestParams.cuid)
	req.Header.Set("ota-loaded", s.requestParams.loaded)
	for _, cookie := range s.requestParams.cookies {
		req.AddCookie(cookie)
	}

	return req
}

func (s *Scraper) getArea(areaCode string) (apiArea, error) {
	client := &http.Client{}

	req := s.apiCall("location")

	q := req.URL.Query()
	q.Add("query", areaCode)

	req.URL.RawQuery = q.Encode()

	resp, _ := client.Do(req)

	var allMatching []apiArea
	err := json.NewDecoder(resp.Body).Decode(&allMatching)
	if err != nil {
		return apiArea{}, err
	}

	cityAreas := make([]apiArea, 0)
	for _, area := range allMatching {
		if _, ok := acceptedCardIDs[area.Card.CardType]; ok {
			cityAreas = append(cityAreas, area)
		}
	}

	if len(cityAreas) != 1 {
		return apiArea{}, fmt.Errorf("Expected 1 card for code '%s', got %d", areaCode, len(cityAreas))
	}

	return allMatching[0], nil
}

func (s *Scraper) getListings(area *models.Area) ([]*models.Listing, error) {
	req, _ := http.NewRequest("GET", cardsURL, nil)
	req.Header.Set("ota-token", s.requestParams.token)
	req.Header.Set("ota-cuid", s.requestParams.cuid)
	req.Header.Set("ota-loaded", s.requestParams.loaded)
	for _, cookie := range s.requestParams.cookies {
		req.AddCookie(cookie)
	}

	q := req.URL.Query()
	q.Add("buildingType[]", "1") // Kerrostalo
	q.Add("buildingType[]", "256")
	q.Add("cardType", "100")      // Not sure
	q.Add("conditionType[]", "1") // Erinomainen, Hyvä, Tyydyttävä, Välttävä, Huono
	q.Add("conditionType[]", "2")
	q.Add("conditionType[]", "4")
	q.Add("conditionType[]", "8")
	q.Add("conditionType[]", "16")
	// areaStrings := make([]string, len(areas))
	// for i, a := range areas {
	// 	areaStrings[i] = fmt.Sprintf("[%d,%d,\"%s, Helsinki\"]", a.AreaID, a.CardType, a.Name)
	// }
	// q.Add("locations", fmt.Sprintf("[%s]", strings.Join(areaStrings, ",")))
	q.Add("locations", fmt.Sprintf("[[%d, %d,\"%s, %s\"]]", area.ExternalID, area.CardType, area.Name, area.City))

	q.Add("lotOwnershipType[]", "1") // Oma, Vuokralla, Valinnainen vuokratontti
	q.Add("lotOwnershipType[]", "2")
	q.Add("lotOwnershipType[]", "3")
	q.Add("price[max]", strconv.Itoa(s.options.MaxPrice))
	q.Add("price[min]", strconv.Itoa(s.options.MinPrice))
	q.Add("size[max]", strconv.Itoa(s.options.MaxSize))
	q.Add("size[min]", strconv.Itoa(s.options.MinSize))
	q.Add("sortBy", "published_sort_desc")
	// q.Add("limit", "1")
	req.URL.RawQuery = q.Encode()

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	listings := []*models.Listing{}

	var listingsResponse struct {
		Cards []map[string]interface{} `json:"cards"`
	}
	err = json.NewDecoder(resp.Body).Decode(&listingsResponse)
	if err != nil {
		return nil, err
	}

	for _, apiListing := range listingsResponse.Cards {
		listing := &models.Listing{}

		err = listing.ListingData.Marshal(apiListing)
		if err != nil {
			return nil, err
		}

		err = listing.SetArea(s.db, false, area)
		if err != nil {
			return nil, err
		}

		listingDetails, err := getListingDetails(int(apiListing["id"].(float64)), area)
		if err != nil {
			return nil, err
		}

		err = listing.ListingDetails.Marshal(listingDetails)
		if err != nil {
			return nil, err
		}

		err = SetDerivedFields(listing)
		if err != nil {
			log.Printf("Failed to set derived fields, err: [%s], listing id: %d", err.Error(), listing.ExternalID)
		}

		err = listing.Insert(s.db, boil.Infer())
		if err != nil {
			return nil, err
		}

		listings = append(listings, listing)
	}

	return listings, nil
}

func getRequestParams() (requestParams, error) {
	var params requestParams

	resp, err := http.Get("https://asunnot.oikotie.fi/myytavat-asunnot")
	if err != nil {
		return params, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return params, fmt.Errorf("Request failed %d %s", resp.StatusCode, resp.Status)
	}

	params.cookies = resp.Cookies()

	err = parseMetaAttrs(&params, resp.Body)
	if err != nil {
		return params, err
	}

	return params, nil
}

func parseMetaAttrs(params *requestParams, body io.Reader) error {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return err
	}

	res := make(map[string]string)

	attrs := []string{"api-token", "loaded", "cuid"}
	for _, attr := range attrs {
		var value string
		doc.Find(fmt.Sprintf("meta[name=%s]", attr)).Each(func(i int, s *goquery.Selection) {
			value, _ = s.Attr("content")
		})

		if value == "" {
			return fmt.Errorf("Attr %s not found", attr)
		}

		res[attr] = value
	}

	params.cuid = res["cuid"]
	params.loaded = res["loaded"]
	params.token = res["api-token"]

	return nil
}

type listingDetails = map[string]map[string]string

func getListingDetails(externalID int, area *models.Area) (listingDetails, error) {
	resp, err := http.Get(fmt.Sprintf("https://asunnot.oikotie.fi/myytavat-asunnot/%s/%d", area.City, externalID))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Request failed %d %s, external id: %d", resp.StatusCode, resp.Status, externalID)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	details := make(listingDetails)

	doc.Find(".listing-details-container").Find(".listing-details").Each(func(i int, s *goquery.Selection) {
		h := s.Find("h3").First().Text()
		if h == "" {
			return
		}

		cat := make(map[string]string)

		s.Find(".info-table__row").Each(func(i int, s *goquery.Selection) {
			k := s.Find("dt").First().Text()
			v := s.Find("dd").First().Text()
			if k != "" && v != "" {
				cat[k] = v
			}
		})

		details[h] = cat
	})

	return details, nil
}

var onlyNumbers = regexp.MustCompile("[^0-9]+")
var floorReg = regexp.MustCompile("^[0-9]+")

func SetDerivedFields(listing *models.Listing) error {
	var data map[string]interface{}
	err := listing.ListingData.Unmarshal(&data)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("data empty")
	}

	var details listingDetails
	err = listing.ListingDetails.Unmarshal(&details)
	if err != nil {
		return err
	}
	if len(details) == 0 {
		return fmt.Errorf("details empty")
	}

	externalID, ok := data["id"].(float64)
	if !ok {
		return errors.New("Cast failed: ExternalID")
	}
	listing.ExternalID = int(externalID)

	price, err := strconv.Atoi(onlyNumbers.ReplaceAllString(data["price"].(string), ""))
	if err != nil {
		return fmt.Errorf("price, %w", err)
	}
	listing.Price = price

	size, ok := data["size"].(float64)
	if !ok {
		return errors.New("Cast failed: Size")
	}
	listing.Size = size

	rooms, ok := data["rooms"].(float64)
	if !ok {
		return errors.New("Cast failed: Rooms")
	}
	listing.Rooms = int(rooms)

	visits, ok := data["visits"].(float64)
	if !ok {
		return errors.New("Cast failed: Visits")
	}
	listing.Visits = int(visits)

	floor, err := strconv.Atoi(floorReg.FindString(details["Perustiedot"]["Kerros"]))
	if err != nil {
		return fmt.Errorf("floor, %w", err)
	}
	listing.Floor = floor

	lat, err := getCoordinates(data, "latitude")
	if err != nil {
		return err
	}
	long, err := getCoordinates(data, "longitude")
	if err != nil {
		return err
	}
	listing.Coord.Point = pgeo.NewPoint(lat, long)
	listing.Coord.Valid = true

	return nil
}

func UpdateListing(db *sql.DB, id int) error {
	listing, err := models.FindListing(db, id)
	if err != nil {
		return err
	}

	err = SetDerivedFields(listing)
	if err != nil {
		return err
	}

	_, err = listing.Update(db, boil.Infer())
	return err
}

func getCoordinates(data map[string]interface{}, latLong string) (float64, error) {
	coords, ok := data["coordinates"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("Cast failed: Coordinates")
	}
	value, ok := coords[latLong].(float64)
	if !ok {
		return 0, fmt.Errorf("Cast failed: %s", latLong)
	}

	return value, nil
}
