package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"text/template"
)

type Client struct {
	OfficialURL           *url.URL
	TrendingRepositoryURL *url.URL
	HTTPClient            *http.Client
}

type Item struct {
	ID              int      `json:"id"`
	Name            string   `json:"name,repo"`
	FullName        string   `json:"full_name"`
	URL             string   `json:"repo_link"`
	HTMLURL         string   `json:"html_url"`
	CloneURL        string   `json:"clone_url"`
	Description     string   `json:"description"`
	Desc            string   `json:"desc"`
	StargazersCount int      `json:"stargazers_count,stars"`
	Stars           string   `json:"stars"`
	Watchers        int      `json:"watchers"`
	Topics          []string `json:"topics"`
	Language        string   `json:"language"`
	Lang            string   `json:"lang"`
	DefaultBranch   string   `json:"default_branch"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
	DataSource      string
}

type Readme struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	HTMLURL     string `json:"html_url"`
	DownloadURL string `json:"download_url"`
	Content     string `json:"content"`
}

type Result struct {
	Items []Item `json:"items"`
}

func (item *Item) GetRepositoryName() string {
	name := item.FullName
	if name == "" {
		url, err := url.Parse(item.URL)
		if err != nil {
			return ""
		}
		name = url.Path[1:]
	}
	return name
}

func (item *Item) GetStars() int {
	stars, _ := strconv.Atoi(strings.Replace(item.Stars, ",", "", -1))
	if stars == 0 {
		return item.StargazersCount
	}
	return stars
}

func (item *Item) GetRepositoryURL() string {
	url := item.HTMLURL
	if url == "" {
		return item.URL
	}
	return url
}
func (item *Item) GetDescription() string {
	description := item.Description
	if description == "" {
		return item.Desc
	}
	return description
}
func (item *Item) GetLanguage() string {
	language := item.Language
	if language == "" {
		return item.Lang
	}
	return language
}
func (item *Item) GetCloneURL() string {
	url := item.GetRepositoryURL()
	if !strings.HasSuffix(url, ".git") {
		return url + ".git"
	}
	return url
}

func (item *Item) String() string {
	const officialTemplateText = `
	Name       : {{.GetRepositoryName}}
	URL        : {{.GetRepositoryURL}}
	Star       : ⭐️ {{.StargazersCount}}
	Clone URL  : {{.GetCloneURL}}
	Description: {{.Description}}
	Watchers   : {{.Watchers}}
	Topics     : {{.Topics}}
	Language   : {{.Language}}
	CreatedAt  : {{.CreatedAt}}
	UpdatedAt  : {{.UpdatedAt}}
	`
	const trendingTemplateText = `
	Name       : {{.GetRepositoryName}}
	URL        : {{.GetRepositoryURL}}
	Star       : ⭐️ {{.Stars}}
	Clone URL  : {{.GetCloneURL}}
	Description: {{.GetDescription}}
	Language   : {{.GetLanguage}}
	`
	templateText := trendingTemplateText
	if item.DataSource == "OfficialAPI" {
		templateText = officialTemplateText
	}
	template, err := template.New("Repository").Parse(templateText)
	if err != nil {
		panic(err)
	}
	var doc bytes.Buffer
	if err := template.Execute(&doc, item); err != nil {
		panic(err)
	}
	return doc.String()
}

func (result *Result) Draw(writer io.Writer) error {
	for _, item := range result.Items {
		starText := " ⭐️ " + strconv.Itoa(item.GetStars())
		fmt.Fprintf(writer, "%-10.10s\033[32m%s\033[0m\n", starText, item.GetRepositoryName())
	}
	return nil
}

func NewClient() (*Client, error) {
	officialURL, err := url.Parse("https://api.github.com")
	if err != nil {
		return nil, err
	}
	trendingRepositoryURL, err := url.Parse("https://trendings.herokuapp.com/repo")
	if err != nil {
		return nil, err
	}
	return &Client{
		OfficialURL:           officialURL,
		TrendingRepositoryURL: trendingRepositoryURL,
		HTTPClient:            http.DefaultClient,
	}, nil
}

func (client *Client) SearchRepository(query string) (*Result, error) {
	url := *client.OfficialURL
	url.Path = path.Join(url.Path, "search", "repositories")
	req, err := http.NewRequest("GET", url.String()+"?q="+query, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Accept", "application/vnd.github.mercy-preview+json")
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result *Result
	if err = json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	items := result.Items
	for i := range items {
		result.Items[i].DataSource = "OfficialAPI"
	}
	return result, nil
}

func (client *Client) GetReadme(item Item) (*Readme, error) {
	url := *client.OfficialURL
	url.Path = path.Join(url.Path, "repos", item.GetRepositoryName(), "readme")
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Accept", "application/vnd.github.mercy-preview+json")
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var readme *Readme
	if err = json.Unmarshal(body, &readme); err != nil {
		return nil, err
	}
	return readme, nil
}
func (client *Client) GetTrendingRepository(language string, since string) (*Result, error) {
	q := client.TrendingRepositoryURL.Query()
	if language != "" {
		q.Set("lang", language)
	}
	if since != "" {
		q.Set("since", since)
	}
	url := client.TrendingRepositoryURL
	if len(q) != 0 {
		url.RawQuery = q.Encode()
	}
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/vnd.github.mercy-preview+json")
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result *Result
	if err = json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	for i := range result.Items {
		result.Items[i].DataSource = "TrendingAPI"
	}
	return result, nil
}
