package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/cavaliergopher/grab/v3"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/go-querystring/query"
)

type JellyfinItem struct {
	Name,
	Id,
	SeriesName,
	SeasonName string
	SeasonNumber  int `json:"ParentIndexNumber"`
	EpisodeNumber int `json:"IndexNumber"`
	IsFolder      bool
}

type Response struct {
	Items []JellyfinItem
}

type Query struct {
	ParentId string   `url:"parentId,omitempty"`
	Ids      []string `url:"ids,omitempty"`
}

const itemsUrl = "/Users/{userId}/Items"
const downloadUrl = "/Items/{id}/Download"
const auth = "MediaBrowser Client=\"Download Client\", Device=\"Linux\", DeviceId=\"PlRvNOqV9GYvBBUssdhY\", Version=\"1.0\", Token=\"{token}\""

var client = &http.Client{}

func getChilds(parentId string, config *Config) Response {
	if parentId == "" {
		return queryItems(nil, config)
	} else {
		return queryItems(&Query{ParentId: parentId}, config)
	}
}

func getItems(items []string, config *Config) Response {
	var res Response
	chunked := chunkBy(items, 200)
	for _, v := range chunked {
		current := queryItems(&Query{Ids: v}, config)
		res.Items = append(res.Items, current.Items...)
	}
	return res
}

type incorrectAPIEndPointMsg string //With the message if there is

func queryItems(q *Query, config *Config) Response {
	var userId = config.UserId
	if userId == "" {
		userId = "0"
	}
	requestUrl := strings.ReplaceAll(config.APIEndpoint+itemsUrl, "{userId}", userId)
	if q != nil {
		queryString, _ := query.Values(q)
		requestUrl += "?" + queryString.Encode()
	}
	program.Send(infoMsg{"Fetching " + requestUrl})
	req, err := http.NewRequest("GET", requestUrl, nil)
	checkError(err)
	req.Header.Add("X-Emby-Authorization", strings.ReplaceAll(auth, "{token}", config.APIKey))
	resp, err2 := client.Do(req)

	if err2 != nil {
		switch err2 := err2.(type) {
		case *url.Error:
			program.Send(incorrectAPIEndPointMsg("Incorrect Endpoint"))
			program.Send(infoMsg{""})
			return Response{}
		default:
			checkError(err2)
		}
	}

	checkResp(resp)
	data, err3 := ioutil.ReadAll(resp.Body)
	checkError(err3)
	body := Response{}
	json.Unmarshal(data, &body)
	program.Send(infoMsg{""})
	return body
}

var red = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

type incorrectAPIKeyMsg string
type incorrectUserIdMsg string

func checkResp(resp *http.Response) {
	if resp.StatusCode != 200 {
		bodyText, err := ioutil.ReadAll(resp.Body)
		checkError(err)
		if resp.StatusCode == 401 {
			program.Send(incorrectAPIKeyMsg("Incorrect API key"))
			return
		}

		if resp.StatusCode == 400 {
			program.Send(incorrectUserIdMsg("Incorrect UserId"))
			return
		}

		body := string(bodyText)
		if body != "" {
			body = "null"
		}
		checkError(fmt.Errorf("Server responded with error code %d when calling %s \nbody: %s", resp.StatusCode, resp.Request.URL, body))
	}
}

type startDownloadingItemMsg string
type downloadStartedMsg struct {
	Id   string
	resp *grab.Response
}

var grabClient = grab.NewClient()

func downloadFile(id, dest string, config *Config) (string, error) {
	program.Send(startDownloadingItemMsg(id))

	req, err := grab.NewRequest(dest, strings.ReplaceAll(config.APIEndpoint+downloadUrl, "{id}", id))
	checkError(err)
	req.HTTPRequest.Header.Add("X-Emby-Authorization", strings.ReplaceAll(auth, "{token}", config.APIKey))
	resp := grabClient.Do(req)
	program.Send(downloadStartedMsg{id, resp})

	<-resp.Done

	if err := resp.Err(); err != nil {
		os.Remove(resp.Filename)
		return "", err
	}

	return resp.Filename, nil
}
