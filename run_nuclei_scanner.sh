#!/bin/bash

# Configuration
CONFIG_FILE="/home/brainspiller/Documents/hunt/domains.txt"
LOG_FILE="/home/brainspiller/Documents/hunt/logs/nuclei_scanner.log"
TOOLS_DIR="/home/brainspiller/Documents/hunt/NucleiVulnerabilityScanning"
NUCLEI_SCRIPT="$TOOLS_DIR/nuclei-scanner"

# Check if Nuclei script exists
if [[ ! -f "$NUCLEI_SCRIPT" ]]; then
    echo "Nuclei script $NUCLEI_SCRIPT not found!" | tee -a $LOG_FILE
    exit 1
fi

# Check if the configuration file exists
if [[ ! -f "$CONFIG_FILE" ]]; then
    echo "Configuration file $CONFIG_FILE not found!" | tee -a $LOG_FILE
    exit 1
fi

# Main Script
while IFS= read -r line; do
    domain=$(echo $line | awk '{print $1}')
    program=$(echo $line | awk '{print $2}')

    # Ensure domain and program are not empty
    if [[ -z "$domain" || -z "$program" ]]; then
        echo "Error: Domain or Program is empty, skipping..." | tee -a $LOG_FILE
        continue
    fi

    echo "Running Nuclei scan for $domain under $program..." | tee -a $LOG_FILE

    # Run the Nuclei scanner script
    "$NUCLEI_SCRIPT" "$domain" "$program" >> $LOG_FILE 2>&1
    if [ $? -eq 0 ]; then
        echo "Finished Nuclei scan for $domain under $program." | tee -a $LOG_FILE
    else
        echo "Error occurred during Nuclei scan for $domain under $program." | tee -a $LOG_FILE
    fi

done < "$CONFIG_FILE"
