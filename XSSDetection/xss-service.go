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
    "syscall"
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

// showProgress shows progress while a long-running command is executed
func showProgress(pid int) {
    delay := 10 * time.Second
    for {
        if process, err := os.FindProcess(pid); err == nil {
            if err := process.Signal(syscall.Signal(0)); err == nil {
                fmt.Println("XSS detection in progress, please wait...")
                time.Sleep(delay)
            } else {
                break
            }
        } else {
            break
        }
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
        body, _ := ioutil.ReadAll(resp.Body)
        return fmt.Errorf("unexpected response from Discord: %s", body)
    }

    return nil
}

func main() {
    remindToStartDocker()

    if len(os.Args) != 3 {
        fmt.Println("Usage: go run xss-service.go <domain> <program>")
        fmt.Println("Programs: Bugcrowd, HackerOne, Intigriti, Synack, YesWeHack")
        os.Exit(1)
    }

    domain := os.Args[1]
    program := os.Args[2]
    outputBaseDir := "/home/brainspiller/Documents/hunt"
    toolDir := "/home/brainspiller/go/bin"
    uroPath := "/usr/local/bin/uro"
    discordWebhookURL := "https://discord.com/api/webhooks/1260997894035734548/gy5BbNRUfbxRshN5xSAYNk3YJGEwiZnVBRps596flOGgFBQlh2n8YTp1_q9IZmnTH-9y"

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

    fmt.Println("\033[32mProcessing URLs for potential XSS...\033[0m")

    xssCmd := exec.Command("sh", "-c", fmt.Sprintf("cat %s | %s | %s xss | tee %s", filepath.Join(outputDir, "urls.txt"), uroPath, filepath.Join(toolDir, "gf"), filepath.Join(outputDir, "xss.txt")))
    executeCommand(xssCmd)

    xssFile := filepath.Join(outputDir, "xss.txt")

    if _, err := os.Stat(xssFile); err != nil {
        log.Fatalf("Failed to find xss.txt: %v", err)
    }

    if fileInfo, err := os.Stat(xssFile); err == nil && fileInfo.Size() == 0 {
        fmt.Println("\033[31mNothing found for XSS! Better luck next time.\033[0m")
    } else {
        fmt.Println("\033[32mPotential XSS URLs have been saved to xss.txt.\033[0m")

        dalfoxCmd := exec.Command(filepath.Join(toolDir, "dalfox"), "file", xssFile)
        fmt.Printf("\033[33mExecuting command: %s\033[0m\n", strings.Join(dalfoxCmd.Args, " "))
        executeCommand(dalfoxCmd)

        fmt.Println("\033[32mDalfox XSS scan completed. Results saved to XSSVulnerablepayloads.txt.\033[0m")
    }

    fmt.Printf("User-Agent: %s\n", userAgent)

    fmt.Println("\033[31mXSS detection completed\033[0m")

    message := fmt.Sprintf("Bug bounty - **XSS Detection Service** has completed for **%s** on **%s**. Check your **xss.txt** and **XSSVulnerablepayloads.txt** to see the results.", domain, program)
    if err := sendDiscordNotification(discordWebhookURL, message); err != nil {
        log.Fatalf("Failed to send Discord notification: %v", err)
    }
}
