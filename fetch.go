package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"

	"github.com/doraemonkeys/doraemon"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/term"
)

type syncThingConn struct {
	userName string
	host     string
	// password   string
	authPassed bool
	client     *http.Client
}

func NewSyncThingConn(userName, host string) (*syncThingConn, error) {
	if userName == "" {
		return nil, fmt.Errorf("user is empty")
	}
	if host == "" {
		return nil, fmt.Errorf("host is empty")
	}
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, fmt.Errorf("error creating cookie jar: %v", err)
	}
	return &syncThingConn{
		userName: userName,
		host:     host,
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Jar: jar,
		},
	}, nil
}

func (s *syncThingConn) Connect(password string) error {
	// GET host
	resp, err := s.client.Get(s.host)
	if err != nil {
		return fmt.Errorf("error getting host: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get host with status code: %d", resp.StatusCode)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// POST auth
	authURL := fmt.Sprintf("%s/rest/noauth/auth/password", s.host)
	authData := struct {
		Username     string `json:"username"`
		Password     string `json:"password"`
		StayLoggedIn bool   `json:"stayLoggedIn"`
	}{
		Username:     s.userName,
		Password:     password,
		StayLoggedIn: true,
	}

	authPayload, err := json.Marshal(authData)
	if err != nil {
		return fmt.Errorf("error marshaling auth data: %v", err)
	}
	// fmt.Println(string(authPayload))
	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(authPayload))
	if err != nil {
		return fmt.Errorf("error creating auth request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err = s.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending auth request: %v", err)
	}
	defer resp.Body.Close()

	if len(resp.Cookies()) == 0 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error reading auth response body: %v", err)
		}
		fmt.Println(resp.StatusCode)
		fmt.Println("headers:")
		for k, v := range resp.Header {
			fmt.Println(k, v)
		}
		fmt.Println(string(body))
		return fmt.Errorf("auth failed: no cookies received, body: %s", string(body))
	}
	// CSRF-Token-EP7GDPV 54SuTFJuHVSWkBxDAFiVSzAbcM2KMTQY4rJ9bRKk5rZXYNbAivYHp2Wstt24if6i
	// sessionid-EP7GDPV 1gXTPCdLnPPnPbFKeLQgQn2F4mxgWkgehYwb2pmfMsQjyDaVNzFKZHdTZgfbjzrT
	const CSRFTokenName = "CSRF-Token" // e.g. CSRF-Token-7APTNV7
	var CSRFTokenHeader string
	for _, cookie := range s.client.Jar.Cookies(req.URL) {
		// fmt.Println(cookie.Name, cookie.Value)
		if strings.HasPrefix(cookie.Name, CSRFTokenName) {
			CSRFTokenHeader = cookie.Value
		}
	}
	if CSRFTokenHeader == "" {
		return fmt.Errorf("CSRF-Token not found in cookiejar")
	}
	s.authPassed = true
	return nil
}

func (s *syncThingConn) FetchDirectories() ([]string, error) {
	if !s.authPassed {
		return nil, fmt.Errorf("not connected, please pass auth first")
	}
	// GET config
	configURL := fmt.Sprintf("%s/rest/config", s.host)
	req, err := http.NewRequest("GET", configURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating config request: %v", err)
	}
	req.Header.Set("Accept", "application/json")

	const CSRFTokenName = "CSRF-Token" // e.g. CSRF-Token-7APTNV7
	for _, cookie := range s.client.Jar.Cookies(req.URL) {
		// fmt.Println(cookie.Name, cookie.Value)
		if strings.HasPrefix(cookie.Name, CSRFTokenName) {
			// e.g. x-csrf-token-7aptnv7
			req.Header.Set("x-"+strings.ToLower(cookie.Name), cookie.Value)
		}
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending config request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading config response body: %v", err)
		}
		return nil, fmt.Errorf("failed to get config with status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var config struct {
		Folders []struct {
			Path string `json:"path"`
		} `json:"folders"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("error decoding config response: %v", err)
	}

	var dirs []string
	for _, folder := range config.Folders {
		dirs = append(dirs, folder.Path)
	}

	return dirs, nil
}

func (s *syncThingConn) ReadPassword(pwdFile ...string) (string, error) {
	if len(pwdFile) > 0 && doraemon.FileIsExist(pwdFile[0]).IsTrue() {
		content, err := os.ReadFile(pwdFile[0])
		if err != nil {
			return "", fmt.Errorf("error reading password file: %v", err)
		}
		return string(content), nil
	}
	const ENV_PASSWORD = "SYNCTHING_PASSWORD"
	password := os.Getenv(ENV_PASSWORD)
	if password != "" {
		return password, nil
	}

	fmt.Print("Enter password: ")
	passwordIn, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("error reading password: %v", err)
	}
	fmt.Println()
	return string(passwordIn), nil
}

func (s *syncThingConn) RestartSyncThing() error {
	restartURL := fmt.Sprintf("%s/rest/system/restart", s.host)
	req, err := http.NewRequest("POST", restartURL, nil)
	if err != nil {
		return fmt.Errorf("error creating restart request: %v", err)
	}
	req.Header.Set("Accept", "application/json")

	const CSRFTokenName = "CSRF-Token" // e.g. CSRF-Token-7APTNV7
	for _, cookie := range s.client.Jar.Cookies(req.URL) {
		// fmt.Println(cookie.Name, cookie.Value)
		if strings.HasPrefix(cookie.Name, CSRFTokenName) {
			// e.g. x-csrf-token-7aptnv7
			req.Header.Set("x-"+strings.ToLower(cookie.Name), cookie.Value)
		}
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending restart request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error reading restart response body: %v", err)
		}
		return fmt.Errorf("failed to restart sync thing with status code: %d, body: %s", resp.StatusCode, string(body))
	}
	return nil
}
