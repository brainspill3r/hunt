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

var uniqueMessages = make(map[string]bool) // A map to track unique vulnerable URLs

// logUniqueMessage logs a message only if it hasn't been logged before
func logUniqueMessage(message string) {
    if _, exists := uniqueMessages[message]; !exists {
        fmt.Printf("\033[32m%s\033[0m\n", message) // Green text for unique messages
        uniqueMessages[message] = true
    }
}

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
    log.Println("Checking for open redirects...") // Now logs to file as well
    file, err := os.Open(urlsFile)
    if err != nil {
        log.Fatalf("Failed to open urls.txt: %v", err)
    }
    defer file.Close()

    var urls []string
    scanner := bufio.NewScanner(file)

    // List of potential redirect parameters
    possibleRedirectParams := []string{
        "redirect", "redirect_url", "redirect_uri", "redirectUrl", "redirectUri",
        "redir", "return", "returnTo", "return_url", "return_uri", "next", "url",
        "goto", "destination", "forward", "forwardTo", "callback", "target", "rurl", "dest",
        "success", "return_path", "continue", "page", "service", "origin", "originUrl",
    }

    for scanner.Scan() {
        line := scanner.Text()

        // Check if any of the redirect parameters are in the URL
        for _, param := range possibleRedirectParams {
            if strings.Contains(strings.ToLower(line), param+"=") {
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
            if strings.Contains(url, redirectParam+"=") {
                // Find the position of the redirect parameter and replace its value
                startIdx := strings.Index(url, redirectParam+"=")
                if startIdx != -1 {
                    // Construct the URL up to the redirect parameter and then append the new value
                    replacedUrl := url[:startIdx+len(redirectParam)+1] + "https://www.google.com"

                    // Fetch the URL using the client
                    resp, err := client.Get(replacedUrl)
                    if err != nil {
                        log.Printf("Error fetching URL: %v, skipping...\n", err)
                        found = true
                        break // Exit the inner loop once a redirect is found
                    }
                    defer resp.Body.Close()

                    if isRedirect(resp, "www.google.com") {
                        logUniqueMessage(fmt.Sprintf("Vulnerable: %s", replacedUrl)) // Log unique vulnerable URLs
                        vulnerableUrls = append(vulnerableUrls, replacedUrl)
                        found = true
                        break // Exit the inner loop once a redirect is found
                    }
                }
            }
        }

        if found {
            continue // Move to the next URL once a redirect is found and handled
        }
    }

    if len(vulnerableUrls) > 0 {
        if err := os.WriteFile(outputFile, []byte(strings.Join(vulnerableUrls, "\n")), 0644); err != nil {
            log.Fatalf("Failed to write to %s: %v", outputFile, err)
        }
        log.Println("Open redirect findings saved to", outputFile)
    } else {
        log.Println("No open redirects found.")
    }

    done <- true
}

func main() {
     //Set up logging to both a log file and the console
     logFile, err := os.OpenFile("/home/brainspiller/Documents/hunt/logs/open_redirect.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
     if err != nil {
         log.Fatalf("Failed to create log file: %v", err)
     }
     defer logFile.Close()

    // // Create a multi-writer to write to both the file and console
    // mw := io.MultiWriter(os.Stdout, logFile)
    // log.SetOutput(mw)

    // Load environment variables from the .discordhooks.env file
    Utils.LoadEnv("/home/brainspiller/Documents/hunt/.discordhooks.env")

    // Get the Discord webhook URL from environment variables
    discordWebhookURL := Utils.GetOpenRedirectWebhook()

    remindToStartDocker()

    if len(os.Args) != 3 {
        log.Println("Usage: go run open-redirect-check-service.go <domain> <program>")
        log.Println("Programs: Bugcrowd, HackerOne, Intigriti, Synack, YesWeHack")
        os.Exit(1)
    }

    domain := os.Args[1]
    program := os.Args[2]
    outputBaseDir := "/home/brainspiller/Documents/hunt"

    if !validateProgram(program) {
        log.Println("Invalid program. Choose from: Bugcrowd, HackerOne, Intigriti, Synack, YesWeHack")
        os.Exit(1)
    }

    outputDir := filepath.Join(outputBaseDir, program, domain)

    log.Printf("Changing to output directory: %s\n", outputDir)
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
