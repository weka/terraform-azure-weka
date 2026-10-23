#!/bin/bash

# WEKA Maintenance Event Monitor for Azure
# Polls the Azure Instance Metadata Service (IMDS) Scheduled Events endpoint and
# reports Azure scheduled events (reboot, redeploy, freeze, ...) to the WEKA cluster
# as custom events.
#
# - every event in the IMDS document is reported, not only the first one
# - an event is reported when it appears, when its EventStatus changes and when it
#   leaves the document (completed or cancelled); state is kept per EventId, so a
#   change to one event does not re-report the others
# - the message carries EventType, EventStatus, EventSource, the affected Resources,
#   NotBefore and the EventId; scheduled events are delivered to every VM in the
#   placement group, so Resources (not the reporting VM) names the affected VMs
# - every changed IMDS document is appended to a raw JSON lines log
#
# IMDS reference: https://learn.microsoft.com/azure/virtual-machines/linux/scheduled-events

# Configuration (overridable from /etc/default/weka-maintenance-monitor)
IMDS_ENDPOINT=${IMDS_ENDPOINT:-"http://169.254.169.254/metadata/scheduledevents?api-version=2020-07-01"}
IMDS_TIMEOUT=${IMDS_TIMEOUT:-10}
CHECK_INTERVAL=${CHECK_INTERVAL:-30}  # Check every 30 seconds by default
CLUSTER_NAME=${CLUSTER_NAME:-""}
LOG_FILE=${LOG_FILE:-"/var/log/weka-maintenance-monitor.log"}
RAW_LOG_FILE=${RAW_LOG_FILE:-"/var/log/weka-maintenance-monitor-raw.jsonl"}
STATE_FILE=${STATE_FILE:-"/var/lib/weka-maintenance-monitor/state.json"}
MAX_MESSAGE_LENGTH=${MAX_MESSAGE_LENGTH:-128}
MAX_RESOURCES_IN_MESSAGE=${MAX_RESOURCES_IN_MESSAGE:-3}

# Logging function
# Writes to the log file; also to stdout when running interactively. Under systemd
# stdout is appended to the same file by the unit, so write to the file only once.
log() {
    local line
    line="$(date '+%Y-%m-%d %H:%M:%S') - $1"
    echo "$line" >> "$LOG_FILE"
    if [ -t 1 ]; then
        echo "$line"
    fi
}

utc_now() {
    date -u '+%Y-%m-%dT%H:%M:%SZ'
}

# Fetch function definition will be injected here
# FETCH_FUNCTION_PLACEHOLDER

# Get WEKA credentials and login
weka_login() {
    local fetch_result weka_username weka_password

    fetch_result=$(fetch "{\"fetch_weka_credentials\": true}")
    weka_username="$(echo "$fetch_result" | jq -r .username 2>/dev/null)"
    weka_password="$(echo "$fetch_result" | jq -r .password 2>/dev/null)"

    if [ -z "$weka_username" ] || [ "$weka_username" = "null" ]; then
        log "ERROR: Could not fetch WEKA credentials"
        return 1
    fi

    if weka user login "$weka_username" "$weka_password" >> "$LOG_FILE" 2>&1; then
        return 0
    fi

    log "ERROR: Could not login to WEKA cluster"
    return 1
}

# Send custom event to WEKA cluster
send_weka_event() {
    local event_message=$1

    log "Sending WEKA event: $event_message"

    if weka events trigger-event "$event_message" >> "$LOG_FILE" 2>&1; then
        log "Successfully sent WEKA custom event"
        return 0
    fi

    log "ERROR: Failed to send WEKA custom event"
    return 1
}

# Append a field to the message only if the result stays within MAX_MESSAGE_LENGTH
append_if_fits() {
    local current=$1
    local addition=$2
    if [ $(( ${#current} + ${#addition} )) -le "$MAX_MESSAGE_LENGTH" ]; then
        echo "${current}${addition}"
    else
        echo "$current"
    fi
}

# Resources list for the message: up to MAX_RESOURCES_IN_MESSAGE names, then "+N more"
format_resources() {
    local resources_json=$1
    echo "$resources_json" | jq -r --argjson n "$MAX_RESOURCES_IN_MESSAGE" \
        'if length == 0 then "unknown"
         elif length > $n then (.[0:$n] | join(",")) + " +" + ((length - $n) | tostring) + " more"
         else join(",") end' 2>/dev/null || echo "unknown"
}

# NotBefore as compact UTC. IMDS returns RFC 1123 ("Mon, 19 Sep 2016 18:29:47 GMT");
# anything else is passed through unchanged.
format_not_before() {
    local not_before=$1 day month year time month_num
    [ -z "$not_before" ] && return 0
    if [[ "$not_before" =~ ^[A-Za-z]{3},\ ([0-9]{1,2})\ ([A-Za-z]{3})\ ([0-9]{4})\ ([0-9]{2}:[0-9]{2}:[0-9]{2})\ GMT$ ]]; then
        day=${BASH_REMATCH[1]}; month=${BASH_REMATCH[2]}; year=${BASH_REMATCH[3]}; time=${BASH_REMATCH[4]}
        case "$month" in
            Jan) month_num=01 ;; Feb) month_num=02 ;; Mar) month_num=03 ;; Apr) month_num=04 ;;
            May) month_num=05 ;; Jun) month_num=06 ;; Jul) month_num=07 ;; Aug) month_num=08 ;;
            Sep) month_num=09 ;; Oct) month_num=10 ;; Nov) month_num=11 ;; Dec) month_num=12 ;;
            *) echo "$not_before"; return 0 ;;
        esac
        printf '%s-%s-%02dT%sZ\n' "$year" "$month_num" "$((10#$day))" "$time"
        return 0
    fi
    echo "$not_before"
}

# Message for a new event or a status change
# Fields are added in priority order and dropped from the end when they do not fit.
compose_event_message() {
    local event_type=$1 event_status=$2 event_source=$3 resources_json=$4
    local not_before=$5 event_id=$6 duration=$7
    local message

    message="Azure ${event_type} ${event_status} (${event_source}) for $(format_resources "$resources_json")"
    if [ -n "$not_before" ]; then
        message=$(append_if_fits "$message" ", NotBefore=$(format_not_before "$not_before")")
    fi
    message=$(append_if_fits "$message" ", id=${event_id:0:8}")
    if [ -n "$duration" ] && [ "$duration" != "null" ] && [ "$duration" -ge 0 ] 2>/dev/null; then
        message=$(append_if_fits "$message" ", duration=${duration}s")
    fi
    message=$(append_if_fits "$message" ", via $(hostname)")

    echo "${message:0:$MAX_MESSAGE_LENGTH}"
}

# Message for an event that left the IMDS document
compose_removed_message() {
    local event_type=$1 resources_json=$2 event_id=$3
    local message

    message="Azure ${event_type} for $(format_resources "$resources_json") no longer scheduled (completed or cancelled)"
    message=$(append_if_fits "$message" ", id=${event_id:0:8}")
    message=$(append_if_fits "$message" ", via $(hostname)")

    echo "${message:0:$MAX_MESSAGE_LENGTH}"
}

# State file: {"<EventId>": {"status": ..., "type": ..., "resources": [...], "not_before": ..., "updated": ...}}
init_state_file() {
    mkdir -p "$(dirname "$STATE_FILE")"
    if [ ! -s "$STATE_FILE" ] || ! jq -e 'type == "object"' "$STATE_FILE" > /dev/null 2>&1; then
        echo '{}' > "$STATE_FILE"
        log "Initialized state file $STATE_FILE"
    fi
}

write_state_file() {
    local state=$1
    echo "$state" > "${STATE_FILE}.tmp" && mv -f "${STATE_FILE}.tmp" "$STATE_FILE"
}

append_raw_document() {
    local document=$1
    echo "$document" | jq -c --arg ts "$(utc_now)" --arg host "$(hostname)" \
        '{ts: $ts, host: $host, document: .}' >> "$RAW_LOG_FILE" 2>/dev/null \
        || log "WARN: Could not append the IMDS document to $RAW_LOG_FILE"
}

# Report new, changed and removed events. Returns 1 if any report failed (retried next poll).
process_document() {
    local document=$1
    local state new_state seen_ids=" " failures=0 logged_in=0
    local event event_id event_type event_status event_source not_before duration resources
    local known_status message

    state=$(cat "$STATE_FILE" 2>/dev/null)
    jq -e 'type == "object"' <<< "$state" > /dev/null 2>&1 || state='{}'
    new_state=$state

    while IFS= read -r event; do
        event_id=$(echo "$event" | jq -r '.EventId // ""')
        if [ -z "$event_id" ]; then
            log "WARN: Skipping event without EventId: ${event:0:200}"
            continue
        fi
        event_type=$(echo "$event" | jq -r '.EventType // "Unknown"')
        event_status=$(echo "$event" | jq -r '.EventStatus // "Unknown"')
        event_source=$(echo "$event" | jq -r '.EventSource // "Unknown"')
        not_before=$(echo "$event" | jq -r '.NotBefore // ""')
        duration=$(echo "$event" | jq -r '.DurationInSeconds // -1')
        resources=$(echo "$event" | jq -c '.Resources // []')
        seen_ids="${seen_ids}${event_id} "

        known_status=$(echo "$new_state" | jq -r --arg id "$event_id" '.[$id].status // ""')
        if [ "$known_status" = "$event_status" ]; then
            continue
        fi

        log "Event ${event_id}: ${event_type} ${event_status} (${event_source}), Resources=$(echo "$resources" | jq -r 'join(",")'), NotBefore=${not_before}, DurationInSeconds=${duration}"

        if [ "$logged_in" -eq 0 ]; then
            weka_login || { failures=$((failures + 1)); continue; }
            logged_in=1
        fi

        message=$(compose_event_message "$event_type" "$event_status" "$event_source" "$resources" "$not_before" "$event_id" "$duration")
        if send_weka_event "$message"; then
            new_state=$(echo "$new_state" | jq -c --arg id "$event_id" --arg status "$event_status" \
                --arg type "$event_type" --argjson resources "$resources" --arg nb "$not_before" --arg now "$(utc_now)" \
                '.[$id] = {status: $status, type: $type, resources: $resources, not_before: $nb, updated: $now}')
        else
            failures=$((failures + 1))
        fi
    done < <(echo "$document" | jq -c '.Events[]?')

    # Events that are no longer in the document have completed or were cancelled
    for event_id in $(echo "$new_state" | jq -r 'keys[]'); do
        case "$seen_ids" in
            *" ${event_id} "*) continue ;;
        esac
        event_type=$(echo "$new_state" | jq -r --arg id "$event_id" '.[$id].type // "Unknown"')
        resources=$(echo "$new_state" | jq -c --arg id "$event_id" '.[$id].resources // []')
        log "Event ${event_id}: ${event_type} no longer in the IMDS document, Resources=$(echo "$resources" | jq -r 'join(",")')"

        if [ "$logged_in" -eq 0 ]; then
            weka_login || { failures=$((failures + 1)); continue; }
            logged_in=1
        fi

        message=$(compose_removed_message "$event_type" "$resources" "$event_id")
        if send_weka_event "$message"; then
            new_state=$(echo "$new_state" | jq -c --arg id "$event_id" 'del(.[$id])')
        else
            failures=$((failures + 1))
        fi
    done

    write_state_file "$new_state"
    [ "$failures" -eq 0 ]
}

# Main monitoring loop
main() {
    local last_hash="" retry_pending=0
    local response curl_rc current_hash

    log "Starting WEKA maintenance event monitor"
    log "Cluster: $CLUSTER_NAME, Check Interval: ${CHECK_INTERVAL}s, IMDS: $IMDS_ENDPOINT"
    init_state_file

    while true; do
        response=$(curl -s --max-time "$IMDS_TIMEOUT" -H "Metadata: true" "$IMDS_ENDPOINT" 2>/dev/null)
        curl_rc=$?

        if [ "$curl_rc" -ne 0 ] || [ -z "$response" ]; then
            log "ERROR: Failed to query IMDS endpoint (curl exit code $curl_rc)"
        elif ! echo "$response" | jq -e '.Events | type == "array"' > /dev/null 2>&1; then
            log "ERROR: Unexpected IMDS response: ${response:0:200}"
        else
            current_hash=$(echo "$response" | md5sum | cut -d' ' -f1)

            # Process when the document changed, or when a previous report failed
            if [ "$current_hash" != "$last_hash" ] || [ "$retry_pending" -ne 0 ]; then
                if [ "$current_hash" != "$last_hash" ]; then
                    log "IMDS document changed: DocumentIncarnation=$(echo "$response" | jq -r '.DocumentIncarnation // "?"'), $(echo "$response" | jq -r '.Events | length') scheduled event(s)"
                    append_raw_document "$response"
                fi
                if process_document "$response"; then
                    retry_pending=0
                else
                    retry_pending=1
                    log "Not all events were reported, retrying in ${CHECK_INTERVAL}s"
                fi
                last_hash="$current_hash"
            fi
        fi

        sleep "$CHECK_INTERVAL"
    done
}

# Run main function
main
