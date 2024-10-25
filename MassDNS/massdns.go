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
	"regexp"
	"strings"
	"github.com/joho/godotenv"
)

// Vulnerable services for subdomain takeovers and their associated fingerprints
var vulnerableServices = map[string]string{
	"elasticbeanstalk.com": "NXDOMAIN",
	"s3.amazonaws.com":     "The specified bucket does not exist",
	"agilecrm.com":         "Sorry, this page is no longer available.",
	"airee.ru":             "Ошибка 402. Сервис Айри.рф не оплачен",
	"animaapp.io":          "The page you were looking for does not exist.",
	"bitbucket.io":         "Repository not found",
	"canny.io":             "Company Not Found There is no such company. Did you enter the right URL?",
	"cargo.site":           "404 Not Found",
	"digitalocean.com":     "Domain uses DO name servers with no records in DO.",
	"trydiscourse.com":     "NXDOMAIN",
	"furyns.com":           "404: This page could not be found.",
	"getresponse.com":      "With GetResponse Landing Pages, lead generation has never been easier",
	"ghost.io":             "Site unavailable\\.|Failed to resolve DNS path for this host",
	"github.io":            "There isn't a GitHub Pages site here.",
	"hatenablog.com":       "404 Blog is not found",
	"helpjuice.com":        "We could not find what you're looking for.",
	"helpscoutdocs.com":    "No settings were found for this company:",
	"helprace.com":         "HTTP_STATUS=301",
	"heroku.com":           "No such app",
	"youtrack.cloud":       "is not a registered InCloud YouTrack",
	"launchrock.com":       "HTTP_STATUS=500",
	"ngrok.io":             "Tunnel .*\\.ngrok.io not found",
	"pantheon.io":          "404 error unknown site!",
	"pingdom.com":          "Sorry, couldn't find the status page",
	"readme.io":            "The creators of this project are still working on making everything perfect!",
	"readthedocs.org":      "The link you have followed or the URL that you entered does not exist.",
	"short.io":             "Link does not exist",
	"52.16.160.97":         "This job board website is either expired or its domain name is invalid.",
	"strikinglydns.com":    "PAGE NOT FOUND.",
	"na-west1.surge.sh":    "project not found",
	"surveysparrow.com":    "Account not found.",
	"read.uberflip.com":    "The URL you've accessed does not provide a hub.",
	"uptimerobot.com":      "page not found",
	"wordpress.com":        "Do you want to register .*.wordpress.com?",
	"worksites.net":        "Hello! Sorry, but the website you’re looking for doesn’t exist.",

	// AWS-specific patterns for DNS subdomain takeovers
	`s3-website\.\w+-\w+-\d+\.amazonaws\.com`:  "The specified bucket does not exist",          // S3 static website
	`s3\.\w+-\w+-\d+\.amazonaws\.com`:          "The specified bucket does not exist",          // Regular S3 bucket
	`cloudfront\.net`:                          "403 ERROR The request could not be satisfied", // CloudFront distributions
	`elb\.amazonaws\.com`:                      "403 ERROR",                                    // Elastic Load Balancer
	`execute-api\.\w+-\w+-\d+\.amazonaws\.com`: "403 Forbidden",                                // API Gateway
	`lambda-url\.\w+-\w+-\d+\.amazonaws\.com`:  "403 Forbidden",                                // Lambda URL patterns

	// Added Azure Services (All check for NXDOMAIN)
	"cloudapp.net":            "NXDOMAIN",
	"cloudapp.azure.com":      "NXDOMAIN",
	"azurewebsites.net":       "NXDOMAIN",
	"blob.core.windows.net":   "NXDOMAIN",
	"azure-api.net":           "NXDOMAIN",
	"azurehdinsight.net":      "NXDOMAIN",
	"azureedge.net":           "NXDOMAIN",
	"azurecontainer.io":       "NXDOMAIN",
	"database.windows.net":    "NXDOMAIN",
	"azuredatalakestore.net":  "NXDOMAIN",
	"search.windows.net":      "NXDOMAIN",
	"azurecr.io":              "NXDOMAIN",
	"redis.cache.windows.net": "NXDOMAIN",
	"servicebus.windows.net":  "NXDOMAIN",
	"visualstudio.com":        "NXDOMAIN",
}

// List of vulnerable nameservers (with wildcards)
var vulnerableNameservers = []string{
	`ns1\w*\.name\.com`, // Matches ns1 followed by any characters ending with .name.com
	`ns2\w*\.name\.com`,
	`ns3\w*\.name\.com`,
	`ns4\w*\.name\.com`,
	`ns-cloud-\w*\.googledomains\.com`, // Matches ns-cloud- followed by any characters ending with .googledomains.com
	`yns1\.yahoo\.com`,
	`yns2\.yahoo\.com`,
	`ns1\.reg\.ru`,
	`ns2\.reg\.ru`,
	`ns1\.mydomain\.com`,
	`ns2\.mydomain\.com`,
	`ns1\.linode\.com`,
	`ns2\.linode\.com`,
	`ns5\.he\.net`,
	`ns4\.he\.net`,
	`ns3\.he\.net`,
	`ns2\.he\.net`,
	`ns1\.he\.net`,
	`ns1\.dnsimple\.com`,
	`ns2\.dnsimple\.com`,
	`ns3\.dnsimple\.com`,
	`ns4\.dnsimple\.com`,
	`ns1\.digitalocean\.com`,
	`ns2\.digitalocean\.com`,
	`ns3\.digitalocean\.com`,
	`ns1\.000domains\.com`,
	`ns2\.000domains\.com`,
	`fwns1\.000domains\.com`,
	`fwns2\.000domains\.com`,

	// AWS Route 53 nameserver patterns
	`ns-\d+\.awsdns-\d+\.co\.uk`, // Matches ns-<digits>.awsdns-<digits>.co.uk
	`ns-\d+\.awsdns-\d+\.org`,    // Matches ns-<digits>.awsdns-<digits>.org
	`ns-\d+\.awsdns-\d+\.net`,    // Matches ns-<digits>.awsdns-<digits>.net
	`ns-\d+\.awsdns-\d+\.com`,    // Matches ns-<digits>.awsdns-<digits>.com
}

// Load Discord webhook URLs from the .env file
func loadDiscordHooks() (string, error) {
	err := godotenv.Load("/home/brainspiller/Documents/hunt/.discordhooks.env")
	if err != nil {
		return "", fmt.Errorf("error loading .env file: %v", err)
	}
	//scanCompletionWebhook := os.Getenv("DISCORD_SCAN_COMPLETION_WEBHOOK")
	potentialTakeoverWebhook := os.Getenv("DISCORD_POTENTIAL_TAKEOVER_WEBHOOK")
	return potentialTakeoverWebhook, nil
}

// Send a notification to Discord
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected response from Discord: %s", body)
	}

	return nil
}

// Run MassDNS on a given domain
func runMassDNS(inputFile, outputFile string) error {
	cmd := exec.Command("/home/brainspiller/go/bin/massdns", "-r", "/opt/massdns/lists/resolvers.txt", "-o", "J", "-w", outputFile, inputFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Filter results for NXDOMAIN and SERVFAIL
func filterResults(outputFile string) ([]map[string]interface{}, error) {
    data, err := os.ReadFile(outputFile)
    if err != nil {
        return nil, fmt.Errorf("failed to read MassDNS output: %v", err)
    }

    var results []map[string]interface{}
    lines := strings.Split(string(data), "\n")
    for _, line := range lines {
        if len(line) == 0 {
            continue // Skip empty lines
        }

        var result map[string]interface{}
        if err := json.Unmarshal([]byte(line), &result); err != nil {
            log.Printf("Error parsing JSON: %v", err)
            log.Printf("Raw data: %s", line)
            continue // Skip if parsing fails
        }

        // Check for the status
        if status, ok := result["status"].(string); ok {
            // Only add results with NXDOMAIN or SERVFAIL statuses
            if status == "NXDOMAIN" || status == "SERVFAIL" {
                results = append(results, result) // Add the result to the list if it matches
            }
        }
    }
    return results, nil
}


func main() {
	// Load Discord webhook URLs
	potentialTakeoverWebhook, err := loadDiscordHooks()
	if err != nil {
		log.Fatalf("Failed to load Discord webhooks: %v", err)
	}

	// Open domains_sub.txt
	file, err := os.Open("/home/brainspiller/Documents/hunt/domains_sub.txt")
	if err != nil {
		log.Fatalf("Failed to open domains_sub.txt: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)

		if len(parts) != 2 {
			log.Printf("Skipping invalid line: %s", line)
			continue
		}

		domain := parts[0]
		program := parts[1]

		fmt.Printf("Processing domain: %s for program: %s\n", domain, program)

		// Build path to Allsubs.txt
		allSubsPath := filepath.Join("/home/brainspiller/Documents/hunt", program, domain, "AllSubs.txt")
		massdnsOutputFile := filepath.Join("/home/brainspiller/Documents/hunt", program, domain, "massdns_output.json")

		// Check if Allsubs.txt exists
		if _, err := os.Stat(allSubsPath); os.IsNotExist(err) {
			log.Printf("AllSubs.txt does not exist for domain: %s in program: %s. Skipping...", domain, program)
			continue // Skip to the next entry if Allsubs.txt does not exist
		}

		// Run MassDNS
		if err := runMassDNS(allSubsPath, massdnsOutputFile); err != nil {
			log.Fatalf("MassDNS execution failed: %v", err)
		}

		// Filter results
		results, err := filterResults(massdnsOutputFile)
		if err != nil {
			log.Fatalf("Filtering results failed: %v", err)
		}

		// Notify for each NXDOMAIN or SERVFAIL result
for _, result := range results {
    // Safely get the status
    status, statusOk := result["status"].(string)
    subdomain, subdomainOk := result["name"].(string)

    if !statusOk || !subdomainOk {
        log.Printf("Invalid result structure: %v", result)
        continue
    }

    if status == "NXDOMAIN" {
        // Extracting the data field safely
        dataField, dataOk := result["data"].(map[string]interface{})
        if !dataOk {
            log.Printf("Missing data field in result: %v", result)
            continue
        }

        answers, answersOk := dataField["answers"].([]interface{})
        var cname string
        if answersOk && len(answers) > 0 {
            if firstAnswer, ok := answers[0].(map[string]interface{}); ok {
                cname = firstAnswer["data"].(string) // Adjusted for correct structure
            } else {
                log.Printf("Expected map structure for answer: %v", answers[0])
                continue
            }
        } else {
            log.Printf("No answers found for NXDOMAIN: %s", subdomain)
            continue
        }

        // Check if CNAME is in the vulnerable services
        vulnerable := false
        for service := range vulnerableServices {
            if strings.Contains(cname, service) {
                vulnerable = true
                break
            }
        }
        if vulnerable {
            message := fmt.Sprintf("**New Dangling Record**: %s with code: **%s** and records:**[%s]**", subdomain, status, cname)
            if err := sendDiscordNotification(potentialTakeoverWebhook, message); err != nil {
                log.Printf("Failed to send Discord notification: %v", err)
            }
        }
    } else if status == "SERVFAIL" {
        // Get nameservers safely
        nameservers, nsOk := result["ns"].([]interface{})
        if !nsOk {
            log.Printf("No nameservers found for SERVFAIL: %s", subdomain)
            continue
        }

        // Check if any nameservers are in the vulnerable nameservers
        for _, ns := range nameservers {
            for _, vulnerableNS := range vulnerableNameservers {
                match, _ := regexp.MatchString(vulnerableNS, ns.(string))
                if match {
                    message := fmt.Sprintf("**New NS Server match**: %s with code: **%s** and nameservers:\n%v", subdomain, status, nameservers)
                    if err := sendDiscordNotification(potentialTakeoverWebhook, message); err != nil {
                        log.Printf("Failed to send Discord notification: %v", err)
                    }
                    break
                }
            }
        }
    }
}

// Handle any scanner errors
	if err := scanner.Err(); err != nil {
		log.Fatalf("Failed to read domains_sub.txt: %v", err)

	}

	// Send a completion message to Discord
	//completionMessage := "MassDNS Processing completed."
	//if err := sendDiscordNotification(scanCompletionWebhook, completionMessage); err != nil {
	//	log.Printf("Failed to send completion notification: %v", err)

	//}

	fmt.Println("MassDNS Processing completed.")
	}
}