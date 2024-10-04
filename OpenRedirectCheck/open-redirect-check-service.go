package main

import (
    "bufio"
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"
    Utils "open-redirect-check-service/Utils"
)

// remindToStartDocker checks if Docker is running and reminds the user to start it if not
func remindToStartDocker() {
    cmd := exec.Command("docker", "info")
    err := cmd.Run()
    if err != nil {
        fmt.Println("Have you remembered to start Docker? If not, cancel the script and run 'sudo dockerd' in another terminal.")
    }
}

// validateProgram checks if the provided program is valid
func validateProgram(program string) bool {
    validPrograms := []string{"Bugcrowd", "HackerOne", "Intigriti", "Synack", "YesWeHack"}
    for _, validProgram := range validPrograms {
        if program == validProgram {
            return true
        }
    }
    return false
}

// sendDiscordNotification sends a notification to a Discord webhook
func sendDiscordNotification(webhookURL, message string) error {
    payload := map[string]string{"content": message}
    jsonData, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("failed to marshal payload: %v", err)
    }

    resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return fmt.Errorf("failed to send POST request: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusNoContent {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("unexpected response from Discord: %s", body)
    }

    return nil
}

// isRedirect checks if the response is a redirect to the specified target domain
func isRedirect(resp *http.Response, targetDomain string) bool {
    if resp.StatusCode >= 300 && resp.StatusCode < 400 {
        location := resp.Header.Get("Location")
        fmt.Printf("Found redirect: %s\n", location) // Debugging output
        return strings.Contains(location, targetDomain)
    }
    return false
}

// checkOpenRedirect checks for open redirects
func checkOpenRedirect(urlsFile, outputFile string, done chan bool) {
    fmt.Println("Checking for open redirects...")

    file, err := os.Open(urlsFile)
    if err != nil {
        log.Fatalf("Failed to open %s: %v", urlsFile, err)
    }
    defer file.Close()

    var urls []string
    scanner := bufio.NewScanner(file)

    // List of potential redirect parameters and paths
    possibleRedirectParams := []string{
        "redirect", "redirect_url", "redirect_uri", "redirectUrl", "redirectUri",
        "redir", "return", "returnTo", "return_url", "return_uri", "next", "url",
        "goto", "destination", "forward", "forwardTo", "callback", "target", "rurl",
        "dest", "view", "checkout_url", "continue", "success", "data", "qurl",
        "login", "logout", "ext", "clickurl", "rit_url", "forward_url", "action",
        "action_url", "sp_url", "recurl", "service", "u", "link", "src", "origin",
        "originUrl", "jump", "jump_url", "callback_url", "page", "backurl", "burl",
        "request", "ReturnUrl", "pic", "tc?src", "allinurl", "j?url", "go", "view",
        "image_url", "out", "/redirect", "/cgi-bin/redirect.cgi", "/out", "/login?to",
        "return_path", "click?u", "url", "uri", "linkAddress", "location", "q",
    }

    // Read the URLs and filter by redirect parameters
    for scanner.Scan() {
        line := scanner.Text()
        for _, param := range possibleRedirectParams {
            if strings.Contains(strings.ToLower(line), param+"=") || strings.Contains(line, param) {
                urls = append(urls, line)
                break
            }
        }
    }
    if err := scanner.Err(); err != nil {
        log.Fatalf("Failed to read urls.txt: %v", err)
    }

    // Disable redirection in the HTTP client and manually handle Location header
    client := &http.Client{
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            return http.ErrUseLastResponse // Prevent automatic redirects
        },
        Timeout: 10 * time.Second, // Timeout set to 10 seconds
    }

    var vulnerableUrls []string
    for _, url := range urls {
        found := false
        for _, redirectParam := range possibleRedirectParams {
            if strings.Contains(url, redirectParam) {
                // Replace the redirect URL value with `https://www.google.com`
                replacedUrl := strings.Replace(url, redirectParam+"=", redirectParam+"=https://www.google.com", 1)

                // Fetch the URL using the global client
                resp, err := client.Get(replacedUrl)
                if err != nil {
                    log.Printf("Error fetching URL: %v, skipping...\n", err)
                    found = true
                    break // Exit the inner loop once a redirect is found
                }
                defer resp.Body.Close()

                // Check if the Location header redirects to https://www.google.com
                if isRedirect(resp, "www.google.com") {
                    fmt.Printf("\033[32mVulnerable: %s\n\033[0m", replacedUrl) // Green color for vulnerable URLs
                    vulnerableUrls = append(vulnerableUrls, replacedUrl)
                    found = true
                    break // Exit the inner loop once a redirect is found
                }
            }
        }
        if found {
            continue // Move to the next URL once a redirect is found and handled
        }
    }

    // Output the results
    if len(vulnerableUrls) > 0 {
        if err := os.WriteFile(outputFile, []byte(strings.Join(vulnerableUrls, "\n")), 0644); err != nil {
            log.Fatalf("Failed to write to %s: %v", outputFile, err)
        }
        fmt.Println("Open redirect findings saved to", outputFile)
    } else {
        fmt.Println("No open redirects found.")
    }

    done <- true
}

func main() {
    // Load environment variables from the .discordhooks.env file
    Utils.LoadEnv("/home/brainspiller/Documents/hunt/.discordhooks.env")

    // Get the Discord webhook URL from environment variables
    discordWebhookURL := Utils.GetOpenRedirectWebhook()

    remindToStartDocker()

    if len(os.Args) != 3 {
        fmt.Println("Usage: go run open-redirect-check-service.go <domain> <program>")
        fmt.Println("Programs: Bugcrowd, HackerOne, Intigriti, Synack, YesWeHack")
        os.Exit(1)
    }

    domain := os.Args[1]
    program := os.Args[2]
    outputBaseDir := "/home/brainspiller/Documents/hunt"

    if !validateProgram(program) {
        fmt.Println("Invalid program. Choose from: Bugcrowd, HackerOne, Intigriti, Synack, YesWeHack")
        os.Exit(1)
    }

    outputDir := filepath.Join(outputBaseDir, program, domain)

    fmt.Printf("Changing to output directory: %s\n", outputDir)
    if err := os.Chdir(outputDir); err != nil {
        log.Fatalf("Failed to change directory: %v", err)
    }

    urlsFile := filepath.Join(outputDir, "urls.txt")
    if _, err := os.Stat(urlsFile); os.IsNotExist(err) {
        log.Fatalf("urls.txt does not exist in the specified directory")
    }

    outputFile := filepath.Join(outputDir, "open_redirects.txt")

    done := make(chan bool)
    go checkOpenRedirect(urlsFile, outputFile, done)

    <-done

    message := fmt.Sprintf("Bug bounty - **Open Redirect Check** has completed for **%s** on **%s**. Check your **open_redirects.txt** to see the results.", domain, program)
    if err := sendDiscordNotification(discordWebhookURL, message); err != nil {
        log.Fatalf("Failed to send Discord notification: %v", err)
    }
}
