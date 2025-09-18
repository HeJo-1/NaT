// filename: main.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/codingsince1985/geo-golang/openstreetmap"
	"github.com/cozy/goexif2/exif"
	"github.com/gookit/color"
)

// =====================================================================================
// MOD 1: USERNAME SEARCH
// =====================================================================================

type Site struct {
	Name           string
	UrlPattern     string
	NotFoundMarker string
}

type Result struct {
	Site     string `json:"site"`
	Username string `json:"username"`
	URL      string `json:"url"`
	Found    bool   `json:"found"`
	Status   int    `json:"status"`
	Reason   string `json:"reason,omitempty"`
	Duration string `json:"duration_ms"`
}

var sites = []Site{
	{"Facebook", "https://www.facebook.com/%s", "Content not available right now"},
	{"Twitter", "https://twitter.com/%s", "This account doesn’t exist"},
	{"Instagram", "https://www.instagram.com/%s/", "Sorry, this page isn't available."},
	{"GitHub", "https://github.com/%s", "Not Found"},
	{"Reddit", "https://www.reddit.com/user/%s", "page not found"},
	{"YouTube", "https://www.youtube.com/%s", "This channel does not exist."},
	{"TikTok", "https://www.tiktok.com/@%s", "Page not found"},
	{"Medium", "https://medium.com/@%s", "There is no profile for"},
	{"StackOverflow", "https://stackoverflow.com/users/%s", "Page Not Found"},
	{"LinkedIn", "https://www.linkedin.com/in/%s", "Profile not found"},
	{"Pinterest", "https://www.pinterest.com/%s/", "User not found"},
	{"Telegram", "https://t.me/%s", "User does not exist"},
	{"Snapchat", "https://www.snapchat.com/add/%s", "User not found"},
	{"Tumblr", "https://%s.tumblr.com", "There's nothing here."},
	{"Threads", "https://www.threads.net/@%s", "Sorry, this page isn't available."},
	{"Bluesky", "https://bsky.app/profile/%s", "Profile not found"},
	{"Xing", "https://www.xing.com/profile/%s", "This page is unfortunately not available."},
	{"Quora", "https://www.quora.com/profile/%s", "The page you were looking for could not be found"},
	{"Vimeo", "https://vimeo.com/%s", "404 Not Found"},
	{"Twitch", "https://www.twitch.tv/%s", "Sorry. Unless you’ve got a time machine, that content is unavailable."},
	{"SoundCloud", "https://soundcloud.com/%s", "We can't find that user."},
	{"Spotify", "https://open.spotify.com/user/%s", "Page not found"},
	{"Behance", "https://www.behance.net/%s", "Page Not Found"},
	{"Dribbble", "https://dribbble.com/%s", "Whoops, that page is gone."},
	{"ArtStation", "https://www.artstation.com/%s", "Page not found!"},
	{"DeviantArt", "https://www.deviantart.com/%s", "404 Not Found"},
	{"Flickr", "https://www.flickr.com/people/%s/", "Page not found"},
	{"GitLab", "https://gitlab.com/%s", "User not found"},
	{"Steam", "https://steamcommunity.com/id/%s", "The specified profile could not be found."},
	{"Discord", "https://discord.com/users/%s", ""}, // Discord check relies on API/status, not a simple text marker
}

type Job struct {
	Site     Site
	Username string
}

func worker(ctx context.Context, client *http.Client, jobs <-chan Job, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for j := range jobs {
		start := time.Now()
		url := fmt.Sprintf(j.Site.UrlPattern, j.Username)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			results <- Result{Site: j.Site.Name, Username: j.Username, URL: url, Found: false, Status: 0, Reason: err.Error()}
			continue
		}
		req.Header.Set("User-Agent", "username-checker/1.0")

		resp, err := client.Do(req)
		if err != nil {
			results <- Result{Site: j.Site.Name, Username: j.Username, URL: url, Found: false, Status: 0, Reason: err.Error()}
			continue
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		body := strings.ToLower(string(bodyBytes))

		found := false
		reason := ""
		if resp.StatusCode == 200 {
			if j.Site.NotFoundMarker != "" {
				if !strings.Contains(body, strings.ToLower(j.Site.NotFoundMarker)) {
					found = true
				} else {
					reason = "not found marker present"
				}
			} else {
				found = true
			}
		} else if resp.StatusCode == 301 || resp.StatusCode == 302 {
			found = true
			reason = "redirect"
		} else if resp.StatusCode == 404 {
			reason = "not found"
		} else {
			reason = fmt.Sprintf("status %d", resp.StatusCode)
		}

		results <- Result{
			Site:     j.Site.Name,
			Username: j.Username,
			URL:      url,
			Found:    found,
			Status:   resp.StatusCode,
			Reason:   reason,
			Duration: fmt.Sprintf("%d", time.Since(start).Milliseconds()),
		}

		time.Sleep(300 * time.Millisecond) // Be respectful to services
	}
}

// InvertCase swaps the case of each character in a string.
func InvertCase(s string) string {
	var result strings.Builder
	for _, r := range s {
		if unicode.IsUpper(r) {
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(unicode.ToUpper(r))
		}
	}
	return result.String()
}

func runUsernameSearch(username string, concurrency int, timeout int, output string, alternative bool) {
	if username == "" {
		fmt.Println("Please provide a username with the -username parameter.")
		os.Exit(1)
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	ctx := context.Background()

	jobs := make(chan Job)
	results := make(chan Result)
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go worker(ctx, client, jobs, results, &wg)
	}

	go func() {
		for _, s := range sites {
			jobs <- Job{Site: s, Username: username}
		}
		if alternative {
			invertedUsername := InvertCase(username)
			for _, s := range sites {
				jobs <- Job{Site: s, Username: invertedUsername}
			}
		}
		close(jobs)
	}()

	var resList []Result
	done := make(chan struct{})
	go func() {
		for r := range results {
			resList = append(resList, r)
		}
		done <- struct{}{}
	}()

	wg.Wait()
	close(results)
	<-done

	file, err := os.Create(output)
	if err != nil {
		fmt.Println("Could not create file:", err)
		return
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	_ = enc.Encode(resList)

	color.Red.Println(`
 ███▄    █  ▄▄▄     ▄▄▄█████▓    ▒█████    ██████  ██▓ ███▄    █ ▄▄▄█████▓
 ██ ▀█   █ ▒████▄   ▓  ██▒ ▓▒   ▒██▒  ██▒▒██    ▒ ▓██▒ ██ ▀█   █ ▓  ██▒ ▓▒
▓██  ▀█ ██▒▒██  ▀█▄ ▒ ▓██░ ▒░   ▒██░  ██▒░ ▓██▄   ▒██▒▓██  ▀█ ██▒▒ ▓██░ ▒░
▓██▒  ▐▌██▒░██▄▄▄▄██░ ▓██▓ ░    ▒██   ██░  ▒   ██▒░██░▓██▒  ▐▌██▒░ ▓██▓ ░ 
▒██░   ▓██░ ▓█   ▓██▒ ▒██▒ ░    ░ ████▓▒░▒██████▒▒░██░▒██░   ▓██░  ▒██▒ ░ 
░ ▒░   ▒ ▒  ▒▒   ▓▒█░ ▒ ░░      ░ ▒░▒░▒░ ▒ ▒▓▒ ▒ ░░▓  ░ ▒░   ▒ ▒   ▒ ░░   
░ ░░   ░ ▒░  ▒   ▒▒ ░   ░         ░ ▒ ▒░ ░ ░▒  ░ ░ ▒ ░░ ░░   ░ ▒░    ░    
   ░   ░ ░   ░   ▒    ░         ░ ░ ░ ▒  ░  ░  ░   ▒ ░   ░   ░ ░   ░      
         ░       ░  ░               ░ ░        ░   ░           ░          
         `)

	madebyhejo := color.New(color.FgWhite, color.OpBold)
	madebyhejo.Println("Made By HeJo")
	println("")
	println("")

	usernamesToDisplay := []string{username}
	if alternative {
		usernamesToDisplay = append(usernamesToDisplay, InvertCase(username))
	}
	fmt.Printf("Searching for usernames: %s\n", strings.Join(usernamesToDisplay, ", "))

	foundAccounts := make(map[string][]string)
	for _, r := range resList {
		if r.Found {
			foundAccounts[r.Username] = append(foundAccounts[r.Username], fmt.Sprintf("- %-12s %s", r.Site, r.URL))
		}
	}

	for _, u := range usernamesToDisplay {
		if accounts, ok := foundAccounts[u]; ok {
			fmt.Printf("\nAccounts found for '%s':\n", u)
			for _, acc := range accounts {
				fmt.Println(acc)
			}
		} else {
			fmt.Printf("\nNo accounts found for '%s'.\n", u)
		}
	}

	fmt.Printf("\nAll results have been saved to '%s'.\n", output)
}

// =====================================================================================
// MOD 2: WEBSITE TEXT SIMILARITY
// =====================================================================================

func getTextFromURL(url string) (string, error) {
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return "", fmt.Errorf("request failed, status code: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", err
	}

	// Remove script and style elements
	doc.Find("script, style").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	text := doc.Find("body").Text()
	return strings.TrimSpace(text), nil
}

func textToWordSet(text string) map[string]struct{} {
	lowerText := strings.ToLower(text)
	// Regex to split by non-letter and non-number characters
	re := regexp.MustCompile(`[^\p{L}\p{N}]+`)
	words := re.Split(lowerText, -1)

	wordSet := make(map[string]struct{})
	for _, word := range words {
		if word != "" {
			wordSet[word] = struct{}{}
		}
	}
	return wordSet
}

func calculateJaccardSimilarity(setA, setB map[string]struct{}) float64 {
	intersection := 0
	for word := range setA {
		if _, found := setB[word]; found {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0.0
	}
	return float64(intersection) / float64(union)
}

func runWebSimilarity(urlsString string) {
	urls := strings.Split(urlsString, ",")
	if len(urls) < 2 || urlsString == "" {
		fmt.Println("Please provide at least 2 URLs to compare using the -urls parameter (comma-separated).")
		os.Exit(1)
	}

	siteContents := make(map[string]string)

	fmt.Println("Fetching text from websites...")
	for _, url := range urls {
		trimmedURL := strings.TrimSpace(url)
		text, err := getTextFromURL(trimmedURL)
		if err != nil {
			log.Printf("Could not fetch data from %s: %v\n", trimmedURL, err)
			continue
		}
		siteContents[trimmedURL] = text
		fmt.Printf("✓ Fetched text from %s.\n", trimmedURL)
	}
	fmt.Println("-------------------------------------------")

	for i := 0; i < len(urls); i++ {
		for j := i + 1; j < len(urls); j++ {
			url1 := strings.TrimSpace(urls[i])
			url2 := strings.TrimSpace(urls[j])

			text1, ok1 := siteContents[url1]
			text2, ok2 := siteContents[url2]
			if !ok1 || !ok2 {
				continue // Skip if fetching failed for one of the URLs
			}

			wordSet1 := textToWordSet(text1)
			wordSet2 := textToWordSet(text2)
			similarity := calculateJaccardSimilarity(wordSet1, wordSet2)
			fmt.Printf("Text similarity between '%s' and '%s': %.2f%%\n", url1, url2, similarity*100)
		}
	}
}

// =====================================================================================
// MOD 3: REVERSE IMAGE SEARCH (LENS)
// =====================================================================================

func reverseImageSearch(filePath string) ([]string, error) {
	url := "https://lens.google.com/uploadbyurl?url=" // This URL is a placeholder and might not work as expected
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("encoded_image", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("could not create form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return nil, fmt.Errorf("could not copy file content: %w", err)
	}

	writer.Close()

	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		return nil, fmt.Errorf("could not create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not parse HTML: %w", err)
	}

	var links []string
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		link, exists := s.Attr("href")
		if exists && strings.HasPrefix(link, "http") {
			links = append(links, link)
		}
	})

	return links, nil
}

func runLensSearch(imagePath string) {
	if imagePath == "" {
		fmt.Println("Please provide an image file path with the -image parameter.")
		os.Exit(1)
	}

	links, err := reverseImageSearch(imagePath)
	if err != nil {
		log.Fatalf("An error occurred during the search: %v", err)
	}

	if len(links) == 0 {
		fmt.Println("No results found for the image.")
		return
	}

	fmt.Println("Found Result Links:")
	for _, link := range links {
		fmt.Println(link)
	}
}

// =====================================================================================
// MOD 4: GEOLOCATION FROM IMAGE
// =====================================================================================

func runGeoFromImage(filePath string) {
	if filePath == "" {
		fmt.Println("Please provide an image file path with the -image parameter.")
		os.Exit(1)
	}

	f, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Could not open file: %v\n", err)
		return
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		fmt.Printf("Could not read Exif data: %v\n", err)
		return
	}

	lat, long, err := x.LatLong()
	if err != nil {
		fmt.Printf("No GPS data found in this photo: %v\n", err)
		return
	}

	fmt.Printf("Coordinates Found -> Latitude: %f, Longitude: %f\n", lat, long)

	geocoder := openstreetmap.Geocoder()
	address, err := geocoder.ReverseGeocode(lat, long)
	if err != nil {
		fmt.Printf("Could not retrieve address information: %v\n", err)
		return
	}

	fmt.Printf("Estimated Location: %s\n", address.FormattedAddress)
}

// =====================================================================================
// MAIN FUNCTION
// =====================================================================================

func main() {
	mode := flag.String("mode", "username", "Execution mode (username, websimilarity, lens, geo)")
	username := flag.String("username", "", "Username to search for")
	concurrency := flag.Int("c", 6, "Number of concurrent requests")
	timeout := flag.Int("t", 10, "Timeout for each request (seconds)")
	output := flag.String("o", "results.json", "File to save the results")
	alternative := flag.Bool("a", false, "Performs an additional search by inverting the case of the username")
	urls := flag.String("urls", "", "URLs to compare (comma-separated)")
	image := flag.String("image", "", "File path of the photo to process (for lens, geo modes)")
	flag.Parse()

	switch *mode {
	case "username":
		runUsernameSearch(*username, *concurrency, *timeout, *output, *alternative)
	case "websimilarity":
		runWebSimilarity(*urls)
	case "lens":
		runLensSearch(*image)
	case "geo":
		runGeoFromImage(*image)
	default:
		fmt.Println("Invalid mode selected. Please use one of the following: username, websimilarity, lens, geo")
		flag.Usage()
		os.Exit(1)
	}
}
