package bl3_auto_vip

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// ShiftCode represents a shiftcode and its information
type ShiftCode struct {
	Code        string
	Reward      string
	Platforms   map[string]struct{}
	IsUniversal bool
}

// GetShiftPlatforms returns the list of platforms available for a single code
func (c *bl3Client) GetCodePlatforms(code string) (map[string]struct{}, error) {
	// See testdata/shift_info.json to see an output sample
	url := fmt.Sprintf("https://api.2k.com/borderlands/code/%s/info", code)
	resp, err := c.get(url)
	if err != nil {
		return nil, fmt.Errorf("http request error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		content, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("the request returned unexpected code %d with body %s", resp.StatusCode, string(content))
	}

	// create the type of the response inline since we don't need it outside of
	// this method
	type codeInfo struct {
		Platform     string `json:"offer_service"`
		IsActive     bool   `json:"is_active"`
		GameCodeName string `json:"offer_title"`
	}
	type codeInfoList struct {
		codes []codeInfo `json:"entitlement_offer_codes"`
	}

	// Parse the request response into an object
	list := codeInfoList{}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("could not JSON decode the response %w", err)
	}

	// get all the platforms
	platforms := map[string]struct{}{}
	for _, info := range list.codes {
		if info.IsActive && info.GameCodeName == bl3CodeName {
			platforms[info.Platform] = struct{}{}
		}
	}

	return platforms, nil
}

// RedeemShiftCode redeems the given shift code on the given platform
// The way the redemption works is by first sending a request to
// /reedem/:platform which will create a new job, and then by check the job
// status at /job/:job-id
func (c *bl3Client) RedeemShiftCode(code, platform string) error {
	// Create a new job by requesting a redemption
	url := fmt.Sprintf("https://api.2k.com/borderlands/code/%s/redeem/%s", code, platform)
	resp, err := c.post(url, "", nil)
	if err != nil {
		return fmt.Errorf("http request to redeem the code failed: %w", err)
	}
	defer resp.Body.Close()
	// If the status code is not 200 we can get the actual error by parsing it
	if resp.StatusCode != http.StatusCreated {
		type errorResp struct {
			Error struct {
				Code string `json:"code"`
				Msg  string `json:"message"`
			} `json:"error"`
		}
		serverErr := errorResp{}
		err := json.NewDecoder(resp.Body).Decode(&serverErr)
		if err != nil {
			serverErr.Error.Code = "INTERNAL"
			serverErr.Error.Msg = fmt.Sprintf("could not JSON decode the error: %s", err.Error())
		}
		return fmt.Errorf("the request to redeem the code returned an unexpected code %d with error %s - %s", resp.StatusCode, serverErr.Error.Code, serverErr.Error.Msg)
	}
	// create the type of the response inline since we don't need it outside of
	// this method
	type redemptionJob struct {
		ID   string `json:"job_id"`
		Wait int    `json:"max_wait_milliseconds"`
	}
	// Parse the response
	job := redemptionJob{}
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return fmt.Errorf("could not JSON decode the response %w", err)
	}
	// we no longer need the body
	resp.Body.Close()

	// wait to make sure the job has finished
	time.Sleep(time.Duration(job.Wait) * time.Millisecond)

	// Check the redemption status
	url = fmt.Sprintf("https://api.2k.com/borderlands/code/%s/job/%s", code, job.ID)
	resp, err = c.get(url)
	if err != nil {
		return fmt.Errorf("http request to check on the job failed %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		content, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("the request to check the redemption returned an unexpected code %d with body %s", resp.StatusCode, content)
	}

	return nil
}

func (c *bl3Client) GetUserPlatforms() (map[string]struct{}, error) {
	// Send the request to get info about the user
	// See testdata/user_info.json to see an output sample
	url := "https://api.2k.com/borderlands/users/me"
	resp, err := c.post(url, "", nil)
	if err != nil {
		return nil, fmt.Errorf("http request error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		content, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("the request returned unexpected code %d with body %s", resp.StatusCode, string(content))
	}

	// create the type of the response inline since we don't need it outside of
	// this method
	type userInfo struct {
		Platforms []string `json:"platforms"`
	}

	// Parse the request response into an object
	uInfo := userInfo{}
	if err := json.NewDecoder(resp.Body).Decode(&uInfo); err != nil {
		return nil, fmt.Errorf("could not JSON decode the response %w", err)
	}

	// get all the platforms
	platforms := map[string]struct{}{}
	for _, p := range uInfo.Platforms {
		if p != "twitch" {
			platforms[p] = struct{}{}
		}

	}
	return platforms, nil
}

func (c *bl3Client) GetFullShiftCodeList() ([]*ShiftCode, error) {
	// See testdata/shift_list.json to see an output sample
	resp, err := c.http.Get("https://shift.orcicorn.com/tags/borderlands3/index.json")
	if err != nil {
		return nil, fmt.Errorf("http request error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		content, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("the request returned unexpected code %d with body %s", resp.StatusCode, string(content))
	}
	// create the type of the response inline since we don't need it outside of
	// this method
	type shiftCode struct {
		Code     string `json:"code"`
		Platform string `json:"platform"`
		Reward   string `json:"reward"`
	}
	type codeList struct {
		Codes []shiftCode `json:"codes"`
	}

	// Parse the request response into an object
	// The response is wrapped in an array containing one elem
	var respObj []codeList
	if err := json.NewDecoder(resp.Body).Decode(&respObj); err != nil {
		return nil, fmt.Errorf("could not JSON decode the response %w", err)
	}
	if len(respObj) == 0 {
		return nil, nil
	}

	// create a list of ShiftCode off our current list
	// The issue here is that the format of a ShiftCode is different on
	// orcicorn and borderland's website.
	codes := make([]*ShiftCode, 0, len(respObj[0].Codes))
	for _, code := range respObj[0].Codes {
		newCode := &ShiftCode{
			Code:   code.Code,
			Reward: code.Reward,
		}
		platform := strings.ToLower(code.Platform)
		switch platform {
		case "universal":
			newCode.IsUniversal = true
		default:
			newCode.Platforms = map[string]struct{}{
				platform: struct{}{},
			}
		}
		codes = append(codes, newCode)
	}

	return codes, nil
}
