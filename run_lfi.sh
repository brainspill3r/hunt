#!/bin/bash

# Configuration
CONFIG_FILE="/home/brainspiller/Documents/hunt/domains.txt"
LOG_FILE="/home/brainspiller/Documents/hunt/logs/lfi.log"
TOOLS_DIR="/home/brainspiller/Documents/hunt"

# Function to run a tool
run_tool() {
    local domain=$1
    local program=$2
    local tool_dir=$3
    local tool_name=$4
    local tool_path="$TOOLS_DIR/$tool_dir/$tool_name"

    if [[ -z "$domain" || -z "$program" ]]; then
        echo "Error: Domain or Program is empty, skipping..." | tee -a $LOG_FILE
        return
    fi

    echo "Running $tool_dir/$tool_name for $domain under $program..." | tee -a $LOG_FILE
    sudo "$tool_path" "$domain" "$program" >> $LOG_FILE 2>&1
    if [ $? -eq 0 ]; then
        echo "Successfully finished $tool_dir/$tool_name for $domain under $program." | tee -a $LOG_FILE
    else
        echo "Failed to run $tool_dir/$tool_name for $domain under $program." | tee -a $LOG_FILE
    fi
}

# Main Script
while IFS= read -r line; do
    domain=$(echo $line | awk '{print $1}')
    program=$(echo $line | awk '{print $2}')

    run_tool "$domain" "$program" "LFIDetection" "lfi-detection"
    run_tool "$domain" "$program" "XSSDetection" "xss-service"
done < "$CONFIG_FILE"
