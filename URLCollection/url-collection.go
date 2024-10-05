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
        fmt.Println("Usage: go run url-collection.go <domain> <program>")
        fmt.Println("Programs: Bugcrowd, HackerOne, Intigriti, Synack, YesWeHack")
        os.Exit(1)
    }

    domain := os.Args[1]
    program := os.Args[2]
    outputBaseDir := "/home/brainspiller/Documents/hunt"
    toolDir := "/home/brainspiller/go/bin"
    discordWebhookURL := "https://discord.com/api/webhooks/1260547967639879711/7g2e_dKU6tgsDAU7UL0pp0Z2-1Afjpbn6T4r939ox0mLJd6XBNR2c4s7Y8-fDFrFHDey"

    if !validateProgram(program) {
        fmt.Println("Invalid program. Choose from: Bugcrowd, HackerOne, Intigriti, Synack, YesWeHack")
        os.Exit(1)
    }

    userAgent := getUserAgent(program)
    outputDir := filepath.Join(outputBaseDir, program, domain)

    fmt.Printf("Creating output directory: %s\n", outputDir)
    if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
        log.Fatalf("Failed to create output directory: %v", err)
    }

    fmt.Printf("Changing to output directory: %s\n", outputDir)
    if err := os.Chdir(outputDir); err != nil {
        log.Fatalf("Failed to change directory: %v", err)
    }

    allSubsFile := filepath.Join(outputDir, "AllSubs.txt")
    if _, err := os.Stat(allSubsFile); os.IsNotExist(err) {
        log.Fatalf("AllSubs.txt does not exist in the specified directory")
    }

    waybackurlsCmd := exec.Command("sh", "-c", fmt.Sprintf("cat %s | %s | tee urls.txt", allSubsFile, filepath.Join(toolDir, "waybackurls")))
    executeCommand(waybackurlsCmd)
    fmt.Println("\033[32mWaybackurls completed\033[0m")

    fmt.Println("Extracting parameters from URLs...")
    grepParamCmd := exec.Command("sh", "-c", "grep '=' urls.txt | tee param.txt")
    executeCommand(grepParamCmd)
    fmt.Println("\033[32mParameter extraction completed\033[0m")

    fmt.Println("Extracting JavaScript files from URLs...")
    grepJsCmd := exec.Command("sh", "-c", "grep -iE '.js' urls.txt | grep -ivE '.json' | sort -u | tee js.txt")
    executeCommand(grepJsCmd)
    fmt.Println("\033[32mJavaScript file extraction completed\033[0m")

    fmt.Printf("User-Agent: %s\n", userAgent)

    fmt.Println("\033[31mURL collection completed\033[0m")

    message := fmt.Sprintf("Bug bounty - **URL Collection Service** has completed for **%s** on **%s**. Check your **param.txt** and **js.txt** to see extracted parameters and JavaScript files.", domain, program)
    if err := sendDiscordNotification(discordWebhookURL, message); err != nil {
        log.Fatalf("Failed to send Discord notification: %v", err)
    }
}
