package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
		fmt.Println(err)
		os.Exit(1)
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

type animeEp struct {
	url  string
	name string
	num  int
}

func (ep *animeEp) GetName(max int) string {
	format := "%0" + fmt.Sprint((len(strconv.Itoa(max)))) + "d - %v\n"

	return fmt.Sprintf(format, ep.num, ep.name)
}

type Video struct {
	File string `json:"file"`
}

func (ep *animeEp) GetMediaURL() string {
	req, err := http.NewRequest("GET", ep.url, nil)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	client := &http.Client{}
	episodeResp, err := client.Do(req)

	if err != nil {
		return ""
	}

	defer episodeResp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(episodeResp.Body)

	if err != nil {
		return ""
	}

	episodeTable := doc.Find("tbody").Children()

	playerUrl := ""

	for i := 0; i < episodeTable.Length(); i++ {
		episodeRow := episodeTable.Eq(i)

		if strings.TrimSpace(episodeRow.Eq(3).Text()) == "cda" {
			playerUrl = ep.url + "/odtwarzacz-" + episodeRow.Find(".odtwarzacz_link").AttrOr("rel", "") + ".html"
			break
		}
	}

	if playerUrl == "" {
		return ""
	}

	req, err = http.NewRequest("GET", playerUrl, nil)

	if err != nil {
		return ""
	}

	episodePlayerResp, err := client.Do(req)

	if err != nil {
		return ""
	}

	defer episodePlayerResp.Body.Close()

	doc, err = goquery.NewDocumentFromReader(episodePlayerResp.Body)

	if err != nil {
		return ""
	}

	iframes := doc.Find("iframe")
	cdaPlayerLink := ""

	for i := 0; i < iframes.Length(); i++ {
		iframe := iframes.Eq(i)

		if strings.Contains(iframe.AttrOr("src", ""), "cda") {
			cdaPlayerLink = iframe.AttrOr("src", "")
			break
		}
	}

	if cdaPlayerLink == "" {
		return ""
	}

	req, err = http.NewRequest("GET", cdaPlayerLink, nil)

	if err != nil {
		return ""
	}

	cdaPlayerResp, err := client.Do(req)

	if err != nil {
		return ""
	}

	defer cdaPlayerResp.Body.Close()

	doc, err = goquery.NewDocumentFromReader(cdaPlayerResp.Body)

	if err != nil {
		return ""
	}

	playerId := strings.Split(cdaPlayerLink, "/")[2]

	jsonRaw := doc.Find("#mediaplayer"+playerId).AttrOr("player_data", "")

	var data map[string]Video
	if err := json.Unmarshal([]byte(jsonRaw), &data); err != nil {
		return ""
	}

	videoUrl := data["video"].File

	fmt.Println(videoUrl)

	return ""
}

func (ep *animeEp) Download(dir string, fileNum int, bar *prog.Bar) {
	mediaUrl := ep.GetMediaURL()

	if mediaUrl == "" {
		return
	}

	resp, err := http.Get(mediaUrl)

	if err != nil {
		fmt.Println(err)
		return
	}

	defer resp.Body.Close()

	out, err := os.Create(dir + "/" + ep.GetName(fileNum) + ".mp4")

	if err != nil {
		fmt.Println(err)
		return
	}

	defer out.Close()

	_, err = io.Copy(out, resp.Body)

	if err != nil {
		fmt.Println(err)
		return
	}
}

func getEpisodes(url string) ([]animeEp, error) {
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
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

	for i := 0; i < episodeTable.Length(); i++ {

		episodeRow := episodeTable.Eq(i).Find("a")

		epNameRaw := strings.TrimSpace(episodeRow.Text())

		epNameCleaned := QuoteRegex.FindString(epNameRaw)

		episodes = append(episodes, animeEp{
			url:  url + episodeRow.AttrOr("href", ""),
			name: epNameCleaned[1 : len(epNameCleaned)-1],
			num:  episodeTable.Length() - i,
		})
	}

	return episodes, nil
}
