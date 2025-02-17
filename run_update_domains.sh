#!/bin/bash

# Configuration
CONFIG_FILE="/home/brainspiller/Documents/hunt/domains_sub.txt"
LOG_FILE="/home/brainspiller/Documents/hunt/logs/update_domains.log"
TOOLS_DIR="/home/brainspiller/Documents/hunt/UpdateDomains"
GO_SCRIPT="$TOOLS_DIR/update_domains"

# Function to run the update_domains Go script
run_update_domains() {
    echo "Running update_domains script..." | tee -a "$LOG_FILE"

    # Execute the Go script
    sudo "$GO_SCRIPT" >> "$LOG_FILE" 2>&1

    # Check if the script ran successfully
    if [ $? -eq 0 ]; then
        echo "update_domains completed successfully." | tee -a "$LOG_FILE"
    else
        echo "update_domains failed to run." | tee -a "$LOG_FILE"
        exit 1
    fi
}

# Function to sort the domains_sub.txt file alphabetically
sort_domains() {
    echo "Sorting $CONFIG_FILE in alphabetical order..." | tee -a "$LOG_FILE"
    
    # Sort the file in place
    sort "$CONFIG_FILE" -o "$CONFIG_FILE"
    
    if [ $? -eq 0 ]; then
        echo "Sorting completed successfully." | tee -a "$LOG_FILE"
    else
        echo "Failed to sort $CONFIG_FILE." | tee -a "$LOG_FILE"
        exit 1
    fi
}

# Run the update_domains script to refresh the domain list
run_update_domains

# Sort the domains_sub.txt file after pulling the new data
sort_domains

# Confirm completion
echo "Script completed and log written to $LOG_FILE."
