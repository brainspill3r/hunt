package main

import (
    "bufio"
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"
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

// executeCommand runs a command and handles errors, capturing its output
func executeCommand(cmd *exec.Cmd) (string, error) {
    var out bytes.Buffer
    cmd.Stdout = &out
    cmd.Stderr = &out
    err := cmd.Run()
    return out.String(), err
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
        body, _ := ioutil.ReadAll(resp.Body)
        return fmt.Errorf("unexpected response from Discord: %s", body)
    }

    return nil
}

// showProgress shows progress while a long-running command is executed
func showProgress(done chan bool) {
    delay := 30 * time.Second
    ticker := time.NewTicker(delay)
    defer ticker.Stop()
    for {
        select {
        case <-done:
            return
        case <-ticker.C:
            fmt.Println("Open Redirects check in progress, please wait...")
        }
    }
}

// isRedirect checks if the response is a redirect to the specified target domain
func isRedirect(resp *http.Response, targetDomain string) bool {
    if resp.StatusCode >= 300 && resp.StatusCode < 400 {
        location := resp.Header.Get("Location")
        return strings.Contains(location, targetDomain)
    }
    return false
}

// checkOpenRedirect checks for open redirects with improved validation
func checkOpenRedirect(urlsFile, outputFile string, done chan bool) {
    fmt.Println("Checking for open redirects...")
    file, err := os.Open(urlsFile)
    if err != nil {
        log.Fatalf("Failed to open urls.txt: %v", err)
    }
    defer file.Close()

    var urls []string
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.Contains(strings.ToLower(line), "=http") {
            urls = append(urls, line)
        }
    }
    if err := scanner.Err(); err != nil {
        log.Fatalf("Failed to read urls.txt: %v", err)
    }

    var vulnerableUrls []string
    for _, url := range urls {
        replacedUrl := strings.ReplaceAll(url, "=", "=https://www.google.com")
        // Follow the redirect to the final destination
        resp, err := http.Get(replacedUrl)
        if err != nil {
            log.Printf("Error fetching URL: %v\n", err)
            continue
        }
        defer resp.Body.Close()

        if isRedirect(resp, "www.google.com") {
            fmt.Printf("\033[32mVulnerable: %s\n\033[0m", replacedUrl) // Green color for vulnerable URLs
            vulnerableUrls = append(vulnerableUrls, replacedUrl)
        }
    }

    if len(vulnerableUrls) > 0 {
        if err := ioutil.WriteFile(outputFile, []byte(strings.Join(vulnerableUrls, "\n")), 0644); err != nil {
            log.Fatalf("Failed to write to %s: %v", outputFile, err)
        }
        fmt.Println("Open redirect findings saved to open_redirects.txt.")
    } else {
        fmt.Println("No open redirects found.")
    }

    done <- true
}

func main() {
    remindToStartDocker()

    if len(os.Args) != 3 {
        fmt.Println("Usage: go run open-redirect-check-service.go <domain> <program>")
        fmt.Println("Programs: Bugcrowd, HackerOne, Intigriti, Synack, YesWeHack")
        os.Exit(1)
    }

    domain := os.Args[1]
    program := os.Args[2]
    outputBaseDir := "/home/brainspiller/Documents/hunt"
    discordWebhookURL := "https://discord.com/api/webhooks/1260862675500666910/fJcZBC6DyJe8cGJarMhzO9sIu9EcSugRDRZMcWeoD9wc_Ht8YSHvnvx4gZCoyCoOS2NO"

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
    go showProgress(done)

    <-done

    message := fmt.Sprintf("Bug bounty - **Open Redirect Check** has completed for **%s** on **%s**. Check your **open_redirects.txt** to see the results.", domain, program)
    if err := sendDiscordNotification(discordWebhookURL, message); err != nil {
        log.Fatalf("Failed to send Discord notification: %v", err)
    }
}
