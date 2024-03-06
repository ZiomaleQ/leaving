package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	prog "github.com/gosuri/uiprogress"
)

var QuoteRegex = regexp.MustCompile(`"([^"]+)"`)

func getSeasons(url string) ([]entry, error) {
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)

	if err != nil {
		return nil, err
	}

	menuElt := doc.Find("#menu_prawa")

	if menuElt.Length() == 0 {
		return nil, errors.New("no menu found")
	}

	seriesPanel := menuElt.Children().Get(4)
	seriesElt := goquery.NewDocumentFromNode(seriesPanel).Find("a")

	if seriesElt.Length() == 0 {
		return nil, errors.New("no series found")
	}

	choices := make([]entry, 0)

	seriesElt.Each(func(i int, s *goquery.Selection) {
		txt := s.Text()

		if txt == "Openingi" || txt == "Endingi" {
			return
		}

		choices = append(choices, entry{
			url:  url + s.AttrOr("href", ""),
			name: strings.TrimSpace(txt),
		})
	})

	return choices, nil
}

func FetchAndParse(url string, client *http.Client) (*goquery.Document, error) {
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, errors.New("error creating request")
	}

	resp, err := client.Do(req)

	if err != nil {
		return nil, errors.New("error getting page")
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)

	if err != nil {
		return nil, errors.New("error parsing page")
	}

	return doc, nil

}

type animeEp struct {
	url  string
	name string
	num  int
}

func (ep *animeEp) GetName(max int) string {
	format := "%0" + fmt.Sprint((len(strconv.Itoa(max)))) + "d - %v"

	return fmt.Sprintf(format, ep.num, ep.name)
}

func (ep *animeEp) GetMediaURL() (string, error) {
	client := &http.Client{}

	doc, err := FetchAndParse(ep.url, client)

	if err != nil {
		return "", errors.New("error parsing episode page")
	}

	episodeTable := doc.Find("tbody").Children()

	playerUrl := ""

	baseUrl, err := url.Parse(ep.url)

	if err != nil {
		return "", err
	}

	for i := 0; i < episodeTable.Length(); i++ {
		episodeRow := episodeTable.Eq(i).Children()

		if strings.TrimSpace(episodeRow.Eq(2).Text()) == "cda" {
			playerUrl = baseUrl.Scheme + "://" + baseUrl.Host + "/" + "/odtwarzacz-" + episodeRow.Find(".odtwarzacz_link").AttrOr("rel", "") + ".html"
			break
		}
	}

	if playerUrl == "" {
		return "", errors.New("no wbijam cda player found")
	}

	doc, err = FetchAndParse(playerUrl, client)

	if err != nil {
		return "", errors.New("error parsing player page")
	}

	iframes := doc.Find("iframe").Map(func(i int, s *goquery.Selection) string {
		return s.AttrOr("src", "")
	})

	cdaPlayerLink := ""

	for _, iframe := range iframes {
		if strings.Contains(iframe, "cda.pl") {
			cdaPlayerLink = iframe
			break
		}
	}

	if cdaPlayerLink == "" {
		return "", errors.New("no cda player link found")
	}

	doc, err = FetchAndParse(cdaPlayerLink, client)

	if err != nil {
		return "", errors.New("error parsing cda player page")
	}

	baseUrl, err = url.Parse(cdaPlayerLink)

	if err != nil {
		return "", err
	}

	playerId := strings.Split(baseUrl.Path, "/")[2]

	jsonRaw := doc.Find("#mediaplayer"+playerId).AttrOr("player_data", "")

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonRaw), &data); err != nil {
		return "", errors.New("error parsing json")
	}

	videoUrl := data["video"].(map[string]interface{})["file"].(string)

	return decodeFromCda(videoUrl), nil
}

type StructWriter struct {
	bar *prog.Bar
}

func (sw *StructWriter) Write(p []byte) (n int, err error) {
	sw.bar.Set(sw.bar.Current() + len(p))
	return len(p), nil
}

func (ep *animeEp) Download(dir string, fileNum int) error {
	mediaUrl, err := ep.GetMediaURL()

	if err != nil {
		return err
	}

	resp, err := http.Get(mediaUrl)

	if err != nil {
		return err
	}

	bar := prog.AddBar(int(resp.ContentLength)).PrependCompleted().AppendElapsed()

	defer resp.Body.Close()

	out, err := os.Create(dir + "/" + ep.GetName(fileNum) + ".mp4")

	if err != nil {
		return err
	}

	defer out.Close()

	temp := &StructWriter{bar: bar}

	_, err = io.Copy(io.MultiWriter(out, temp), resp.Body)

	if err != nil {
		return err
	}

	return nil
}

func getEpisodes(seasonUrl string) ([]animeEp, error) {
	req, err := http.NewRequest("GET", seasonUrl, nil)

	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)

	if err != nil {
		return nil, err
	}

	episodeTable := doc.Find("tbody").Children()

	episodes := make([]animeEp, 0)

	baseUrl, err := url.Parse(seasonUrl)

	if err != nil {
		return nil, err
	}

	for i := 0; i < episodeTable.Length(); i++ {

		episodeRow := episodeTable.Eq(i).Find("a")

		epNameRaw := strings.TrimSpace(episodeRow.Text())

		epNameCleaned := QuoteRegex.FindString(epNameRaw)

		episodes = append(episodes, animeEp{
			url:  baseUrl.Scheme + "://" + baseUrl.Host + "/" + episodeRow.AttrOr("href", ""),
			name: epNameCleaned[1 : len(epNameCleaned)-1],
			num:  episodeTable.Length() - i,
		})
	}

	return episodes, nil
}
