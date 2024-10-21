#!/bin/bash

CONFIG_FILE="/home/brainspiller/Documents/hunt/domains_sub.txt"
LOG_FILE="/home/brainspiller/Documents/hunt/logs/massdns.log"
TOOLS_DIR="/home/brainspiller/Documents/hunt/MassDNS"
GO_SCRIPT="$TOOLS_DIR/massdns.go"

# Check if Go script exists
if [[ ! -f "$GO_SCRIPT" ]]; then
    echo "Go script $GO_SCRIPT not found!" | tee -a $LOG_FILE
    exit 1
fi

# Check if the configuration file exists
if [[ ! -f "$CONFIG_FILE" ]]; then
    echo "Configuration file $CONFIG_FILE not found!" | tee -a $LOG_FILE
    exit 1
fi

# Main Script
while IFS= read -r line; do
    domain=$(echo "$line" | awk '{print $1}')
    program=$(echo "$line" | awk '{print $2}')

    # Ensure domain and program are not empty
    if [[ -z "$domain" || -z "$program" ]]; then
        echo "Error: Domain or Program is empty, skipping..." | tee -a $LOG_FILE
        continue
    fi

    # Build path to Allsubs.txt
    allSubsPath="/home/brainspiller/Documents/hunt/$program/$domain/AllSubs.txt"

    # Check if Allsubs.txt exists
    if [[ ! -f "$allSubsPath" ]]; then
        echo "AllSubs.txt does not exist for domain: $domain in program: $program. Skipping..." | tee -a $LOG_FILE
        continue
    fi

    echo "Running MassDNS for $domain under $program..." | tee -a $LOG_FILE

    # Log the command being run
    command="go run \"$GO_SCRIPT\" \"$allSubsPath\""
    echo "Executing command: $command" | tee -a $LOG_FILE

    # Run the Go script
    eval "$command" >> $LOG_FILE 2>&1
    if [ $? -eq 0 ]; then
        echo "Finished running MassDNS for $domain under $program." | tee -a $LOG_FILE
    else
        echo "Error occurred while running MassDNS for $domain under $program. Check the log for details." | tee -a $LOG_FILE
    fi

done < "$CONFIG_FILE"

# Send a completion message to Discord or log it
echo "MassDNS processing has been completed." | tee -a $LOG_FILE
