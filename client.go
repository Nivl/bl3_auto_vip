package bl3_auto_vip

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"time"
)

const (
	bl3CodeName = "oak"
	loginURL    = "https://api.2k.com/borderlands/users/authenticate"
)

// we make sure bl3Client implements Bl3Client
var _ Bl3Client = (*bl3Client)(nil)

// Bl3Client is an interface used to interact with the different APIs
// needed to get the data we need
type Bl3Client interface {
	Login(username string, password string) error
	GetCodePlatforms(code string) (map[string]struct{}, error)
	GetUserPlatforms() (map[string]struct{}, error)
	GetFullShiftCodeList() ([]*ShiftCode, error)
	RedeemShiftCode(code, platform string) error
}

type bl3Client struct {
	http    *http.Client
	headers http.Header
	Config  *Bl3Config
}

// NewBl3Client creates an returns a new client used to interact with
// all the needed APIs
func NewBl3Client() (Bl3Client, error) {
	// We need to setup a cookie jar so we can read/write on multiple
	// endpoints
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("could not setup cookies jar: %w", err)
	}

	clt := &bl3Client{
		http: &http.Client{
			Jar:     jar,
			Timeout: 1 * time.Minute,
		},
		headers: http.Header{
			"User-Agent": []string{"BL3 Auto Vip"},
		},
	}

	// Fetch & decode the config fle
	// TODO(melvin): don't hardcode the link
	resp, err := clt.http.Get("https://raw.githubusercontent.com/Nivl/bl3_auto_vip/master/config.json")
	if err != nil {
		return nil, fmt.Errorf("could not retrieve config file from github: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		content, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("config file request returned unexpected code %d with body %s", resp.StatusCode, string(content))
	}
	if err = json.NewDecoder(resp.Body).Decode(&clt.Config); err != nil {
		return nil, fmt.Errorf("could not JSON decode the config file %w", err)
	}

	// Set the needed headers
	// Taken from https://raw.githubusercontent.com/matt1484/bl3_auto_vip/master/config.json
	clt.headers.Set("Origin", "https://borderlands.com")
	clt.headers.Set("Referer", "https://borderlands.com/en-US/vip/")
	return clt, nil
}

// Login logs the user in 2k's website
func (c *bl3Client) Login(username string, password string) error {
	// Encode the request
	creds := map[string]string{
		"username": username,
		"password": password,
	}
	jsonData, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("could not json encode the credentials: %w", err)
	}

	// Perform the request
	resp, err := c.post(loginURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("could not submit login credentials: %w", err)
	}
	defer resp.Body.Close()

	// Validate the response
	if resp.StatusCode != http.StatusOK {
		content, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("login request returned unexpected status code %d with body %s", resp.StatusCode, content)
	}
	// We don't use the body beside for the error
	resp.Body.Close()

	// Store the session token
	c.headers.Set("X-SESSION", resp.Header.Get("X-SESSION-SET"))
	return nil
}

// do adds the headers to the request and sends it
func (c *bl3Client) do(req *http.Request) (*http.Response, error) {
	for k, v := range c.headers {
		for _, x := range v {
			req.Header.Set(k, x)
		}
	}
	return c.http.Do(req)
}

// get sends a GET request to a 2k/borderland server
// This is not safe to use for a non 2k/borderland website
func (c *bl3Client) get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// head sends a HEAD request to a 2k/borderland server
// This is not safe to use for a non 2k/borderland website
func (c *bl3Client) head(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// post sends a POST request to a 2k/borderland server
// This is not safe to use for a non 2k/borderland website
func (c *bl3Client) post(url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.do(req)
}
