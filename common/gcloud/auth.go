package gcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

var (
	ch chan string
)

func AuthHelper(redirectPath string, redirectPort int, conf *Config) error {
	ch = make(chan string)
	http.HandleFunc(redirectPath, handleGoogleCallback)

	go func() {
		port := strconv.Itoa(redirectPort)
		fmt.Println("Starting listen on http://localhost:" + port)
		err := http.ListenAndServe(":"+port, nil)
		if err != nil {
			logrus.Warnf("Failed to listen %s: %v", port, err)
		}
	}()

	b, err := os.ReadFile(conf.CredFilename)
	if err != nil {
		return err
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		return fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	_, err = tokenFromFile(conf.TokenFilename)
	if err == nil {
		fmt.Println("Token file exists already. Skip")
		return nil
	}

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then follow the instruction: "+
		"\n%v\n", authURL)

	authCode := <-ch

	fmt.Printf("Start exchange: %s\n", authCode)
	token, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		return err
	}

	fmt.Printf("Exchange success, saving token into %s\n", conf.TokenFilename)

	return saveToken(conf.TokenFilename, token)
}

func handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Handle google callback: %s\n", r.URL)

	code := r.FormValue("code")
	if code == "" {
		return
	}

	ch <- code
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}
