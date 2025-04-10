package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type UserInfo struct {
	User  string `json:"user"`
	Email string `json:"email"`
}

func getUserInfo(c *gin.Context) (*UserInfo, error) {
	if os.Getenv("DEV") == "true" {
		return &UserInfo{
			User:  "test",
			Email: "test@example.com",
		}, nil
	}

	oauth2UserinfoEndpoint := os.Getenv("OAUTH2_USERINFO_ENDPOINT")
	if oauth2UserinfoEndpoint == "" {
		return nil, errors.New("OAUTH2_USERINFO_ENDPOINT is not set")
	}

	cookie, err := c.Cookie("_oauth2_proxy")
	if err != nil {
		return nil, err
	}

	// Send cookie to auth endpoint to get userinfo
	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", oauth2UserinfoEndpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Cookie", "_oauth2_proxy="+cookie)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, err
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}
