#!/bin/bash

# WEKA Maintenance Event Monitor for Azure
# This script monitors Azure maintenance events and reports them to the WEKA cluster

set -e

# Configuration
IMDS_ENDPOINT="http://169.254.169.254/metadata/scheduledevents?api-version=2020-07-01"
CHECK_INTERVAL=${CHECK_INTERVAL:-30}  # Check every 30 seconds by default
CLUSTER_NAME=${CLUSTER_NAME:-""}
LOG_FILE="/var/log/weka-maintenance-monitor.log"

# Logging function
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "$LOG_FILE"
}

# Fetch function definition will be injected here
# FETCH_FUNCTION_PLACEHOLDER

# Send custom event to WEKA cluster
send_weka_event() {
    local event_message=$1
    local hostname=$(hostname)

    log "Sending WEKA event: $event_message"

    # Create a custom event using WEKA events
    if weka events trigger-event "$event_message" 2>&1 | tee -a "$LOG_FILE"; then
        log "Successfully sent WEKA custom event"
        return 0
    else
        log "ERROR: Failed to send WEKA custom event"
        return 1
    fi
}

# Check for maintenance events
check_maintenance_events() {
    local response=$(curl -s -H "Metadata: true" "$IMDS_ENDPOINT" 2>&1)

    if [ $? -ne 0 ]; then
        log "ERROR: Failed to query IMDS endpoint"
        return 1
    fi

    # Check if there are any scheduled events using jq
    local events_count=$(echo "$response" | jq -r '.Events | length' 2>/dev/null || echo "0")

    if [ "$events_count" -gt 0 ]; then
        log "Detected $events_count scheduled maintenance event(s)"

        # Extract event details using jq
        local event_type=$(echo "$response" | jq -r '.Events[0].EventType' 2>/dev/null || echo "")
        local event_status=$(echo "$response" | jq -r '.Events[0].EventStatus' 2>/dev/null || echo "")
        local not_before=$(echo "$response" | jq -r '.Events[0].NotBefore' 2>/dev/null || echo "")

        log "Event Type: $event_type, Status: $event_status, NotBefore: $not_before"

        # Format event message with 128 character limit
        local hostname=$(hostname)
        local base_message="Azure maintenance event on $hostname"
        local event_message="$base_message"
        local max_length=128

        # Add fields in priority order: NotBefore, Type, Status
        # Only add if not empty and if it fits within the limit
        if [ -n "$not_before" ] && [ ${#event_message} -lt $max_length ]; then
            local temp_message="$event_message, NotBefore=$not_before"
            if [ ${#temp_message} -le $max_length ]; then
                event_message="$temp_message"
            fi
        fi

        if [ -n "$event_type" ] && [ ${#event_message} -lt $max_length ]; then
            local temp_message="$event_message, Type=$event_type"
            if [ ${#temp_message} -le $max_length ]; then
                event_message="$temp_message"
            fi
        fi

        if [ -n "$event_status" ] && [ ${#event_message} -lt $max_length ]; then
            local temp_message="$event_message, Status=$event_status"
            if [ ${#temp_message} -le $max_length ]; then
                event_message="$temp_message"
            fi
        fi

        # Truncate if still over limit (shouldn't happen, but safety check)
        if [ ${#event_message} -gt $max_length ]; then
            event_message="${event_message:0:$max_length}"
        fi

        # Get WEKA credentials and login
        fetch_result=$(fetch "{\"fetch_weka_credentials\": true}")
        weka_username="$(echo $fetch_result | jq -r .username)"
        weka_password="$(echo $fetch_result | jq -r .password)"

        if weka user login "$weka_username" "$weka_password"; then
            send_weka_event "$event_message"
        else
            log "ERROR: Could not login to WEKA cluster"
            return 1
        fi

        return 0
    fi

    return 0
}

# Main monitoring loop
main() {
    log "Starting WEKA maintenance event monitor"
    log "Cluster: $CLUSTER_NAME, Check Interval: ${CHECK_INTERVAL}s"

    # Track if we've already reported a maintenance event to avoid duplicate reports
    local last_event_hash=""

    while true; do
        # Get current events
        local current_events=$(curl -s -H "Metadata: true" "$IMDS_ENDPOINT" 2>&1 || echo "")
        local current_hash=$(echo "$current_events" | md5sum | cut -d' ' -f1)

        # Only process if events have changed
        if [ "$current_hash" != "$last_event_hash" ]; then
            check_maintenance_events
            last_event_hash="$current_hash"
        fi

        sleep "$CHECK_INTERVAL"
    done
}

# Run main function
main
