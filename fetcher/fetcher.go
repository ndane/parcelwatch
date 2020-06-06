package fetcher

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"

	log "github.com/sirupsen/logrus"
)

type Parcel struct {
	Code          string
	Collected     bool
	CollectedBy   string
	CollectedDate time.Time
	DeliveredDate time.Time
}

type session struct {
	resdata    string
	resdataKey string
	phpsessid  string
	subdomain  string
}

const (
	phpsessidKey = "PHPSESSID"
	baseURLFmt   = "https://%s.resi-sense.co.uk"
)

var (
	ErrInvalidPHPSESSID = errors.New("authenticate could not fetch phpsessid")
	ErrInvalidRESDATA   = errors.New("authenticate could not fetch resdata")
)

var sesh *session

func NewFetcher(subdomain string, refreshInterval time.Duration, username, password string) chan []Parcel {
	c := make(chan []Parcel)
	var err error
	sesh, err = authenticate(subdomain, username, password)
	if err != nil {
		log.WithError(err).Panic("could not authenticate")
	}

	go manageSession(subdomain, username, password)
	go getParcels(c, refreshInterval)
	return c
}

func baseURL(s *session) string {
	return fmt.Sprintf(baseURLFmt, s.subdomain)
}

func loginURL(s *session) string {
	return baseURL(s) + "/login/"
}

func deliveriesURL(s *session) string {
	return baseURL(s) + "/requests/deliveries/"
}

func manageSession(subdomain, username, password string) {
	for {
		<-time.After(time.Hour * 8)
		var err error
		sesh, err = authenticate(subdomain, username, password)
		if err != nil {
			log.WithError(err).Error("failed to refresh authentication credentials")
		}
	}
}

func (c *session) SetCookies(u *url.URL, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		switch cookie.Name {
		case phpsessidKey:
			c.phpsessid = cookie.Value
		}

		if strings.HasSuffix(cookie.Name, "_resdata") {
			c.resdata = cookie.Value
			c.resdataKey = cookie.Name
		}
	}
}

func (c *session) Cookies(u *url.URL) []*http.Cookie {
	cookies := make([]*http.Cookie, 0)
	if len(c.phpsessid) > 0 {
		cookies = append(cookies, &http.Cookie{
			Name:  phpsessidKey,
			Value: c.phpsessid,
			Path:  "/",
		})
	}
	if len(c.resdata) > 0 {
		cookies = append(cookies, &http.Cookie{
			Name:  c.resdataKey,
			Value: c.resdata,
			Path:  "/",
		})
	}
	return cookies
}

func authenticate(subdomain, username, password string) (*session, error) {
	s := &session{subdomain: subdomain}

	payload := strings.NewReader(fmt.Sprintf("page=&res_username=%s&res_password=%s", username, password))

	client := &http.Client{
		Jar: s,
	}

	req, err := http.NewRequest("POST", loginURL(s), payload)
	if err != nil {
		log.WithError(err).Error("failed to authenticate")
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(req)
	if err != nil {
		log.WithError(err).Error("failed to authenticate")
		return nil, err
	}

	if len(s.phpsessid) == 0 {
		return nil, ErrInvalidPHPSESSID
	}

	if len(s.resdata) == 0 {
		return nil, ErrInvalidRESDATA
	}

	return s, nil
}

func getParcels(c chan []Parcel, refreshInterval time.Duration) {
	f := false
	for {
		if f {
			<-time.After(time.Second * 5)
		}
		f = true

		p, err := getPage(sesh)
		if err != nil {
			log.WithError(err).Error("failed to fetch deliveries page")
			continue
		}

		parcels, err := deguffHTML(p)
		if err != nil {
			log.WithError(err).Error("failed to deguff HTML page")
			continue
		}

		c <- parcels

		<-time.After(refreshInterval)
	}
}

func getPage(s *session) (string, error) {
	client := new(http.Client)
	req, err := http.NewRequest("GET", deliveriesURL(s), nil)
	if err != nil {
		log.WithError(err).Error("failed to create a http request")
		return "", err
	}

	cookies := fmt.Sprintf("WILBURN_resdata=%s; PHPSESSID=%s", s.resdata, s.phpsessid)
	req.Header.Add("Cookie", cookies)

	res, err := client.Do(req)
	if err != nil {
		log.WithError(err).Error("failed to perform http request")
		return "", err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	b := strings.ReplaceAll(string(body), "\n", "")
	b = strings.ReplaceAll(b, "\t", "")

	return b, err
}

func deguffHTML(page string) ([]Parcel, error) {
	doc, _ := html.Parse(strings.NewReader(page))

	var findParcelTableNode func(*html.Node) (*html.Node, error)
	findParcelTableNode = func(n *html.Node) (*html.Node, error) {
		if n.Type == html.ElementNode && n.Data == "table" {
			for _, a := range n.Attr {
				if a.Key == "id" && a.Val == "historic_parcels" {
					return n, nil
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			n2, err := findParcelTableNode(c)
			if err == nil {
				return n2, nil
			}
		}

		return nil, errors.New("parcel table not found")
	}

	parcelTable, err := findParcelTableNode(doc)
	if err != nil {
		log.WithError(err).Error("failed to find parcel table")
		return nil, err
	}

	parcels := make([]Parcel, 0)
	var getRows func(*html.Node)
	getRows = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			code := n.FirstChild.FirstChild.Data
			status := n.LastChild.FirstChild.Data
			parcels = append(parcels, parseParcelRow(code, status))
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			getRows(c)
		}
	}
	getRows(parcelTable)

	return parcels, nil
}

func parseParcelRow(code, status string) Parcel {
	p := Parcel{
		Code: code,
	}

	p.Collected = strings.HasPrefix(status, "Collected")

	cbR := regexp.MustCompile("by (.+) on")
	p.CollectedBy = cbR.FindStringSubmatch(status)[1]

	dateR := regexp.MustCompile("\\d{1,2}[a-z][a-z] .+ \\d\\d\\d\\d")
	date := dateR.FindString(status)
	if i := strings.Index(date, "st"); i != -1 && i <= 2 {
		date = date[0:i] + date[i+2:]
	}

	if i := strings.Index(date, "nd"); i != -1 && i <= 2 {
		date = date[0:i] + date[i+2:]
	}

	if i := strings.Index(date, "rd"); i != -1 && i <= 2 {
		date = date[0:i] + date[i+2:]
	}

	if i := strings.Index(date, "th"); i != -1 && i <= 2 {
		date = date[0:i] + date[i+2:]
	}

	if d, err := time.Parse("2 January 2006", date); err != nil {
		log.WithError(err).Error("failed to parse collected at date")
	} else {
		if p.Collected {
			p.CollectedDate = d
		} else {
			p.DeliveredDate = d
		}
	}

	return p
}
