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
	Utils "subdomain-enumeration/Utils"
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

// readAndCombineFiles reads and combines unique lines from multiple files
//func readAndCombineFiles(files []string, outputFile string) {
//	contentMap := make(map[string]struct{})
//	for _, file := range files {
//		data, err := os.ReadFile(file)
//		if err != nil {
//			log.Fatalf("Failed to read file %s: %v", file, err)
//		}
//		lines := strings.Split(string(data), "\n")
//		for _, line := range lines {
//			if line != "" {
//				contentMap[line] = struct{}{}
//			}
//		}
//	}
//
//	outputData := ""
//	for line := range contentMap {
//		outputData += line + "\n"
//	}
//
//	if err := os.WriteFile(outputFile, []byte(outputData), 0644); err != nil {
//		log.Fatalf("Failed to write to %s: %v", outputFile, err)
//	}
//}

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
	// Load environment variables from the .discordhooks.env file
	Utils.LoadEnv("/home/brainspiller/Documents/hunt/.discordhooks.env")

	// Get the Discord webhook URL from environment variables
	discordWebhookURL := Utils.GetScanCompletionWebhook()

	remindToStartDocker()

	if len(os.Args) != 3 {
		fmt.Println("Usage: go run subdomain-enumeration.go <domain> <program>")
		fmt.Println("Programs: Bugcrowd, HackerOne, Intigriti, Synack, YesWeHack")
		os.Exit(1)
	}

	domain := os.Args[1]
	program := os.Args[2]
	outputBaseDir := "/home/brainspiller/Documents/hunt"
	toolDir := "/home/brainspiller/go/bin"
	crtShScript := "/opt/crt.sh/crt.sh"
	cnameCheckScript := "/opt/cname-check/cname.sh"
	//discordWebhookURL := Utils.GetScanCompletionWebhook()
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

	targetsFile := filepath.Join(outputDir, "targets.txt")
	fmt.Printf("Writing targets to file: %s\n", targetsFile)
	if err := ioutil.WriteFile(targetsFile, []byte(domain), 0644); err != nil {
		log.Fatalf("Failed to write to targets.txt: %v", err)
	}

	subfinderCmd := exec.Command(filepath.Join(toolDir, "subfinder"), "-dL", targetsFile, "-all", "--recursive", "-o", "Subs.txt")
	executeCommand(subfinderCmd)
	fmt.Println("\033[32mSubfinder completed\033[0m")

	crtshCmd := exec.Command("sh", "-c", fmt.Sprintf("%s -d %s -o Subs02.txt", crtShScript, domain))
	executeCommand(crtshCmd)
	fmt.Println("\033[32mcrt.sh completed\033[0m")

	assetfinderCmd := exec.Command(filepath.Join(toolDir, "assetfinder"), "--subs-only", domain)
	assetfinderOutput, err := assetfinderCmd.Output()
	if err != nil {
		log.Fatalf("Command failed: %v", err)
	}
	if err := ioutil.WriteFile("Subs03.txt", assetfinderOutput, 0644); err != nil {
		log.Fatalf("Failed to write to Subs03.txt: %v", err)
	}
	fmt.Println("\033[32mAssetfinder completed\033[0m")

	findomainCmd := exec.Command(filepath.Join(toolDir, "findomain"), "-t", domain, "-u", "Subs04.txt")
	executeCommand(findomainCmd)
	fmt.Println("\033[32mFindomain completed\033[0m")

	// Ensure crt.sh output is included
	crtshOutputFile := fmt.Sprintf("output/domain.%s.txt", domain)
	if _, err := os.Stat(crtshOutputFile); os.IsNotExist(err) {
		log.Fatalf("crt.sh did not create %s", crtshOutputFile)
	}

	fmt.Println("Combining subdomain results...")
	anewCmd := exec.Command("sh", "-c", fmt.Sprintf("cat Subs*.txt %s | anew AllSubs.txt", crtshOutputFile))
	executeCommand(anewCmd)
	fmt.Println("\033[32mAnew completed\033[0m")

	// Check if AllSubs.txt is created
	if _, err := os.Stat("AllSubs.txt"); os.IsNotExist(err) {
		log.Fatalf("Anew did not create AllSubs.txt")
	}

	fmt.Println("Checking which subdomains are alive...")
	aliveSubsFile := filepath.Join(outputDir, "AliveSubs.txt")
	httpxCmd := exec.Command("sh", "-c", fmt.Sprintf("cat %s | %s -H %q | tee %s", "AllSubs.txt", filepath.Join(toolDir, "httpx"), userAgent, aliveSubsFile))
	executeCommand(httpxCmd)
	fmt.Println("\033[32mHttpx completed\033[0m")

	fmt.Println("Filtering in-scope subdomains...")
	finalFile := filepath.Join(outputDir, "Final.txt")
	grepCmd := exec.Command("sh", "-c", fmt.Sprintf("cat %s | grep %s | tee %s", aliveSubsFile, domain, finalFile))
	executeCommand(grepCmd)
	fmt.Println("\033[31mFinal in-scope subdomains saved to Final.txt\033[0m")

	fmt.Println("Running cname-check...")
	cnameCheckCmd := exec.Command("sh", "-c", fmt.Sprintf("%s %s", cnameCheckScript, aliveSubsFile))
	executeCommand(cnameCheckCmd)
	fmt.Println("\033[32mcname-check completed\033[0m")

	fmt.Printf("User-Agent: %s\n", userAgent)

	fmt.Println("\033[31mSubdomain enumeration completed\033[0m")

	message := fmt.Sprintf("Bug bounty - **Subdomain-Enumeration** has completed for **%s** on **%s**. Check your **Final.txt** to see Alive in-scope domains.", domain, program)
	if err := sendDiscordNotification(discordWebhookURL, message); err != nil {
		log.Fatalf("Failed to send Discord notification: %v", err)
	}
}
