package blcodes

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/Nivl/blcodes/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCodePlatform(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		// initiate the mock
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		// Setup the response expected by the HTTP mock
		respData, err := ioutil.ReadFile("testdata/shift_info.json")
		require.NoError(t, err, "could not get stub data")
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader(respData)),
		}

		// add the assertions and expectations
		httpClt := mocks.NewMockHTTPClient(mockCtrl)
		httpClt.EXPECT().Do(gomock.Any()).Return(resp, nil)

		// run the tests
		clt := &bl3Client{
			http: httpClt,
		}
		platforms, err := clt.GetCodePlatforms("code string")
		require.NoError(t, err)
		require.NotEmpty(t, platforms)
		assert.Contains(t, platforms, "steam")
		assert.Contains(t, platforms, "epic")
		assert.Contains(t, platforms, "xboxlive")
		assert.Contains(t, platforms, "psn")
		assert.Contains(t, platforms, "stadia")
	})

	t.Run("invalid request/response should return an error", func(t *testing.T) {
		t.Parallel()

		// initiate the mock
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		// add the assertions and expectations
		httpClt := mocks.NewMockHTTPClient(mockCtrl)
		httpClt.EXPECT().Do(gomock.Any()).Return(nil, &url.Error{})

		// run the tests
		clt := &bl3Client{
			http: httpClt,
		}
		platforms, err := clt.GetCodePlatforms("code string")
		require.Empty(t, platforms)
		require.Error(t, err)
		require.Contains(t, err.Error(), "http request error")
	})

	t.Run("invalid status code should return an error", func(t *testing.T) {
		t.Parallel()

		// initiate the mock
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		// Setup the response expected by the HTTP mock
		resp := &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
		}

		// add the assertions and expectations
		httpClt := mocks.NewMockHTTPClient(mockCtrl)
		httpClt.EXPECT().Do(gomock.Any()).Return(resp, nil)

		// run the tests
		clt := &bl3Client{
			http: httpClt,
		}
		platforms, err := clt.GetCodePlatforms("code string")
		require.Empty(t, platforms)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unexpected code 500")
	})

	t.Run("invalid json response should return an error", func(t *testing.T) {
		t.Parallel()

		// initiate the mock
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		// Setup the response expected by the HTTP mock
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte("not JSON"))),
		}

		// add the assertions and expectations
		httpClt := mocks.NewMockHTTPClient(mockCtrl)
		httpClt.EXPECT().Do(gomock.Any()).Return(resp, nil)

		// run the tests
		clt := &bl3Client{
			http: httpClt,
		}
		platforms, err := clt.GetCodePlatforms("code string")
		require.Empty(t, platforms)
		require.Error(t, err)
		require.Contains(t, err.Error(), "could not JSON decode")
	})
}

func TestGetUserPlatforms(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		// initiate the mock
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		// Setup the response expected by the HTTP mock
		respData, err := ioutil.ReadFile("testdata/user_info.json")
		require.NoError(t, err, "could not get stub data")
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader(respData)),
		}

		// add the assertions and expectations
		httpClt := mocks.NewMockHTTPClient(mockCtrl)
		httpClt.EXPECT().Do(gomock.Any()).Return(resp, nil)

		// run the tests
		clt := &bl3Client{
			http: httpClt,
		}
		platforms, err := clt.GetUserPlatforms()
		require.NoError(t, err)
		require.Len(t, platforms, 2)
		require.Contains(t, platforms, "stadia")
		require.Contains(t, platforms, "xboxlive")
	})
}

func TestGetFullShiftCodeList(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		// initiate the mock
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		// Setup the response expected by the HTTP mock
		respData, err := ioutil.ReadFile("testdata/shift_list.json")
		require.NoError(t, err, "could not get stub data")
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader(respData)),
		}

		// add the assertions and expectations
		httpClt := mocks.NewMockHTTPClient(mockCtrl)
		httpClt.EXPECT().Get(gomock.Any()).Return(resp, nil)

		// run the tests
		clt := &bl3Client{
			http: httpClt,
		}
		codes, err := clt.GetFullShiftCodeList()
		require.NoError(t, err)
		require.Len(t, codes, 14)

		// The first code should be universal
		// {
		// 	"code": "WJCBB-WRXS9-R35ZW-HT5TJ-9HJ59",
		// 	"type": "shift",
		// 	"game": "Borderlands 3",
		// 	"platform": "Universal",
		// 	"reward": "1 Gold Key",
		// 	"archived": "20 Dec 2019 17:36:00 -0500",
		// 	"expires": "Unknown",
		// 	"link": "https://shift.orcicorn.com/shift-code/wjcbb-wrxs9-r35zw-ht5tj-9hj59/?utm_source=json&utm_medium=bl3&utm_campaign=automation"
		// }
		if assert.Contains(t, codes[0].Code, "WJCBB-WRXS9-R35ZW-HT5TJ-9HJ59") {
			assert.Contains(t, codes[0].Reward, "1 Gold Key")
			assert.True(t, codes[0].IsUniversal, "code should be universal")
			assert.Empty(t, codes[0].Platforms, "platforms should be empty")
		}

		// The third code should be Epic only
		// {
		// 	"code": "KSW3T-T59JS-CWF96-RBJ33-T3FCW",
		// 	"type": "shift",
		// 	"game": "Borderlands 3",
		// 	"platform": "Epic",
		// 	"reward": "Snowglobe ECHO Skin",
		// 	"archived": "17 Dec 2019 11:25:00 -0500",
		// 	"expires": "10 Jan 2020 23:59:00 -0500",
		// 	"link": "https://shift.orcicorn.com/shift-code/ksw3t-t59js-cwf96-rbj33-t3fcw/?utm_source=json&utm_medium=bl3&utm_campaign=automation"
		// },
		if assert.Contains(t, codes[2].Code, "KSW3T-T59JS-CWF96-RBJ33-T3FCW") {
			assert.Contains(t, codes[2].Reward, "Snowglobe ECHO Skin")
			assert.False(t, codes[2].IsUniversal, "code should NOT be universal")
			require.Contains(t, codes[2].Platforms, "epic")
		}
	})
}
