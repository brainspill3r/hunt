package main

import (
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
    Utils "url-collection/Utils"
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

func main() {
    // Set up logging to both a log file and the console
	logFilePath := "/home/brainspiller/Documents/hunt/logs/url_collection.log"
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to create log file: %v", err)
	}
	defer logFile.Close()

	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

    // Load environment variables from the .discordhooks.env file
    Utils.LoadEnv("/home/brainspiller/Documents/hunt/.discordhooks.env")

    // Get the Discord webhook URL for URL collection from the environment
    discordWebhookURL := Utils.GetURLCollectionWebhook()

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
