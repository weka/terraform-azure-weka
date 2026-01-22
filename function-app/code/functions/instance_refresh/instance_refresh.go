package instance_refresh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
	"weka-deployment/common"
	"weka-deployment/functions/status"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/weka/go-cloud-lib/instance_refresh"
	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
)

const (
	instanceRefreshStateBlobName = "instance-refresh-state"
	defaultScaleUpInterval       = 2
)

// Request structure for instance refresh API
type InstanceRefreshRequest struct {
	Action          string `json:"action"`            // "start", "status", or "cancel"
	ScaleUpInterval *int   `json:"scale_up_interval"` // optional, default 2
}

// Handler handles instance refresh HTTP requests (start, status, cancel)
// Status is read-only - it just returns current state without advancing the state machine
func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	stateBlobName := os.Getenv("STATE_BLOB_NAME")

	clusterStateParams := common.BlobObjParams{
		StorageName:   stateStorageName,
		ContainerName: stateContainerName,
		BlobName:      stateBlobName,
	}

	instanceRefreshStateParams := common.BlobObjParams{
		StorageName:   stateStorageName,
		ContainerName: stateContainerName,
		BlobName:      instanceRefreshStateBlobName,
	}

	// Parse request
	var invokeRequest common.InvokeRequest
	if err := json.NewDecoder(r.Body).Decode(&invokeRequest); err != nil {
		err = fmt.Errorf("cannot decode the request: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	var reqData map[string]interface{}
	if err := json.Unmarshal(invokeRequest.Data["req"], &reqData); err != nil {
		err = fmt.Errorf("cannot unmarshal the request data: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	var req InstanceRefreshRequest
	if err := json.Unmarshal([]byte(reqData["Body"].(string)), &req); err != nil {
		err = fmt.Errorf("cannot unmarshal the request body: %v", err)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	// Validate action
	action := protocol.InstanceRefreshAction(req.Action)
	if action != protocol.InstanceRefreshActionStart &&
		action != protocol.InstanceRefreshActionStatus &&
		action != protocol.InstanceRefreshActionCancel {
		err := fmt.Errorf("invalid action: %s, must be 'start', 'status', or 'cancel'", req.Action)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	var response *protocol.InstanceRefreshProgress
	var err error

	// Status is read-only, no locking needed
	if action == protocol.InstanceRefreshActionStatus {
		clusterState, readErr := common.ReadState(ctx, clusterStateParams)
		if readErr != nil {
			logger.Error().Err(readErr).Msg("cannot read cluster state")
			common.WriteErrorResponse(w, readErr)
			return
		}
		instanceRefreshState, _ := readInstanceRefreshState(ctx, instanceRefreshStateParams)
		response, err = handleStatus(ctx, instanceRefreshState, clusterState, logger)
	} else {
		// Start and Cancel modify state, need locking
		leaseId, lockErr := common.LockContainer(ctx, stateStorageName, stateContainerName)
		if lockErr != nil {
			logger.Error().Err(lockErr).Msg("failed to acquire lock")
			common.WriteErrorResponse(w, fmt.Errorf("failed to acquire lock: %v", lockErr))
			return
		}
		defer common.UnlockContainer(ctx, stateStorageName, stateContainerName, leaseId)

		// Read states under lock
		clusterState, readErr := common.ReadState(ctx, clusterStateParams)
		if readErr != nil {
			logger.Error().Err(readErr).Msg("cannot read cluster state")
			common.WriteErrorResponse(w, readErr)
			return
		}
		instanceRefreshState, readErr := readInstanceRefreshState(ctx, instanceRefreshStateParams)
		if readErr != nil {
			// Check if error is specifically "blob not found" (expected for first refresh)
			var responseErr *azcore.ResponseError
			isBlobNotFound := errors.As(readErr, &responseErr) && responseErr.ErrorCode == "BlobNotFound"

			if isBlobNotFound {
				// Blob doesn't exist - this is expected for the first refresh
				logger.Debug().Msg("no existing instance refresh state blob, will create new")
			} else {
				// Transient error - fail to prevent accidentally overwriting an existing refresh
				logger.Error().Err(readErr).Msg("cannot read instance refresh state")
				common.WriteErrorResponse(w, fmt.Errorf("cannot read instance refresh state (may be transient): %v", readErr))
				return
			}
		}

		switch action {
		case protocol.InstanceRefreshActionStart:
			response, err = handleStart(ctx, req, instanceRefreshState, clusterState,
				clusterStateParams, instanceRefreshStateParams, logger)
		case protocol.InstanceRefreshActionCancel:
			response, err = handleCancel(ctx, instanceRefreshState, clusterState,
				clusterStateParams, instanceRefreshStateParams, logger)
		}
	}

	if err != nil {
		common.WriteErrorResponse(w, err)
		return
	}

	common.WriteSuccessResponse(w, response)
}

// WorkerHandler is called periodically by Logic App to advance the instance refresh state machine
// This is the only function that reads Weka status and advances the state machine
func WorkerHandler(w http.ResponseWriter, r *http.Request) {
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	stateContainerName := os.Getenv("STATE_CONTAINER_NAME")
	stateStorageName := os.Getenv("STATE_STORAGE_NAME")
	stateBlobName := os.Getenv("STATE_BLOB_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")
	nvmesNumStr := os.Getenv("NVMES_NUM")
	frontendContainerCoresNumStr := os.Getenv("FRONTEND_CONTAINER_CORES_NUM")
	nfsGatewaysNumStr := os.Getenv("NFS_PROTOCOL_GATEWAYS_NUM")
	smbGatewaysNumStr := os.Getenv("SMB_PROTOCOL_GATEWAYS_NUM")
	s3GatewaysNumStr := os.Getenv("S3_PROTOCOL_GATEWAYS_NUM")

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	nvmesNum, _ := strconv.Atoi(nvmesNumStr)

	// Calculate containers per VM based on dedicated frontend container setting
	frontendContainerCoresNum, _ := strconv.Atoi(frontendContainerCoresNumStr)
	containersPerVm := 2
	if frontendContainerCoresNum > 0 {
		containersPerVm = 3
	}

	// Calculate protocol gateway containers (each gateway VM adds 1 backend container)
	nfsGatewaysNum, _ := strconv.Atoi(nfsGatewaysNumStr)
	smbGatewaysNum, _ := strconv.Atoi(smbGatewaysNumStr)
	s3GatewaysNum, _ := strconv.Atoi(s3GatewaysNumStr)
	protocolGatewayContainers := nfsGatewaysNum + smbGatewaysNum + s3GatewaysNum

	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)

	clusterStateParams := common.BlobObjParams{
		StorageName:   stateStorageName,
		ContainerName: stateContainerName,
		BlobName:      stateBlobName,
	}

	instanceRefreshStateParams := common.BlobObjParams{
		StorageName:   stateStorageName,
		ContainerName: stateContainerName,
		BlobName:      instanceRefreshStateBlobName,
	}

	vmssParams := &common.ScaleSetParams{
		SubscriptionId:    subscriptionId,
		ResourceGroupName: resourceGroupName,
		ScaleSetName:      vmScaleSetName,
	}

	// Early check without lock - if no refresh in progress, return immediately
	state, err := readInstanceRefreshState(ctx, instanceRefreshStateParams)
	if err != nil {
		logger.Debug().Msgf("no instance refresh state: %v", err)
		common.WriteSuccessResponse(w, map[string]string{"status": "idle", "message": "no instance refresh in progress"})
		return
	}
	if !instance_refresh.IsInProgress(state) {
		logger.Debug().Msgf("instance refresh not in progress (phase: %s)", state.Phase)
		common.WriteSuccessResponse(w, map[string]string{"status": string(state.Phase), "message": "instance refresh not in progress"})
		return
	}

	// Acquire lock for state modifications
	leaseId, err := common.LockContainer(ctx, stateStorageName, stateContainerName)
	if err != nil {
		logger.Error().Err(err).Msg("failed to acquire lock")
		common.WriteErrorResponse(w, fmt.Errorf("failed to acquire lock: %v", err))
		return
	}
	defer common.UnlockContainer(ctx, stateStorageName, stateContainerName, leaseId)

	// Re-read states under lock with ETag for optimistic concurrency
	stateWithETag, err := readInstanceRefreshStateWithETag(ctx, instanceRefreshStateParams)
	if err != nil {
		logger.Debug().Msgf("no instance refresh state after lock: %v", err)
		common.WriteSuccessResponse(w, map[string]string{"status": "idle", "message": "no instance refresh in progress"})
		return
	}
	state = stateWithETag.State
	stateETag := stateWithETag.ETag // Used for optimistic concurrency on writes
	if !instance_refresh.IsInProgress(state) {
		logger.Debug().Msgf("instance refresh not in progress after lock (phase: %s)", state.Phase)
		common.WriteSuccessResponse(w, map[string]string{"status": string(state.Phase), "message": "instance refresh not in progress"})
		return
	}

	clusterState, err := common.ReadState(ctx, clusterStateParams)
	if err != nil {
		logger.Error().Err(err).Msg("cannot read cluster state")
		common.WriteErrorResponse(w, err)
		return
	}

	// Get current VMSS instance IDs
	currentInstanceIds, err := common.GetScaleSetVmInstanceIds(ctx, vmssParams)
	if err != nil {
		logger.Error().Err(err).Msg("cannot get VMSS instance IDs")
		common.WriteErrorResponse(w, err)
		return
	}

	// Get Weka status
	wekaStatus, err := status.GetWekaStatus(ctx, vmssParams, keyVaultUri)
	if err != nil {
		// Store error in state and save it
		errMsg := fmt.Sprintf("cannot get Weka status: %v", err)
		state.LastError = &errMsg
		if writeErr := writeInstanceRefreshStateWithETag(ctx, instanceRefreshStateParams, state, stateETag); writeErr != nil {
			if errors.Is(writeErr, common.ErrBlobModified) {
				logger.Warn().Msg("state was modified by another process, skipping write")
			} else {
				logger.Error().Err(writeErr).Msg("failed to save state with error")
			}
		}
		logger.Warn().Err(err).Msg("cannot get Weka status, will retry on next worker run")
		common.WriteSuccessResponse(w, map[string]string{"status": "waiting", "message": errMsg})
		return
	}

	// Advance state machine
	stateChanged, newDesiredSize, err := instance_refresh.AdvanceStateMachine(state, wekaStatus, currentInstanceIds,
		nvmesNum, containersPerVm, protocolGatewayContainers)

	if err != nil {
		// Store error in state and save it
		errMsg := err.Error()
		state.LastError = &errMsg
		if writeErr := writeInstanceRefreshStateWithETag(ctx, instanceRefreshStateParams, state, stateETag); writeErr != nil {
			if errors.Is(writeErr, common.ErrBlobModified) {
				logger.Warn().Msg("state was modified by another process, skipping error state write")
			} else {
				logger.Error().Err(writeErr).Msg("failed to save state with error")
			}
		}
		logger.Warn().Err(err).Msg("state machine error, will retry on next worker run")
		common.WriteSuccessResponse(w, map[string]string{"status": "retrying", "message": errMsg})
		return
	}

	// Clear any previous error on success
	hadError := state.LastError != nil
	state.LastError = nil

	// Write instance refresh state FIRST (before cluster state)
	// This ensures that if cluster state write fails, next run will retry with correct phase
	// Use ETag to prevent overwriting state modified by another process (optimistic concurrency)
	if stateChanged || hadError {
		if err := writeInstanceRefreshStateWithETag(ctx, instanceRefreshStateParams, state, stateETag); err != nil {
			if errors.Is(err, common.ErrBlobModified) {
				// Another process modified the state - our changes are stale, don't apply them
				logger.Warn().Msg("state was modified by another process (lease likely expired), discarding stale changes")
				common.WriteErrorResponse(w, fmt.Errorf("state conflict: another process modified the state"))
				return
			}
			logger.Error().Err(err).Msg("failed to save instance refresh state")
			common.WriteErrorResponse(w, fmt.Errorf("failed to save instance refresh state: %v", err))
			return
		}
		if stateChanged {
			logger.Info().Msgf("state advanced: phase=%s, iteration=%d/%d", state.Phase, state.CurrentIteration, state.TotalIterations)
		}
	}

	// Update cluster desired size SECOND (only if value differs)
	// AdvanceStateMachine always returns expected DesiredSize for idempotency
	if newDesiredSize != nil && clusterState.DesiredSize != *newDesiredSize {
		logger.Info().Msgf("Setting cluster desired size from %d to %d", clusterState.DesiredSize, *newDesiredSize)
		clusterState.DesiredSize = *newDesiredSize
		if err := common.WriteState(ctx, clusterStateParams, clusterState); err != nil {
			logger.Error().Err(err).Msg("failed to update cluster desired size")
			// Don't fail the request - instance refresh state is already saved
			// Next run will retry setting the correct DesiredSize (idempotent)
			logger.Warn().Msg("cluster desired size will be corrected on next worker run")
		}
	}

	common.WriteSuccessResponse(w, map[string]string{
		"status":    "ok",
		"phase":     string(state.Phase),
		"iteration": fmt.Sprintf("%d/%d", state.CurrentIteration, state.TotalIterations),
	})
}

func handleStart(
	ctx context.Context,
	req InstanceRefreshRequest,
	state *protocol.InstanceRefreshState,
	clusterState protocol.ClusterState,
	clusterStateParams, instanceRefreshStateParams common.BlobObjParams,
	logger *logging.Logger,
) (*protocol.InstanceRefreshProgress, error) {

	if !clusterState.Clusterized {
		return nil, fmt.Errorf("cluster is not clusterized yet, cannot perform instance refresh")
	}

	// Check if already in progress
	if err := instance_refresh.CanTrigger(state); err != nil {
		logger.Info().Msgf("Cannot start new refresh: %v", err)
		errStr := err.Error()
		return &protocol.InstanceRefreshProgress{
			Phase:     protocol.InstanceRefreshPhaseIdle,
			LastError: &errStr,
		}, nil
	}

	// Log what we're overwriting (if anything)
	if state != nil {
		logger.Info().Msgf("Starting new refresh, previous state: phase=%s, iteration=%d/%d, started_at=%s",
			state.Phase, state.CurrentIteration, state.TotalIterations, state.StartedAt.Format(time.RFC3339))
	} else {
		logger.Info().Msg("Starting first refresh (no previous state)")
	}

	// Determine scale up interval
	scaleUpInterval := defaultScaleUpInterval
	if req.ScaleUpInterval != nil && *req.ScaleUpInterval > 0 {
		scaleUpInterval = *req.ScaleUpInterval
	}

	originalSize := clusterState.DesiredSize

	// Validate scale up interval
	if scaleUpInterval > originalSize {
		scaleUpInterval = originalSize
	}

	// Get current VMSS instance IDs for tracking
	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")
	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)

	vmssParams := &common.ScaleSetParams{
		SubscriptionId:    subscriptionId,
		ResourceGroupName: resourceGroupName,
		ScaleSetName:      vmScaleSetName,
	}

	currentInstanceIds, err := common.GetScaleSetVmInstanceIds(ctx, vmssParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get current instance IDs: %v", err)
	}

	// Initialize new state
	newState := instance_refresh.InitializeState(originalSize, scaleUpInterval, currentInstanceIds)

	// Calculate scaled up size for first iteration
	scaledUpSize := instance_refresh.GetScaledUpSize(originalSize, scaleUpInterval, 1, newState.TotalIterations)

	// Update cluster desired size (scale up)
	clusterState.DesiredSize = scaledUpSize
	if err := common.WriteState(ctx, clusterStateParams, clusterState); err != nil {
		return nil, fmt.Errorf("failed to update cluster state: %v", err)
	}

	// Save instance refresh state
	if err := writeInstanceRefreshState(ctx, instanceRefreshStateParams, newState); err != nil {
		return nil, fmt.Errorf("failed to save instance refresh state: %v", err)
	}

	logger.Info().Msgf("Instance refresh started: original_size=%d, scale_up_interval=%d, total_iterations=%d, scaled_up_size=%d",
		originalSize, scaleUpInterval, newState.TotalIterations, scaledUpSize)

	progress := instance_refresh.CalculateProgress(newState, currentInstanceIds)
	return &progress, nil
}

// handleStatus is READ-ONLY - it returns current state and calculates what phase would be next
// The WorkerHandler is responsible for actually advancing and persisting the state machine
func handleStatus(
	ctx context.Context,
	state *protocol.InstanceRefreshState,
	clusterState protocol.ClusterState,
	logger *logging.Logger,
) (*protocol.InstanceRefreshProgress, error) {

	// No instance refresh state exists
	if state == nil || state.Phase == protocol.InstanceRefreshPhaseIdle {
		return &protocol.InstanceRefreshProgress{
			Phase: protocol.InstanceRefreshPhaseIdle,
		}, nil
	}

	// For terminal states, return saved state without fetching live data
	if !instance_refresh.IsInProgress(state) {
		progress := instance_refresh.CalculateProgress(state, nil)
		return &progress, nil
	}

	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")
	nvmesNumStr := os.Getenv("NVMES_NUM")
	frontendContainerCoresNumStr := os.Getenv("FRONTEND_CONTAINER_CORES_NUM")
	nfsGatewaysNumStr := os.Getenv("NFS_PROTOCOL_GATEWAYS_NUM")
	smbGatewaysNumStr := os.Getenv("SMB_PROTOCOL_GATEWAYS_NUM")
	s3GatewaysNumStr := os.Getenv("S3_PROTOCOL_GATEWAYS_NUM")

	nvmesNum, _ := strconv.Atoi(nvmesNumStr)

	frontendContainerCoresNum, _ := strconv.Atoi(frontendContainerCoresNumStr)
	containersPerVm := 2
	if frontendContainerCoresNum > 0 {
		containersPerVm = 3
	}

	// Calculate protocol gateway containers (each gateway VM adds 1 backend container)
	nfsGatewaysNum, _ := strconv.Atoi(nfsGatewaysNumStr)
	smbGatewaysNum, _ := strconv.Atoi(smbGatewaysNumStr)
	s3GatewaysNum, _ := strconv.Atoi(s3GatewaysNumStr)
	protocolGatewayContainers := nfsGatewaysNum + smbGatewaysNum + s3GatewaysNum

	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)

	vmssParams := &common.ScaleSetParams{
		SubscriptionId:    subscriptionId,
		ResourceGroupName: resourceGroupName,
		ScaleSetName:      vmScaleSetName,
	}

	currentInstanceIds, err := common.GetScaleSetVmInstanceIds(ctx, vmssParams)
	if err != nil {
		logger.Warn().Err(err).Msg("cannot get VMSS instance IDs for status, using empty list")
		currentInstanceIds = []string{}
	}

	progress := instance_refresh.CalculateProgress(state, currentInstanceIds)

	// Check if the current phase's work is complete using the same state machine logic as the worker
	wekaStatus, err := status.GetWekaStatus(ctx, vmssParams, keyVaultUri)
	if err != nil {
		logger.Debug().Err(err).Msg("cannot get Weka status for phase check")
		errMsg := fmt.Sprintf("phase status may be stale: %v", err)
		progress.LastError = &errMsg
	} else {
		// Calculate target size based on current phase
		targetSize := state.OriginalSize
		if state.Phase == protocol.InstanceRefreshPhaseProvisioning ||
			state.Phase == protocol.InstanceRefreshPhaseWaitingWekaAfterScaleUp {
			targetSize = instance_refresh.GetScaledUpSize(state.OriginalSize, state.ScaleUpInterval,
				state.CurrentIteration, state.TotalIterations)
		}

		// Add Weka health info to progress
		progress.WekaHealth = &protocol.WekaHealthStatus{
			IoStatus:                wekaStatus.IoStatus,
			ClusterStatus:           wekaStatus.Status,
			BackendContainersActive: wekaStatus.Hosts.Backends.Active,
			BackendContainersTotal:  wekaStatus.Hosts.Backends.Total,
			DrivesActive:            wekaStatus.Drives.Active,
			DrivesTotal:             wekaStatus.Drives.Total,
			TargetBackendContainers: instance_refresh.CalculateTargetContainers(targetSize, containersPerVm, protocolGatewayContainers),
			TargetDrives:            targetSize * nvmesNum,
		}

		// Run state machine on a copy to check if phase would advance (dry-run)
		stateCopy := *state
		stateChanged, _, _ := instance_refresh.AdvanceStateMachine(&stateCopy, wekaStatus, currentInstanceIds,
			nvmesNum, containersPerVm, protocolGatewayContainers)
		if stateChanged {
			progress.PhaseCompleted = true
		}
	}

	return &progress, nil
}

func handleCancel(
	ctx context.Context,
	state *protocol.InstanceRefreshState,
	clusterState protocol.ClusterState,
	clusterStateParams, instanceRefreshStateParams common.BlobObjParams,
	logger *logging.Logger,
) (*protocol.InstanceRefreshProgress, error) {

	// Check if there's anything to cancel
	if state == nil {
		return &protocol.InstanceRefreshProgress{
			Phase: protocol.InstanceRefreshPhaseIdle,
		}, nil
	}

	if !instance_refresh.IsInProgress(state) {
		progress := instance_refresh.CalculateProgress(state, nil)
		return &progress, nil
	}

	logger.Info().Msgf("Cancelling instance refresh (was at phase: %s, iteration: %d/%d)",
		state.Phase, state.CurrentIteration, state.TotalIterations)

	// Restore desired size to original
	if clusterState.DesiredSize != state.OriginalSize {
		logger.Info().Msgf("Restoring cluster desired size from %d to %d", clusterState.DesiredSize, state.OriginalSize)
		clusterState.DesiredSize = state.OriginalSize
		if err := common.WriteState(ctx, clusterStateParams, clusterState); err != nil {
			return nil, fmt.Errorf("failed to restore cluster state: %v", err)
		}
	}

	// Mark as cancelled
	instance_refresh.MarkCancelled(state)

	// Save state
	if err := writeInstanceRefreshState(ctx, instanceRefreshStateParams, state); err != nil {
		return nil, fmt.Errorf("failed to save cancelled state: %v", err)
	}

	progress := instance_refresh.CalculateProgress(state, nil)
	return &progress, nil
}

func readInstanceRefreshState(ctx context.Context, params common.BlobObjParams) (*protocol.InstanceRefreshState, error) {
	data, err := common.ReadBlobObject(ctx, params)
	if err != nil {
		return nil, err
	}

	var state protocol.InstanceRefreshState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance refresh state: %v", err)
	}

	return &state, nil
}

// instanceRefreshStateWithETag holds the state along with its ETag for optimistic concurrency
type instanceRefreshStateWithETag struct {
	State *protocol.InstanceRefreshState
	ETag  *azcore.ETag
}

// readInstanceRefreshStateWithETag reads the state and returns it along with the ETag
func readInstanceRefreshStateWithETag(ctx context.Context, params common.BlobObjParams) (*instanceRefreshStateWithETag, error) {
	result, err := common.ReadBlobObjectWithETag(ctx, params)
	if err != nil {
		return nil, err
	}

	var state protocol.InstanceRefreshState
	if err := json.Unmarshal(result.Data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance refresh state: %v", err)
	}

	return &instanceRefreshStateWithETag{
		State: &state,
		ETag:  result.ETag,
	}, nil
}

func writeInstanceRefreshState(ctx context.Context, params common.BlobObjParams, state *protocol.InstanceRefreshState) error {
	return writeInstanceRefreshStateWithETag(ctx, params, state, nil)
}

// writeInstanceRefreshStateWithETag writes the state with optimistic concurrency.
// If etag is provided, the write will only succeed if the blob hasn't been modified.
// Returns common.ErrBlobModified if there's a conflict.
func writeInstanceRefreshStateWithETag(ctx context.Context, params common.BlobObjParams, state *protocol.InstanceRefreshState, etag *azcore.ETag) error {
	state.UpdatedAt = time.Now()

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal instance refresh state: %v", err)
	}

	return common.WriteBlobObjectWithETag(ctx, params, data, etag)
}
