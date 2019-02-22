package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Billz95/Rocket.Chat.Go.SDK/models"
	"github.com/tebeka/selenium"
	"net/url"
	"os"
	"time"
)

type logoutResponse struct {
	Status
	Data struct {
		Message string `json:"message"`
	} `json:"data"`
}

type logonResponse struct {
	Status
	Data struct {
		Token  string `json:"authToken"`
		UserID string `json:"userID"`
	} `json:"data"`
}

type CreateUserResponse struct {
	Status
	User struct {
		ID        string    `json:"_id"`
		CreatedAt time.Time `json:"createdAt"`
		Services  struct {
			Password struct {
				Bcrypt string `json:"bcrypt"`
			} `json:"password"`
		} `json:"services"`
		Username string `json:"username"`
		Emails   []struct {
			Address  string `json:"address"`
			Verified bool   `json:"verified"`
		} `json:"emails"`
		Type         string            `json:"type"`
		Status       string            `json:"status"`
		Active       bool              `json:"active"`
		Roles        []string          `json:"roles"`
		UpdatedAt    time.Time         `json:"_updatedAt"`
		Name         string            `json:"name"`
		CustomFields map[string]string `json:"customFields"`
	} `json:"user"`
}

// Login a user. The Email and the Password are mandatory. The auth token of the user is stored in the Client instance.
//
// https://rocket.chat/docs/developer-guides/rest-api/authentication/login
func (c *Client) Login(credentials *models.UserCredentials) error {
	if c.auth != nil {
		return nil
	}

	if credentials.ID != "" && credentials.Token != "" {
		c.auth = &authInfo{id: credentials.ID, token: credentials.Token}
		return nil
	}

	response := new(logonResponse)
	data := url.Values{"user": {credentials.Email}, "password": {credentials.Password}}
	if err := c.PostForm("login", data, response); err != nil {
		return err
	}

	c.auth = &authInfo{id: response.Data.UserID, token: response.Data.Token}
	credentials.ID, credentials.Token = response.Data.UserID, response.Data.Token
	return nil
}

const (
	// These paths will be different on your system.
	port = 8123
)

var loginRequset string

func (c *Client) LoginViaGoogle(credentials *models.UserCredentials) error {
	if c.auth != nil {
		return nil
	}

	if credentials.Token != "" {
		succ := c.TryCredential(credentials)
		if succ {
			return nil
		}
	}

	uid, utoken := LoginUsingSelenium()
	credentials.ID = uid
	credentials.Token = utoken
	c.auth = &authInfo{id: credentials.ID, token: credentials.Token}

	return nil
}

func (c *Client) TryCredential(credentials *models.UserCredentials) bool {
	c.auth = &authInfo{id: credentials.ID, token: credentials.Token}
	_, err := c.GetPublicChannels()
	if err != nil {
		return false
	}
	return true
}

func LoginUsingSelenium() (string, string) {
	pwd, _ := os.Getwd()
	seleniumPath := pwd + "/vendor/github.com/tebeka/selenium/vendor/selenium-server-standalone-3.14.0.jar"
	geckoDriverPath := pwd + "/vendor/github.com/tebeka/selenium/vendor/geckodriver_mac"

	opts := []selenium.ServiceOption{
		selenium.GeckoDriver(geckoDriverPath), // Specify the path to GeckoDriver in order to use Firefox.
		selenium.Output(os.Stderr),            // Output debug information to STDERR.
	}

	service, err := selenium.NewSeleniumService(seleniumPath, port, opts...)
	if err != nil {
		panic(err) // panic is used only as an example and is not otherwise recommended.
	}
	defer service.Stop()

	// Connect to the WebDriver instance running locally.
	caps := selenium.Capabilities{"browserName": "firefox"}
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", port))

	if err != nil {
		panic(err)
	}
	defer wd.Quit()

	// Navigate to the simple playground interface.
	wd.Get("https://chat.tools.flnltd.com")
	err = wd.Wait(LoggedIn)

	uId, _ := wd.GetCookie("rc_uid")
	uToken, _ := wd.GetCookie("rc_token")
	return uId.Value, uToken.Value
}

func LoggedIn(wd selenium.WebDriver) (bool, error) {
	_, err2 := wd.GetCookie("rc_token")
	return err2 == nil, nil
}

func Halt(wd selenium.WebDriver) (bool, error) {
	return false, nil
}

// CreateToken creates an access token for a user
//
// https://rocket.chat/docs/developer-guides/rest-api/users/createtoken/
func (c *Client) CreateToken(userID, username string) (*models.UserCredentials, error) {
	response := new(logonResponse)
	data := url.Values{"userId": {userID}, "username": {username}}
	if err := c.PostForm("users.createToken", data, response); err != nil {
		return nil, err
	}
	credentials := &models.UserCredentials{}
	credentials.ID, credentials.Token = response.Data.UserID, response.Data.Token
	return credentials, nil
}

// Logout a user. The function returns the response message of the server.
//
// https://rocket.chat/docs/developer-guides/rest-api/authentication/logout
func (c *Client) Logout() (string, error) {

	if c.auth == nil {
		return "Was not logged in", nil
	}

	response := new(logoutResponse)
	if err := c.Get("logout", nil, response); err != nil {
		return "", err
	}

	return response.Data.Message, nil
}

// CreateUser being logged in with a user that has permission to do so.
//
// https://rocket.chat/docs/developer-guides/rest-api/users/create
func (c *Client) CreateUser(req *models.CreateUserRequest) (*CreateUserResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	response := new(CreateUserResponse)
	err = c.Post("users.create", bytes.NewBuffer(body), response)
	return response, err
}

// UpdateUser updates a user's data being logged in with a user that has permission to do so.
//
// https://rocket.chat/docs/developer-guides/rest-api/users/update/
func (c *Client) UpdateUser(req *models.UpdateUserRequest) (*CreateUserResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	response := new(CreateUserResponse)
	err = c.Post("users.update", bytes.NewBuffer(body), response)
	return response, err
}

// SetUserAvatar updates a user's avatar being logged in with a user that has permission to do so.
// Currently only passing an URL is possible.
//
// https://rocket.chat/docs/developer-guides/rest-api/users/setavatar/
func (c *Client) SetUserAvatar(userID, username, avatarURL string) (*Status, error) {
	body := fmt.Sprintf(`{ "userId": "%s","username": "%s","avatarUrl":"%s"}`, userID, username, avatarURL)
	response := new(Status)
	err := c.Post("users.setAvatar", bytes.NewBufferString(body), response)
	return response, err
}
