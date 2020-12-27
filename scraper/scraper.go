package scraper

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"oikotie/database/models"
	"regexp"
	"strconv"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/lib/pq"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

type requestParams struct {
	token   string
	loaded  string
	cuid    string
	cookies []*http.Cookie
}

type ApiArea struct {
	Card struct {
		Name     string `json:"name"`
		CardID   int    `json:"cardId"`
		CardType int    `json:"cardType"`
	} `json:"card"`
	Parent struct {
		Name string `json:"name"`
	} `json:"parent"`
}

type searchOptions struct {
	MaxPrice  int
	MinPrice  int
	MaxSize   int
	MinSize   int
	AreaCodes []string
}

type Search struct {
	Options       searchOptions
	DB            *sql.DB
	requestParams *requestParams
}

// CreateSearch Initialize with default values
func CreateSearch(db *sql.DB) *Search {
	search := &Search{
		Options: searchOptions{
			MaxPrice:  800000,
			MinPrice:  1,
			MaxSize:   100,
			MinSize:   1,
			AreaCodes: []string{"00200"},
		},
		DB: db,
	}

	return search
}

func (s *Search) SetAreaCodes(areaCodes []string) *Search {
	s.Options.AreaCodes = areaCodes
	return s
}

func (s *Search) SetPrice(min int, max int) *Search {
	s.Options.MinPrice = min
	s.Options.MaxPrice = max
	return s
}

func (s *Search) SetSize(min int, max int) *Search {
	s.Options.MinSize = min
	s.Options.MaxSize = max
	return s
}

func (s *Search) Run() (interface{}, error) {
	params, err := getRequestParams()
	if err != nil {
		return "", err
	}
	s.requestParams = &params

	areas, err := s.getAreas(s.Options.AreaCodes)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	url := "https://asunnot.oikotie.fi/api/cards"

	for _, area := range areas {
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("ota-token", params.token)
		req.Header.Set("ota-cuid", params.cuid)
		req.Header.Set("ota-loaded", params.loaded)
		for _, cookie := range params.cookies {
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
		q.Add("locations", fmt.Sprintf("[[%d, %d,\"%s, %s\"]]", area.ID, area.CardType, area.Name, area.City))

		q.Add("lotOwnershipType[]", "1") // Oma, Vuokralla, Valinnainen vuokratontti
		q.Add("lotOwnershipType[]", "2")
		q.Add("lotOwnershipType[]", "3")
		q.Add("price[max]", strconv.Itoa(s.Options.MaxPrice))
		q.Add("price[min]", strconv.Itoa(s.Options.MinPrice))
		q.Add("size[max]", strconv.Itoa(s.Options.MaxSize))
		q.Add("size[min]", strconv.Itoa(s.Options.MinSize))
		q.Add("sortBy", "published_sort_desc")
		q.Add("limit", "1")
		req.URL.RawQuery = q.Encode()

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		var listings struct {
			Cards []map[string]interface{} `json:"cards"`
		}
		err = json.NewDecoder(resp.Body).Decode(&listings)
		if err != nil {
			return nil, err
		}

		reg, err := regexp.Compile("[^0-9]+")
		if err != nil {
			return "", err
		}
		for _, apiListing := range listings.Cards {
			price, err := strconv.Atoi(reg.ReplaceAllString(apiListing["price"].(string), ""))
			if err != nil {
				return "", err
			}

			listingDataJSON := null.JSON{}
			listingDataJSON.Marshal(apiListing)
			listing := models.Listing{
				ID:          int(apiListing["id"].(float64)),
				Price:       price,
				ListingData: listingDataJSON,
			}
			// check
			listing.SetArea(s.DB, false, area)

			listingDetailsJSON := null.JSON{}
			listingDetails, err := s.getListingDetails(listing)
			if err != nil {
				return "", err
			}
			fmt.Println(listingDetails)
			listingDetailsJSON.Marshal(listingDetails)
			listing.ListingDetails = listingDetailsJSON
			fmt.Println(string(listingDetailsJSON.JSON))

			listing.Insert(s.DB, boil.Infer())
		}
	}

	return "", nil
}

func (s *Search) getAreas(areaCodes []string) ([]*models.Area, error) {
	areasInDB, err := models.Areas(models.AreaWhere.Name.IN(areaCodes)).All(s.DB)
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

			dbArea := models.Area{
				ID:       area.Card.CardID,
				Name:     area.Card.Name,
				CardType: area.Card.CardType,
				City:     area.Parent.Name,
			}

			dbArea.Insert(s.DB, boil.Infer())
		}
	}

	return models.Areas(models.AreaWhere.Name.IN(areaCodes)).All(s.DB)
}

func (s *Search) apiCall(endpoint string) *http.Request {
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

func (s *Search) getArea(areaCode string) (ApiArea, error) {
	client := &http.Client{}

	req := s.apiCall("location")

	q := req.URL.Query()
	q.Add("query", areaCode)

	req.URL.RawQuery = q.Encode()

	resp, _ := client.Do(req)

	var list []ApiArea
	err := json.NewDecoder(resp.Body).Decode(&list)
	if err != nil {
		return ApiArea{}, err
	}

	if len(list) != 1 {
		return ApiArea{}, fmt.Errorf("Expected 1 card got %d", len(list))
	}

	return list[0], nil
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

type details = map[string]map[string]string

func (s *Search) getListingDetails(listing models.Listing) (details, error) {
	area, err := listing.Area().One(s.DB)
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(fmt.Sprintf("https://asunnot.oikotie.fi/myytavat-asunnot/%s/%d", area.City, listing.ID))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Request failed %d %s, id: %d", resp.StatusCode, resp.Status, listing.ID)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	listingDetails := make(details)

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

		listingDetails[h] = cat
	})

	return listingDetails, nil
}
