package main

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	CalendarID     string `json:"calendar_id"`
	DefaultMessage string `json:"default_message"`
	User           string `json:"user"`
}

func getConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Unable to find user home directory: %v", err)
	}
	return filepath.Join(homeDir, ".wfh")
}

func getClient(config *oauth2.Config, tokenPath string) *calendar.Service {
	tok, err := tokenFromFile(tokenPath)
	if err != nil {
		tok = getTokenFromWeb(config, tokenPath)
	}

	client := config.Client(context.Background(), tok)

	srv, err := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}
	return srv
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config, tokenPath string) *oauth2.Token {
	// We'll use a channel to block until we get the authorization code
	codeCh := make(chan string)

	// Start a local server to listen on a specified port
	srv := &http.Server{Addr: ":8066"}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		fmt.Fprintln(w, "Received authentication code. You can close this page now.") // nolint: errcheck
		codeCh <- code                                                                // Send code to our waiting getTokenFromWeb function
	})

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// Here, set your redirect URL to `http://localhost:8066/`
	// This should match one of the URIs you set in your Google Developer Console
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("redirect_uri", "http://localhost:8066/"))
	fmt.Printf("Go to the following link in your browser:\n%v\n", authURL)

	// Block until we receive the code
	authCode := <-codeCh
	fmt.Println("code received from channel")
	// Shutdown the server

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel() // Cancel context when done to release resources

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server Shutdown: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode,
		oauth2.SetAuthURLParam("redirect_uri", "http://localhost:8066/"))
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}

	err = saveToken(tokenPath, tok)
	if err != nil {
		log.Fatalf("Unable to save token: %v", err)
	}

	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	fmt.Printf("Loading token from file: %s\n", file)
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	if err != nil {
		log.Fatalf("Unable to decode token: %v", err)
	}
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("os.Create: %w", err)
	}

	err = json.NewEncoder(f).Encode(token)
	if err != nil {
		return fmt.Errorf("json.NewEncoder.Encode: %w", err)
	}
	err = f.Close()
	if err != nil {
		return fmt.Errorf("f.Close: %w", err)
	}
	return nil
}

func main() {
	configPath := getConfigPath()

	// Check if gconfig directory exists, if not, create it.
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		err := os.Mkdir(configPath, os.ModePerm)
		if err != nil {
			log.Fatalf("Unable to create config directory: %v", err)
		}
	}

	credentialsPath := filepath.Join(configPath, "credentials.json")
	tokenPath := filepath.Join(configPath, "token.json")

	// Load client secrets from a file.
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	gconfig, err := google.ConfigFromJSON(b, calendar.CalendarEventsScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to gconfig: %v", err)
	}
	srv := getClient(gconfig, tokenPath)

	// load the config file:
	config, err := getConfig(configPath)
	if err != nil {
		log.Fatalf("Unable to load config file: %v", err)
	}
	user := getUser(config.User)
	if user == "" {
		log.Fatal("Unable to determine user, please set $USER or add user to config file")
	}
	now := time.Now().Format("2006-01-02") // This will format the time as "yyyy-mm-dd"
	message := config.DefaultMessage
	if message == "" {
		message = "%s - working from home"
	}
	event := &calendar.Event{
		Summary: fmt.Sprintf(message, user),
		Start: &calendar.EventDateTime{
			Date:     now,
			TimeZone: "UTC",
		},
		End: &calendar.EventDateTime{
			Date:     now,
			TimeZone: "UTC",
		},
	}
	fmt.Printf("Event: %+v\n", event)
	fmt.Printf("Creating event: %s %s\n", event.Summary, now)

	event, err = srv.Events.Insert(config.CalendarID, event).Do()
	if err != nil {
		log.Fatalf("Unable to create event. %v\n", err)
	}
	fmt.Printf("Event created: %s\n", event.HtmlLink)
}

func getConfig(path string) (Config, error) {
	var config Config
	configPath := filepath.Join(path, "config.json")
	b, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("os.ReadFile(%s): %w", configPath, err)
	}

	err = json.Unmarshal(b, &config)
	if err != nil {
		return Config{}, fmt.Errorf("json.Unmarshal(%s): %w", configPath, err)
	}

	return config, nil
}

func getUser(user string) string {
	if user == "" {
		return os.Getenv("USER")
	}
	return user
}
