package main

import (
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
    "sync"
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

// getUserAgent returns the user-agent based on the program
func getUserAgent(program string) string {
    switch program {
    case "Bugcrowd":
        return "User-Agent:Bugcrowd:brainspiller"
    case "HackerOne":
        return "User-Agent:HackerOne:brainspiller"
    case "Intigriti":
        return "User-Agent:Intigriti:brainspiller"
    case "Synack":
        return "User-Agent:Synack:brainspiller"
    case "YesWeHack":
        return "User-Agent:YesWeHack:brainspiller"
    default:
        return ""
    }
}

// executeCommand runs a command and handles errors, capturing its output
func executeCommand(cmd *exec.Cmd) {
    fmt.Printf("\033[33mExecuting command: %s\033[0m\n", strings.Join(cmd.Args, " ")) // Orange color for executing commands
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil {
        log.Fatalf("Command failed: %v", err)
    }
}

// executeCommandWithOutput runs a command and captures its output
func executeCommandWithOutput(cmd *exec.Cmd) (string, error) {
    fmt.Printf("\033[33mExecuting command: %s\033[0m\n", strings.Join(cmd.Args, " ")) // Orange color for executing commands
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

func main() {
    remindToStartDocker()

    if len(os.Args) != 3 {
        fmt.Println("Usage: go run param-js-extraction.go <domain> <program>")
        fmt.Println("Programs: Bugcrowd, HackerOne, Intigriti, Synack, YesWeHack")
        os.Exit(1)
    }

    domain := os.Args[1]
    program := os.Args[2]
    outputBaseDir := "/home/brainspiller/Documents/hunt"
    discordWebhookURL := "https://discord.com/api/webhooks/1260557120491753602/uoIu7BKwTE0zovLfyYln6x17Or0qVWoEU11plZg02EgYkr37MMYHiEIIv87QiLK0RYxM"

    if !validateProgram(program) {
        fmt.Println("Invalid program. Choose from: Bugcrowd, HackerOne, Intigriti, Synack, YesWeHack")
        os.Exit(1)
    }

    userAgent := getUserAgent(program)
    outputDir := filepath.Join(outputBaseDir, program, domain)

    fmt.Printf("Changing to output directory: %s\n", outputDir)
    if err := os.Chdir(outputDir); err != nil {
        log.Fatalf("Failed to change directory: %v", err)
    }

    urlsFile := filepath.Join(outputDir, "urls.txt")
    if _, err := os.Stat(urlsFile); os.IsNotExist(err) {
        log.Fatalf("urls.txt does not exist in the specified directory")
    }

    jsFile := filepath.Join(outputDir, "js.txt")
    paramFile := filepath.Join(outputDir, "param.txt")

    // Generate param.txt and js.txt in parallel
    fmt.Println("Processing URLs for parameters and JavaScript files...")
    var wg sync.WaitGroup
    wg.Add(2)

    go func() {
        defer wg.Done()
        grepParamCmd := exec.Command("sh", "-c", fmt.Sprintf("cat %s | grep '=' | tee %s", urlsFile, paramFile))
        executeCommand(grepParamCmd)
    }()

    go func() {
        defer wg.Done()
        grepJsCmd := exec.Command("sh", "-c", fmt.Sprintf("cat %s | grep -iE '.js' | grep -ivE '.json' | sort -u | tee %s", urlsFile, jsFile))
        executeCommand(grepJsCmd)
    }()

    wg.Wait()

    fmt.Printf("User-Agent: %s\n", userAgent)

    fmt.Println("\033[31mParameter and JS File Extraction completed\033[0m")

    message := fmt.Sprintf("Bug bounty - **Parameter and JS File Extraction Service** has completed for **%s** on **%s**. Check your **param.txt** and **js.txt** to see the results.", domain, program)
    if err := sendDiscordNotification(discordWebhookURL, message); err != nil {
        log.Fatalf("Failed to send Discord notification: %v", err)
    }
}
