package bl3_auto_vip

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"

	"github.com/PuerkitoBio/goquery"
	"github.com/thedevsaddam/gojsonq"
)

type HttpClient struct {
	http.Client
	headers http.Header
}

type HttpResponse struct {
	http.Response
}

func NewHttpClient() (*HttpClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("could not setup cookies: %w", err)
	}

	return &HttpClient{
		http.Client{
			Jar: jar,
		},
		http.Header{
			"User-Agent": []string{"BL3 Auto Vip"},
		},
	}, nil
}

func (response *HttpResponse) BodyAsHtmlDoc() (*goquery.Document, error) {
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid response code, expected %d got %d", http.StatusOK, response.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, fmt.Errorf("could not parse HTML: %w", err)
	}

	return doc, nil
}

func (response *HttpResponse) BodyAsJSON() (*gojsonq.JSONQ, error) {
	defer response.Body.Close()

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response: %w", err)
	}

	return JsonFromBytes(bodyBytes), nil
}

func getResponse(res *http.Response, err error) (*HttpResponse, error) {
	return &HttpResponse{
		*res,
	}, err
}

func (client *HttpClient) SetDefaultHeader(k, v string) {
	client.headers.Set(k, v)
}

func (client *HttpClient) Do(req *http.Request) (*HttpResponse, error) {
	for k, v := range client.headers {
		for _, x := range v {
			req.Header.Set(k, x)
		}
	}
	return getResponse(client.Client.Do(req))
}

func (client *HttpClient) Get(url string) (*HttpResponse, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func (client *HttpClient) Head(url string) (*HttpResponse, error) {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func (client *HttpClient) Post(url, contentType string, body io.Reader) (*HttpResponse, error) {
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return client.Do(req)
}

func (client *HttpClient) PostJson(url string, data interface{}) (*HttpResponse, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return client.Post(url, "application/json", bytes.NewBuffer(jsonData))
}

type Bl3Client struct {
	HttpClient
	Config Bl3Config
}

func NewBl3Client() (*Bl3Client, error) {
	client, err := NewHttpClient()
	if err != nil {
		return nil, fmt.Errorf("could not create http client: %w", err)
	}

	res, err := client.Get("https://raw.githubusercontent.com/Nivl/bl3_auto_vip/master/config.json")
	if err != nil {
		return nil, fmt.Errorf("could not retrive config file from github: %w", err)
	}

	configJSON, err := res.BodyAsJSON()
	if err != nil {
		return nil, fmt.Errorf("could not parse body as json: %w", err)
	}
	config := Bl3Config{}
	configJSON.Out(&config)

	for header, value := range config.RequestHeaders {
		client.SetDefaultHeader(header, value)
	}

	return &Bl3Client{
		HttpClient: *client,
		Config:     config,
	}, nil
}

func (client *Bl3Client) Login(username string, password string) error {
	data := map[string]string{
		"username": username,
		"password": password,
	}

	loginRes, err := client.PostJson(client.Config.LoginUrl, data)
	if err != nil {
		return fmt.Errorf("could not submit login credentials: %w", err)
	}
	defer loginRes.Body.Close()

	if loginRes.StatusCode != http.StatusOK {
		return fmt.Errorf("login request return unexpected status code: %d", loginRes.StatusCode)
	}

	redirectHeader := loginRes.Header.Get(client.Config.LoginRedirectHeader)
	if redirectHeader == "" {
		return errors.New("could not find redirect header")
	}

	sessionRes, err := client.Get(redirectHeader)
	if err != nil {
		return fmt.Errorf("could not get session: %w", err)
	}
	defer sessionRes.Body.Close()

	client.SetDefaultHeader(client.Config.SessionHeader, loginRes.Header.Get(client.Config.SessionIdHeader))
	return nil
}
