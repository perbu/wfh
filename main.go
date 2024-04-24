package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

//go:embed credentials.json
var googleCredentials []byte

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
	if tok != nil {
		if len(tok.RefreshToken) == 0 {
			log.Println("No refresh token found, please delete token.json, revoke the token and try again.")
		}
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
	// make a state token to prevent CSRF attacks:
	state := randomString(16)
	// We'll use a channel to block until we get the authorization code
	codeCh := make(chan string)

	// Start a local server to listen on a specified port
	srv := &http.Server{Addr: ":8066"}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		recvState := r.URL.Query().Get("state")
		if recvState != state {
			_, _ = fmt.Fprintf(w, "Invalid state: %s\n", recvState) // nolint: errcheck
			return
		}
		_, _ = fmt.Fprintln(w, "Received authentication code. You can close this page now.") // nolint: errcheck
		codeCh <- code                                                                       // Send code to our waiting getTokenFromWeb function
	})

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// Here, set your redirect URL to `http://localhost:8066/`
	// This should match one of the URIs you set in your Google Developer Console
	authURL := config.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("redirect_uri", "http://localhost:8066/"),
	)
	fmt.Printf("Go to the following link in your browser:\n%v\n", authURL)

	// Block until we receive the code
	authCode := <-codeCh
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

// randomString returns a random string of the specified length, using A-Z, a-z, 0-9
func randomString(i int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, i)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close() // nolint: errcheck
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	if err != nil {
		log.Fatalf("Unable to decode token: %v", err)
	}
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) error {
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
	tokenPath := filepath.Join(configPath, "token.json")

	// If modifying these scopes, delete your previously saved token.json.
	gconfig, err := google.ConfigFromJSON(googleCredentials, calendar.CalendarEventsScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to gconfig: %v", err)
	}
	calService := getClient(gconfig, tokenPath)
	// load the config file:
	config, err := getConfig(configPath)
	if err != nil {
		log.Fatalf("Unable to load config file: %v", err)
	}
	listAction, date, message, err := parseArgs(config.DefaultMessage)
	if err != nil {
		fmt.Printf("while parsing arguments and flags: %v\n", err)
		os.Exit(1)
	}
	if listAction {
		// just list the events and then exit.
		listEvents(calService, config, date)
		os.Exit(0)
	}
	// pick a random number from 1 to 11:
	colorId := rand.Intn(11) + 1
	event := &calendar.Event{
		ColorId: strconv.Itoa(colorId),
		Summary: message,
		Start: &calendar.EventDateTime{
			Date:     date.Format("2006-01-02"),
			TimeZone: "UTC",
		},
		End: &calendar.EventDateTime{
			Date:     date.Format("2006-01-02"),
			TimeZone: "UTC",
		},
	}

	event, err = calService.Events.Insert(config.CalendarID, event).Do()
	if err != nil {
		log.Fatalf("Unable to create event. %v\n", err)
	}
	fmt.Printf("Event created: %s\nLink %s\n", event.Summary, event.HtmlLink)
}

// listEvents lists the events for the given date.
func listEvents(service *calendar.Service, config Config, date time.Time) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
	endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 23, 0, 0, 0, time.Local)
	fmt.Printf("listing events for %s to %s\n", startOfDay.Format(time.RFC3339), endOfDay.Format(time.RFC3339))
	events, err := service.Events.List(config.CalendarID).
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(startOfDay.Format(time.RFC3339)).
		TimeMax(endOfDay.Format(time.RFC3339)).
		OrderBy("startTime").
		Do()
	if err != nil {
		log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
	}
	if len(events.Items) == 0 {
		fmt.Println("No events found.")
	} else {
		fmt.Println("Events:")
		for _, item := range events.Items {
			timeString := "(all day)"
			if item.Start.DateTime != "" {
				timeString = fmt.Sprintf("(%v --> %v)", item.Start.DateTime, item.End.DateTime)
			}
			fmt.Printf("%v %s [%s]\n", item.Summary, timeString, shortEmail(item.Creator.Email))
		}
	}
}

func shortEmail(email string) string {
	atIndex := len(email)
	for i, c := range email {
		if c == '@' {
			atIndex = i
			break
		}
	}
	return email[:atIndex]

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

func parseArgs(defaultMessage string) (bool, time.Time, string, error) {
	// Define flags for the date and message arguments with default values of empty strings.
	dateFlag := flag.String("date", "", "Provide a date in the format YYYY-MM-DD")
	messageFlag := flag.String("message", "", "Provide a custom message")
	list := flag.Bool("list", false, "List all events")

	// Parse the flags
	flag.Parse()
	// Check if there are any non-flag arguments and fail if there are
	if len(flag.Args()) > 0 {
		return false, time.Time{}, "", fmt.Errorf("unexpected non-flag arguments detected")
	}

	// Parse the date if provided
	var parsedDate time.Time
	if *dateFlag != "" {
		var err error
		parsedDate, err = time.Parse("2006-01-02", *dateFlag)
		if err != nil {
			// use today's date if the provided date is invalid
			parsedDate = time.Now()
		}
	} else {
		// use today's date if no date is provided
		parsedDate = time.Now()
	}
	if *list {
		return true, parsedDate, "", nil
	}
	var message string
	if *messageFlag != "" {
		message = *messageFlag
	} else {
		message = defaultMessage
	}
	return false, parsedDate, message, nil
}
