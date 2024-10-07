package main

import (
    "fmt"
    "log"
    "os"
    "io"
    "bytes"
    "os/exec"
    "net/http"
    "path/filepath"
    "strings"
    "syscall"
    "time"
    "encoding/json"
    Utils "port-scanning-service/Utils"
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
    delay := 300 * time.Second
    for {
        if process, err := os.FindProcess(pid); err == nil {
            if err := process.Signal(syscall.Signal(0)); err == nil {
                fmt.Println("Port scanning in progress, please wait...")
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
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("unexpected response from Discord: %s", body)
    }

    return nil
}

func main() {
     // Load environment variables from the .discordhooks.env file
     Utils.LoadEnv("/home/brainspiller/Documents/hunt/.discordhooks.env")

     // Get the Discord webhook URL from the environment
     discordWebhookURL := Utils.GetPortScanningWebhook()
 
     remindToStartDocker()
 
     if len(os.Args) != 3 {
         fmt.Println("Usage: go run port-scanning-service.go <domain> <program>")
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
 
     outputDir := filepath.Join(outputBaseDir, program, domain)
 
     fmt.Printf("Creating output directory: %s\n", outputDir)
     if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
         log.Fatalf("Failed to create output directory: %v", err)
     }
 
     fmt.Printf("Changing to output directory: %s\n", outputDir)
     if err := os.Chdir(outputDir); err != nil {
         log.Fatalf("Failed to change directory: %v", err)
     }
 
     // Naabu scan
     naabuOutputFile := fmt.Sprintf("%s.naabu.txt", domain)
     naabuCmd := exec.Command(filepath.Join(toolDir, "naabu"), "-host", domain)
 
     go showProgress(naabuCmd.Process.Pid)
 
     naabuOutput, err := executeCommandWithOutput(naabuCmd)
     if err != nil {
         log.Fatalf("Naabu scan failed: %v", err)
     }
     if err := os.WriteFile(naabuOutputFile, []byte(naabuOutput), 0644); err != nil {
         log.Fatalf("Failed to write to %s: %v", naabuOutputFile, err)
     }
     fmt.Println("\033[32mNaabu scan completed and results saved in", naabuOutputFile, "\033[0m")
 
     targetsFile := filepath.Join(outputDir, "targets.txt")
     if err := os.WriteFile(targetsFile, []byte(domain), 0644); err != nil {
         log.Fatalf("Failed to write to targets.txt: %v", err)
     }
 
     // Nmap scans - declare the file names here
     nmapScriptBannerFile := "nmapscriptbanner.txt"
     nmapCustomScriptAndVersion1_1000File := "nmapcustomscriptandversion1-1000.txt"
     nmapCustomScriptAndVersion1000_5000File := "nmapcustomscriptandversion1000-5000.txt"
 
     nmapCmd1 := exec.Command("nmap", "-sV", "--script=banner", "-iL", targetsFile, "-oN", nmapScriptBannerFile)
     executeCommand(nmapCmd1)
     fmt.Println("\033[31mNmap script banner scan completed and results saved in", nmapScriptBannerFile, "\033[0m")
 
     nmapCmd2 := exec.Command("nmap", "-sCV", "-iL", targetsFile, "-p", "1-1000", "-Pn", "-oN", nmapCustomScriptAndVersion1_1000File)
     executeCommand(nmapCmd2)
     fmt.Println("\033[31mNmap custom script and version scan (ports 1-1000) completed and results saved in", nmapCustomScriptAndVersion1_1000File, "\033[0m")
 
     nmapCmd3 := exec.Command("nmap", "-sCV", "-iL", targetsFile, "-p", "1000-5000", "-Pn", "-oN", nmapCustomScriptAndVersion1000_5000File)
     executeCommand(nmapCmd3)
     fmt.Println("\033[31mNmap custom script and version scan (ports 1000-5000) completed and results saved in", nmapCustomScriptAndVersion1000_5000File, "\033[0m")
 
     fmt.Println("\033[31mPort scanning completed\033[0m")

    message := fmt.Sprintf("Bug bounty - **Port-Scanning-Service** has completed for **%s** on **%s**. Check your scan results in %s.**naabu.txt**, **nmapscriptbanner.txt, nmapcustomscriptandversion1-1000.txt, and nmapcustomscriptandversion1000-5000.txt**", domain, program, domain)
    if err := sendDiscordNotification(discordWebhookURL, message); err != nil {
        log.Fatalf("Failed to send Discord notification: %v", err)
    }
}
