package instance_refresh

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
	"weka-deployment/common"

	"github.com/weka/go-cloud-lib/connectors"
	"github.com/weka/go-cloud-lib/instance_refresh"
	"github.com/weka/go-cloud-lib/lib/jrpc"
	"github.com/weka/go-cloud-lib/lib/weka"
	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
)

const (
	instanceRefreshStateBlobName = "instance-refresh-state"
	defaultScaleUpInterval       = 2
)

// Request structure for instance refresh API
type InstanceRefreshRequest struct {
	Action          string `json:"action"`            // "start", "status", "cancel", or "resume"
	ScaleUpInterval *int   `json:"scale_up_interval"` // optional, default 2
}

// Handler handles instance refresh HTTP requests (start, status, cancel, resume)
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
		action != protocol.InstanceRefreshActionCancel &&
		action != protocol.InstanceRefreshActionResume {
		err := fmt.Errorf("invalid action: %s, must be 'start', 'status', 'cancel', or 'resume'", req.Action)
		logger.Error().Err(err).Send()
		common.WriteErrorResponse(w, err)
		return
	}

	// Read current cluster state
	clusterState, err := common.ReadState(ctx, clusterStateParams)
	if err != nil {
		logger.Error().Err(err).Msg("cannot read cluster state")
		common.WriteErrorResponse(w, err)
		return
	}

	// Read instance refresh state (may not exist)
	instanceRefreshState, err := readInstanceRefreshState(ctx, instanceRefreshStateParams)
	if err != nil {
		logger.Debug().Msgf("no existing instance refresh state: %v", err)
		// This is fine - state may not exist yet
	}

	var response *protocol.InstanceRefreshProgress

	switch action {
	case protocol.InstanceRefreshActionStart:
		response, err = handleStart(ctx, req, instanceRefreshState, clusterState,
			clusterStateParams, instanceRefreshStateParams, logger)

	case protocol.InstanceRefreshActionStatus:
		response, err = handleStatus(ctx, instanceRefreshState, clusterState, logger)

	case protocol.InstanceRefreshActionCancel:
		response, err = handleCancel(ctx, instanceRefreshState, clusterState,
			clusterStateParams, instanceRefreshStateParams, logger)

	case protocol.InstanceRefreshActionResume:
		response, err = handleResume(ctx, instanceRefreshState, clusterState,
			clusterStateParams, instanceRefreshStateParams, logger)
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

	ctx := r.Context()
	logger := logging.LoggerFromCtx(ctx)

	nvmesNum, _ := strconv.Atoi(nvmesNumStr)

	// Calculate containers per VM based on dedicated frontend container setting
	frontendContainerCoresNum, _ := strconv.Atoi(frontendContainerCoresNumStr)
	containersPerVm := 2
	if frontendContainerCoresNum > 0 {
		containersPerVm = 3
	}

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

	// Read instance refresh state
	state, err := readInstanceRefreshState(ctx, instanceRefreshStateParams)
	if err != nil {
		logger.Debug().Msgf("no instance refresh state: %v", err)
		common.WriteSuccessResponse(w, map[string]string{"status": "idle", "message": "no instance refresh in progress"})
		return
	}

	// Check if refresh is in progress
	if !instance_refresh.IsInProgress(state) {
		logger.Debug().Msgf("instance refresh not in progress (phase: %s)", state.Phase)
		common.WriteSuccessResponse(w, map[string]string{"status": string(state.Phase), "message": "instance refresh not in progress"})
		return
	}

	// Read current cluster state
	clusterState, err := common.ReadState(ctx, clusterStateParams)
	if err != nil {
		logger.Error().Err(err).Msg("cannot read cluster state")
		common.WriteErrorResponse(w, err)
		return
	}

	// Get current VMSS instance IDs
	currentInstanceIds, err := getVmssInstanceIds(ctx, vmssParams)
	if err != nil {
		logger.Error().Err(err).Msg("cannot get VMSS instance IDs")
		common.WriteErrorResponse(w, err)
		return
	}

	// Get Weka status
	wekaStatus, err := getWekaStatus(ctx, vmssParams, keyVaultUri)
	if err != nil {
		// Store error in state and save it
		errMsg := fmt.Sprintf("cannot get Weka status: %v", err)
		state.LastError = &errMsg
		if writeErr := writeInstanceRefreshState(ctx, instanceRefreshStateParams, state); writeErr != nil {
			logger.Error().Err(writeErr).Msg("failed to save state with error")
		}
		logger.Warn().Err(err).Msg("cannot get Weka status, will retry on next worker run")
		common.WriteSuccessResponse(w, map[string]string{"status": "waiting", "message": errMsg})
		return
	}

	// Advance state machine
	stateChanged, err := advanceStateMachine(ctx, state, clusterState, wekaStatus, currentInstanceIds,
		nvmesNum, containersPerVm, clusterStateParams, logger, false)
	if err != nil {
		// Store error in state and save it
		errMsg := err.Error()
		state.LastError = &errMsg
		if writeErr := writeInstanceRefreshState(ctx, instanceRefreshStateParams, state); writeErr != nil {
			logger.Error().Err(writeErr).Msg("failed to save state with error")
		}
		logger.Warn().Err(err).Msg("state machine error, will retry on next worker run")
		common.WriteSuccessResponse(w, map[string]string{"status": "retrying", "message": errMsg})
		return
	}

	// Clear any previous error on success and save if state changed or error was cleared
	hadError := state.LastError != nil
	state.LastError = nil

	if stateChanged || hadError {
		if err := writeInstanceRefreshState(ctx, instanceRefreshStateParams, state); err != nil {
			logger.Error().Err(err).Msg("failed to save instance refresh state")
			common.WriteErrorResponse(w, fmt.Errorf("failed to save instance refresh state: %v", err))
			return
		}
		if stateChanged {
			logger.Info().Msgf("state advanced: phase=%s, iteration=%d/%d", state.Phase, state.CurrentIteration, state.TotalIterations)
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
		errStr := err.Error()
		return &protocol.InstanceRefreshProgress{
			Phase:     protocol.InstanceRefreshPhaseIdle,
			LastError: &errStr,
		}, nil
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

	currentInstanceIds, err := getVmssInstanceIds(ctx, vmssParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get current instance IDs: %v", err)
	}

	// Initialize new state
	newState := instance_refresh.InitializeState("", originalSize, scaleUpInterval, currentInstanceIds)

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

	// No instance refresh in progress
	if state == nil || state.Phase == protocol.InstanceRefreshPhaseIdle {
		return &protocol.InstanceRefreshProgress{
			Phase: protocol.InstanceRefreshPhaseIdle,
		}, nil
	}

	subscriptionId := os.Getenv("SUBSCRIPTION_ID")
	resourceGroupName := os.Getenv("RESOURCE_GROUP_NAME")
	prefix := os.Getenv("PREFIX")
	clusterName := os.Getenv("CLUSTER_NAME")
	keyVaultUri := os.Getenv("KEY_VAULT_URI")
	nvmesNumStr := os.Getenv("NVMES_NUM")
	frontendContainerCoresNumStr := os.Getenv("FRONTEND_CONTAINER_CORES_NUM")

	nvmesNum, _ := strconv.Atoi(nvmesNumStr)

	frontendContainerCoresNum, _ := strconv.Atoi(frontendContainerCoresNumStr)
	containersPerVm := 2
	if frontendContainerCoresNum > 0 {
		containersPerVm = 3
	}

	vmScaleSetName := common.GetVmScaleSetName(prefix, clusterName)

	vmssParams := &common.ScaleSetParams{
		SubscriptionId:    subscriptionId,
		ResourceGroupName: resourceGroupName,
		ScaleSetName:      vmScaleSetName,
	}

	currentInstanceIds, err := getVmssInstanceIds(ctx, vmssParams)
	if err != nil {
		logger.Warn().Err(err).Msg("cannot get VMSS instance IDs for status, using empty list")
		currentInstanceIds = []string{}
	}

	progress := instance_refresh.CalculateProgress(state, currentInstanceIds)

	// Check if the current phase's work is complete using the same state machine logic as the worker
	if instance_refresh.IsInProgress(state) {
		wekaStatus, err := getWekaStatus(ctx, vmssParams, keyVaultUri)
		if err != nil {
			// If we can't get Weka status, indicate why phase status couldn't be calculated
			logger.Debug().Err(err).Msg("cannot get Weka status for phase check")
			errMsg := fmt.Sprintf("phase status may be stale: %v", err)
			progress.LastError = &errMsg
		} else {
			// Run state machine in dry-run mode to check if phase would advance
			stateChanged, _ := advanceStateMachine(ctx, state, clusterState, wekaStatus, currentInstanceIds,
				nvmesNum, containersPerVm, common.BlobObjParams{}, logger, true)
			// If state would change, the current phase's work is complete
			if stateChanged {
				progress.PhaseCompleted = true
			}
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

func handleResume(
	ctx context.Context,
	state *protocol.InstanceRefreshState,
	clusterState protocol.ClusterState,
	clusterStateParams, instanceRefreshStateParams common.BlobObjParams,
	logger *logging.Logger,
) (*protocol.InstanceRefreshProgress, error) {

	// Check if there's anything to resume
	if state == nil {
		errStr := "no instance refresh to resume"
		return &protocol.InstanceRefreshProgress{
			Phase:     protocol.InstanceRefreshPhaseIdle,
			LastError: &errStr,
		}, nil
	}

	// Can only resume from cancelled or failed state
	if state.Phase != protocol.InstanceRefreshPhaseCancelled && state.Phase != protocol.InstanceRefreshPhaseFailed {
		errStr := fmt.Sprintf("cannot resume from phase '%s'", state.Phase)
		return &protocol.InstanceRefreshProgress{
			Phase:     state.Phase,
			LastError: &errStr,
		}, nil
	}

	logger.Info().Msgf("Resuming instance refresh from iteration %d/%d",
		state.CurrentIteration, state.TotalIterations)

	// Clear any previous error and CompletedAt (since we're resuming)
	state.LastError = nil
	state.CompletedAt = nil

	// Resume by starting scale up for current iteration
	instance_refresh.ResumeRefresh(state)

	// Calculate scaled up size for current iteration
	scaledUpSize := instance_refresh.GetScaledUpSize(state.OriginalSize, state.ScaleUpInterval,
		state.CurrentIteration, state.TotalIterations)

	// Update cluster desired size (scale up)
	clusterState.DesiredSize = scaledUpSize
	if err := common.WriteState(ctx, clusterStateParams, clusterState); err != nil {
		return nil, fmt.Errorf("failed to update cluster state: %v", err)
	}

	// Save instance refresh state
	if err := writeInstanceRefreshState(ctx, instanceRefreshStateParams, state); err != nil {
		return nil, fmt.Errorf("failed to save instance refresh state: %v", err)
	}

	// Get current instance IDs for progress reporting
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

	currentInstanceIds, _ := getVmssInstanceIds(ctx, vmssParams)

	progress := instance_refresh.CalculateProgress(state, currentInstanceIds)
	return &progress, nil
}

// advanceStateMachine advances the instance refresh state machine
// When dryRun is true, it calculates what would happen without writing any state
func advanceStateMachine(
	ctx context.Context,
	state *protocol.InstanceRefreshState,
	clusterState protocol.ClusterState,
	wekaStatus protocol.WekaStatus,
	currentInstanceIds []string,
	drivesPerVm int,
	containersPerVm int,
	clusterStateParams common.BlobObjParams,
	logger *logging.Logger,
	dryRun bool,
) (stateChanged bool, err error) {

	switch state.Phase {
	case protocol.InstanceRefreshPhaseProvisioning,
		protocol.InstanceRefreshPhaseWaitingWekaAfterScaleUp:

		// Calculate expected scaled up size
		scaledUpSize := instance_refresh.GetScaledUpSize(state.OriginalSize, state.ScaleUpInterval,
			state.CurrentIteration, state.TotalIterations)

		// First, verify the VMSS instance count has reached the scaled up size
		if len(currentInstanceIds) < scaledUpSize {
			if state.Phase != protocol.InstanceRefreshPhaseProvisioning {
				state.Phase = protocol.InstanceRefreshPhaseProvisioning
				state.UpdatedAt = time.Now()
				return true, nil
			}
			if !dryRun {
				logger.Debug().Msgf("Waiting for instance count to reach %d (currently %d)",
					scaledUpSize, len(currentInstanceIds))
			}
			return false, nil
		}

		// Instances are up, now wait for Weka to reach desired state
		if state.Phase != protocol.InstanceRefreshPhaseWaitingWekaAfterScaleUp {
			instance_refresh.TransitionToWaitingWekaAfterScaleUp(state)
			return true, nil
		}

		// Check if cluster is healthy at scaled up size
		err := instance_refresh.CheckClusterHealthy(wekaStatus, scaledUpSize, containersPerVm, drivesPerVm)
		if err != nil {
			if !dryRun {
				logger.Debug().Msgf("Waiting for cluster to be healthy at scaled up size: %v", err)
			}
			return false, nil
		}

		// Scale up complete, transition to waiting for Weka at original size
		if !dryRun {
			logger.Info().Msgf("Scale up complete (iteration %d/%d), %d instances running, starting scale down",
				state.CurrentIteration, state.TotalIterations, len(currentInstanceIds))
		}

		instance_refresh.TransitionToWaitingWekaAfterScaleDown(state)

		// Update cluster desired size back to original (triggers scale down)
		if !dryRun {
			clusterState.DesiredSize = state.OriginalSize
			if err := common.WriteState(ctx, clusterStateParams, clusterState); err != nil {
				return true, fmt.Errorf("failed to update cluster state for scale down: %v", err)
			}
		}

		return true, nil

	case protocol.InstanceRefreshPhaseWaitingWekaAfterScaleDown:

		// Check if cluster is healthy at original size
		err := instance_refresh.CheckClusterHealthy(wekaStatus, state.OriginalSize, containersPerVm, drivesPerVm)
		if err != nil {
			if !dryRun {
				logger.Debug().Msgf("Waiting for Weka to reach desired state at original size: %v", err)
			}
			return false, nil
		}

		// Weka is ready, now verify instances are terminated
		if !dryRun {
			logger.Info().Msgf("Weka ready at original size (iteration %d/%d), verifying instance termination",
				state.CurrentIteration, state.TotalIterations)
		}

		instance_refresh.TransitionToTerminating(state)
		return true, nil

	case protocol.InstanceRefreshPhaseTerminating:

		// Calculate expected replaced count for this iteration
		expectedReplaced := state.CurrentIteration * state.ScaleUpInterval
		if expectedReplaced > len(state.OriginalInstanceIds) {
			expectedReplaced = len(state.OriginalInstanceIds)
		}
		actualReplaced := countReplacedInstances(state.OriginalInstanceIds, currentInstanceIds)

		// Verify the expected number of old instances have been replaced and instance count is correct
		if actualReplaced < expectedReplaced || len(currentInstanceIds) != state.OriginalSize {
			if !dryRun {
				logger.Debug().Msgf("Waiting for old instances to be terminated: expected %d replaced (actual %d), expected %d instances (actual %d)",
					expectedReplaced, actualReplaced, state.OriginalSize, len(currentInstanceIds))
			}
			return false, nil
		}

		// Scale down complete
		if !dryRun {
			logger.Info().Msgf("Scale down complete (iteration %d/%d), replaced %d instances",
				state.CurrentIteration, state.TotalIterations, actualReplaced)
		}

		instance_refresh.MarkIterationComplete(state)

		// Check if more iterations needed
		if instance_refresh.ShouldContinue(state) {
			instance_refresh.StartNextIteration(state)

			// Calculate next scaled up size
			scaledUpSize := instance_refresh.GetScaledUpSize(state.OriginalSize, state.ScaleUpInterval,
				state.CurrentIteration, state.TotalIterations)

			if !dryRun {
				logger.Info().Msgf("Starting iteration %d/%d, scaling up to %d",
					state.CurrentIteration, state.TotalIterations, scaledUpSize)

				// Update cluster desired size (scale up)
				clusterState.DesiredSize = scaledUpSize
				if err := common.WriteState(ctx, clusterStateParams, clusterState); err != nil {
					return true, fmt.Errorf("failed to update cluster state for next iteration: %v", err)
				}
			}
		} else {
			// All iterations complete
			instance_refresh.MarkCompleted(state)
			if !dryRun {
				logger.Info().Msg("Instance refresh completed successfully")
			}
		}

		return true, nil
	}

	return false, nil
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

func writeInstanceRefreshState(ctx context.Context, params common.BlobObjParams, state *protocol.InstanceRefreshState) error {
	state.UpdatedAt = time.Now()

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal instance refresh state: %v", err)
	}

	return common.WriteBlobObject(ctx, params, data)
}

// countReplacedInstances counts how many original instances are no longer in the current list
func countReplacedInstances(originalIds, currentIds []string) int {
	currentSet := make(map[string]struct{}, len(currentIds))
	for _, id := range currentIds {
		currentSet[id] = struct{}{}
	}

	replaced := 0
	for _, id := range originalIds {
		if _, exists := currentSet[id]; !exists {
			replaced++
		}
	}
	return replaced
}

func getVmssInstanceIds(ctx context.Context, vmssParams *common.ScaleSetParams) ([]string, error) {
	vms, err := common.GetScaleSetVmsExpandedView(ctx, vmssParams)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(vms))
	for _, vm := range vms {
		ids = append(ids, vm.InstanceID)
	}
	return ids, nil
}

func getWekaStatus(ctx context.Context, vmssParams *common.ScaleSetParams, keyVaultUri string) (protocol.WekaStatus, error) {
	logger := logging.LoggerFromCtx(ctx)

	credentials, err := common.GetWekaClusterCredentials(ctx, keyVaultUri)
	if err != nil {
		return protocol.WekaStatus{}, fmt.Errorf("failed to get Weka credentials: %v", err)
	}

	jrpcBuilder := func(ip string) *jrpc.BaseClient {
		return connectors.NewJrpcClient(ctx, ip, weka.ManagementJrpcPort, credentials.Username, credentials.Password)
	}

	vmIps, err := common.GetVmsPrivateIps(ctx, vmssParams)
	if err != nil {
		return protocol.WekaStatus{}, fmt.Errorf("failed to get VM IPs: %v", err)
	}

	ips := make([]string, 0, len(vmIps))
	for _, ip := range vmIps {
		ips = append(ips, ip)
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rng.Shuffle(len(ips), func(i, j int) { ips[i], ips[j] = ips[j], ips[i] })

	jpool := &jrpc.Pool{
		Ips:     ips,
		Clients: map[string]*jrpc.BaseClient{},
		Active:  "",
		Builder: jrpcBuilder,
		Ctx:     ctx,
	}

	var rawWekaStatus json.RawMessage
	if err := jpool.Call(weka.JrpcStatus, struct{}{}, &rawWekaStatus); err != nil {
		return protocol.WekaStatus{}, fmt.Errorf("failed to call Weka status: %v", err)
	}

	var wekaStatus protocol.WekaStatus
	if err := json.Unmarshal(rawWekaStatus, &wekaStatus); err != nil {
		return protocol.WekaStatus{}, fmt.Errorf("failed to unmarshal Weka status: %v", err)
	}

	logger.Debug().Msgf("Weka status: io_status=%s, status=%s, backends=%d, drives=%d",
		wekaStatus.IoStatus, wekaStatus.Status, wekaStatus.Hosts.Backends.Active, wekaStatus.Drives.Active)

	return wekaStatus, nil
}
