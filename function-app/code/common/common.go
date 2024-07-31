package common

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/lease"
	"github.com/google/uuid"
	"github.com/weka/go-cloud-lib/lib/types"
	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
	reportLib "github.com/weka/go-cloud-lib/report"
)

const (
	WekaAdminUsername         = "admin"
	WekaAdminPasswordKey      = "weka-password"
	WekaDeploymentUsername    = "weka-deployment"
	WekaDeploymentPasswordKey = "weka-deployment-password"
	// NFS VMs tag
	NfsInterfaceGroupPortKey   = "nfs_interface_group_port"
	NfsInterfaceGroupPortValue = "ready"
)

var (
	userAssignedClientId = os.Getenv("USER_ASSIGNED_CLIENT_ID")
)

type InvokeRequest struct {
	Data     map[string]json.RawMessage
	Metadata map[string]interface{}
}

type InvokeResponse struct {
	Outputs     map[string]interface{}
	Logs        []string
	ReturnValue interface{}
}

type BlobObjParams struct {
	StorageName   string
	ContainerName string
	BlobName      string
}

const FindDrivesScript = `
import json
import sys
for d in json.load(sys.stdin)['disks']:
	if d['isRotational'] or 'nvme' not in d['devPath']: continue
	print(d['devPath'])
`

func getCredential(ctx context.Context) (*azidentity.ManagedIdentityCredential, error) {
	logger := logging.LoggerFromCtx(ctx)

	credOpt := &azidentity.ManagedIdentityCredentialOptions{
		ID: azidentity.ClientID(userAssignedClientId),
	}
	credential, err := azidentity.NewManagedIdentityCredential(credOpt)
	if err != nil {
		logger.Error().CallerSkipFrame(1).Err(err).Msg("failed to get credential")
		return nil, err
	}
	return credential, nil
}

func WriteResponse(w http.ResponseWriter, resData map[string]any, statusCode *int) {
	outputs := make(map[string]any)

	outputs["res"] = resData
	invokeResponse := InvokeResponse{Outputs: outputs, Logs: nil, ReturnValue: nil}

	responseJson, _ := json.Marshal(invokeResponse)

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}

func WriteErrorResponse(w http.ResponseWriter, err error) {
	resData := make(map[string]any)
	errMsg := map[string]string{"error": err.Error()}
	resData["body"] = errMsg

	badReqStatus := http.StatusBadRequest
	WriteResponse(w, resData, &badReqStatus)
}

func WriteSuccessResponse(w http.ResponseWriter, data any) {
	resData := make(map[string]any)
	resData["body"] = data

	successStatus := http.StatusOK
	WriteResponse(w, resData, &successStatus)
}

func leaseContainerAcquire(ctx context.Context, storageAccountName, containerName string, leaseIdIn *string) (leaseIdOut *string, err error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := getCredential(ctx)
	if err != nil {
		return
	}

	containerUrl := getContainerUrl(storageAccountName, containerName)
	containerClient, err := container.NewClient(containerUrl, credential, nil)
	if err != nil {
		logger.Error().Msgf("container.NewClient: %s", err)
		return
	}

	options := &lease.ContainerClientOptions{
		LeaseID: leaseIdIn,
	}
	leaseContainerClient, err := lease.NewContainerClient(containerClient, options)
	if err != nil {
		logger.Error().Msgf("lease.NewContainerClient: %s", err)
		return
	}
	duration := int32(60)
	for i := 1; i < 1000; i++ {
		lease, err2 := leaseContainerClient.AcquireLease(ctx, duration, nil)
		err = err2
		if err != nil {
			if leaseErr, ok := err.(*azcore.ResponseError); ok && leaseErr.ErrorCode == "LeaseAlreadyPresent" {
				logger.Info().Msg("lease in use, will retry in 1 sec")
				time.Sleep(time.Second)
			} else {
				logger.Error().Err(err).Send()
				return
			}
		} else {
			leaseIdOut = lease.LeaseID
			return
		}
	}
	logger.Error().Err(err).Send()
	return
}

func leaseContainerRelease(ctx context.Context, storageAccountName, containerName string, leaseId *string) (err error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := getCredential(ctx)
	if err != nil {
		return
	}

	containerUrl := getContainerUrl(storageAccountName, containerName)
	containerClient, err := container.NewClient(containerUrl, credential, nil)
	if err != nil {
		logger.Error().Msgf("container.NewClient: %s", err)
		return
	}

	options := &lease.ContainerClientOptions{
		LeaseID: leaseId,
	}

	leaseContainerClient, err := lease.NewContainerClient(containerClient, options)
	if err != nil {
		logger.Error().Msgf("lease.NewContainerClient: %s", err)
		return
	}

	_, err = leaseContainerClient.ReleaseLease(ctx, nil)
	if err != nil {
		logger.Error().Msgf("leaseContainerClient.ReleaseLease: %s", err)
		return
	}
	return
}

func LockContainer(ctx context.Context, storageAccountName, containerName string) (*string, error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Debug().Msgf("locking %s", containerName)
	return leaseContainerAcquire(ctx, storageAccountName, containerName, nil)
}

func UnlockContainer(ctx context.Context, storageAccountName, containerName string, leaseId *string) error {
	logger := logging.LoggerFromCtx(ctx)
	logger.Debug().Msgf("unlocking %s", containerName)
	err := leaseContainerRelease(ctx, storageAccountName, containerName, leaseId)
	if err != nil {
		logger.Error().Msgf("Failed leaseContainerRelease: %s", err)
	}
	return err
}

func ReadBlobObject(ctx context.Context, bl BlobObjParams) (state []byte, err error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := getCredential(ctx)
	if err != nil {
		return
	}

	blobClient, err := azblob.NewClient(getBlobUrl(bl.StorageName), credential, nil)
	if err != nil {
		logger.Error().Msgf("azblob.NewClient: %s", err)
		return
	}

	downloadResponse, err := blobClient.DownloadStream(ctx, bl.ContainerName, bl.BlobName, nil)
	if err != nil {
		logger.Error().Msgf("blobClient.DownloadStream: %s", err)
		return
	}

	state, err = io.ReadAll(downloadResponse.Body)
	if err != nil {
		logger.Error().Err(err).Send()
	}

	return

}

func containerExists(ctx context.Context, containerClient *container.Client, storageName, containerName string) (bool, error) {
	_, err := containerClient.GetProperties(ctx, nil)
	if err != nil {
		var responseErr *azcore.ResponseError
		if errors.As(err, &responseErr) && responseErr.ErrorCode == "ContainerNotFound" {
			return false, nil
		}
		err = fmt.Errorf("failed to get container properties: %w", err)
		return false, err
	}
	return true, nil
}

func ensureStorageContainer(ctx context.Context, storageAccountName, containerName string) (err error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := getCredential(ctx)
	if err != nil {
		return
	}

	containerUrl := getContainerUrl(storageAccountName, containerName)
	containerClient, err := container.NewClient(containerUrl, credential, nil)
	if err != nil {
		err = fmt.Errorf("failed to create container client: %v", err)
		return err
	}

	exists, err := containerExists(ctx, containerClient, storageAccountName, containerName)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	if exists {
		logger.Info().Str("container", containerName).Msg("container already exists")
		return
	}

	logger.Info().Str("container", containerName).Msg("container does not exist, creating new container")

	_, err = containerClient.Create(ctx, nil)
	if err != nil {
		err = fmt.Errorf("failed to create container: %v", err)
		logger.Error().Err(err).Send()
	}
	return
}

func blobExists(ctx context.Context, blobClient *blob.Client) (bool, error) {
	_, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		var responseErr *azcore.ResponseError
		if errors.As(err, &responseErr) && responseErr.ErrorCode == "BlobNotFound" {
			return false, nil
		}
		err = fmt.Errorf("failed to get blob properties: %w", err)
		return false, err
	}
	return true, nil
}

func EnsureStateIsCreated(ctx context.Context, p BlobObjParams, initialState protocol.ClusterState) (exists bool, err error) {
	logger := logging.LoggerFromCtx(ctx)

	err = ensureStorageContainer(ctx, p.StorageName, p.ContainerName)
	if err != nil {
		return
	}

	credential, err := getCredential(ctx)
	if err != nil {
		return
	}

	url := getBlobFileUrl(p.StorageName, p.ContainerName, p.BlobName)
	blobClient, err := blob.NewClient(url, credential, nil)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create blob client")
		return
	}

	exists, err = blobExists(ctx, blobClient)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	if exists {
		logger.Info().Msg("state already exists")
		return
	}

	logger.Info().Msg("state does not exist, creating new state")

	err = WriteState(ctx, p, initialState)
	if err != nil {
		logger.Error().Err(err).Msg("failed to write initial state")
	}
	return
}

func ReadStateOrCreateNew(ctx context.Context, p BlobObjParams, initialState protocol.ClusterState) (state protocol.ClusterState, err error) {
	exists, err := EnsureStateIsCreated(ctx, p, initialState)
	if err != nil {
		return
	}

	if exists {
		return ReadState(ctx, p)
	}
	return initialState, nil
}

func ReadState(ctx context.Context, stateParams BlobObjParams) (state protocol.ClusterState, err error) {
	logger := logging.LoggerFromCtx(ctx)

	stateAsByteArray, err := ReadBlobObject(ctx, stateParams)
	if err != nil {
		return
	}
	err = json.Unmarshal(stateAsByteArray, &state)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	return
}

func WriteBlobObject(ctx context.Context, bl BlobObjParams, state []byte) (err error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := getCredential(ctx)
	if err != nil {
		return
	}

	blobClient, err := azblob.NewClient(getBlobUrl(bl.StorageName), credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	_, err = blobClient.UploadBuffer(ctx, bl.ContainerName, bl.BlobName, state, &azblob.UploadBufferOptions{})

	return

}

func WriteState(ctx context.Context, stateParams BlobObjParams, state protocol.ClusterState) (err error) {
	logger := logging.LoggerFromCtx(ctx)

	stateAsByteArray, err := json.Marshal(state)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	err = WriteBlobObject(ctx, stateParams, stateAsByteArray)
	return
}

func getBlobUrl(storageName string) string {
	return fmt.Sprintf("https://%s.blob.core.windows.net/", storageName)
}

func getBlobFileUrl(storageName, containerName, blobName string) string {
	return fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", storageName, containerName, blobName)
}

func getContainerUrl(storageName, containerName string) string {
	return fmt.Sprintf("https://%s.blob.core.windows.net/%s", storageName, containerName)
}

type ShutdownRequired struct {
	Message string
}

func (e *ShutdownRequired) Error() string {
	return e.Message
}

func AddInstanceToState(ctx context.Context, subscriptionId, resourceGroupName string, stateParams BlobObjParams, newInstance protocol.Vm) (state protocol.ClusterState, err error) {
	logger := logging.LoggerFromCtx(ctx)

	leaseId, err := LockContainer(ctx, stateParams.StorageName, stateParams.ContainerName)
	if err != nil {
		return
	}
	defer UnlockContainer(ctx, stateParams.StorageName, stateParams.ContainerName, leaseId)

	state, err = ReadState(ctx, stateParams)
	if err != nil {
		return
	}

	if len(state.Instances) >= state.InitialSize {
		message := "cluster size is already satisfied"
		err = &ShutdownRequired{
			Message: message,
		}
		logger.Error().Err(err).Send()
	} else if state.Clusterized {
		err = &ShutdownRequired{
			Message: "cluster is already clusterized",
		}
		logger.Error().Err(err).Send()
	} else {
		state.Instances = append(state.Instances, newInstance)
		err = WriteState(ctx, stateParams, state)
	}
	return
}

func GetStateInstancesNames(vms []protocol.Vm) (instanceNames []string) {
	for _, vm := range vms {
		instanceNames = append(instanceNames, vm.Name)
	}
	return
}

func UpdateClusterized(ctx context.Context, subscriptionId, resourceGroupName string, stateParams BlobObjParams) (state protocol.ClusterState, err error) {
	logger := logging.LoggerFromCtx(ctx)

	leaseId, err := LockContainer(ctx, stateParams.StorageName, stateParams.ContainerName)
	if err != nil {
		return
	}
	defer UnlockContainer(ctx, stateParams.StorageName, stateParams.ContainerName, leaseId)

	state, err = ReadState(ctx, stateParams)
	if err != nil {
		return
	}

	state.Instances = []protocol.Vm{}
	state.Clusterized = true

	err = WriteState(ctx, stateParams, state)

	logger.Info().Msg("State updated to 'clusterized'")
	return
}

func CreateStorageAccount(ctx context.Context, subscriptionId, resourceGroupName, obsName, location string) (accessKey string, err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("creating storage account: %s", obsName)

	credential, err := getCredential(ctx)
	if err != nil {
		return
	}

	client, err := armstorage.NewAccountsClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}
	skuName := armstorage.SKUNameStandardZRS
	kind := armstorage.KindStorageV2
	// publicAccessDisabled := armstorage.PublicNetworkAccessDisabled
	_, err = client.BeginCreate(ctx, resourceGroupName, obsName, armstorage.AccountCreateParameters{
		Kind:     &kind,
		Location: &location,
		SKU: &armstorage.SKU{
			Name: &skuName,
		},
		// Properties: &armstorage.AccountPropertiesCreateParameters{
		// 	PublicNetworkAccess: &publicAccessDisabled,
		// },
	}, nil)

	if err != nil {
		if azerr, ok := err.(*azcore.ResponseError); ok {
			if azerr.ErrorCode == "StorageAccountAlreadyExists" {
				logger.Debug().Msgf("storage account %s already exists", obsName)
				err = nil
			} else {
				logger.Error().Msgf("storage creation failed: %s", err)
				return
			}
		} else {
			logger.Error().Msgf("storage creation failed: %s", err)
			return
		}
	}

	for i := 0; i < 10; i++ {
		accessKey, err = getStorageAccountAccessKey(ctx, subscriptionId, resourceGroupName, obsName)

		if err != nil {
			if azerr, ok := err.(*azcore.ResponseError); ok {
				if azerr.ErrorCode == "StorageAccountIsNotProvisioned" {
					logger.Debug().Msgf("new storage account is not ready will retry in 1M")
					time.Sleep(time.Minute)
				} else {
					logger.Error().Err(err).Send()
					return
				}
			} else {
				logger.Error().Err(err).Send()
				return
			}
		} else {
			logger.Debug().Msgf("storage account '%s' is ready for use", obsName)
			break
		}
	}

	return
}

func getStorageAccountAccessKey(ctx context.Context, subscriptionId, resourceGroupName, obsName string) (accessKey string, err error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := getCredential(ctx)
	if err != nil {
		return
	}

	client, err := armstorage.NewAccountsClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}
	keys, err := client.ListKeys(ctx, resourceGroupName, obsName, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}
	accessKey = *keys.Keys[0].Value
	return
}

func CreateContainer(ctx context.Context, storageAccountName, containerName string) (err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("creating obs container %s in storage account %s", containerName, storageAccountName)

	credential, err := getCredential(ctx)
	if err != nil {
		return
	}

	blobClient, err := azblob.NewClient(getBlobUrl(storageAccountName), credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	_, err = blobClient.CreateContainer(ctx, containerName, nil)
	if err != nil {
		if azerr, ok := err.(*azcore.ResponseError); ok {
			if azerr.ErrorCode == "ContainerAlreadyExists" {
				logger.Info().Msgf("obs container %s already exists", containerName)
				err = nil
				return
			}
		}
		logger.Error().Msgf("obs container creation failed: %s", err)
	}
	return
}

func GetKeyVaultValue(ctx context.Context, keyVaultUri, secretName string) (secret string, err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("fetching key vault secret: %s", secretName)

	credential, err := getCredential(ctx)
	if err != nil {
		return
	}

	client, err := azsecrets.NewClient(keyVaultUri, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}
	resp, err := client.GetSecret(ctx, secretName, "", nil)
	if err != nil {
		logger.Info().Err(err).Send()
		return
	}

	secret = *resp.Value

	return
}

func SetKeyVaultValue(ctx context.Context, keyVaultUri, secretName, secretValue string) (err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("setting key vault secret: %s", secretName)

	credential, err := getCredential(ctx)
	if err != nil {
		return
	}

	client, err := azsecrets.NewClient(keyVaultUri, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	params := azsecrets.SetSecretParameters{
		Value: &secretValue,
	}

	_, err = client.SetSecret(ctx, secretName, params, nil)
	if err != nil {
		logger.Error().Err(err).Send()
	}
	return
}

// Gets all network interfaces in a VM scale set (Uniform)
// see https://learn.microsoft.com/en-us/rest/api/virtualnetwork/network-interface-in-vm-ss/list-virtual-machine-scale-set-network-interfaces
func getUniformScaleSetVmsNetworkInterfaces(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string) (networkInterfaces []*armnetwork.Interface, err error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := getCredential(ctx)
	if err != nil {
		return
	}

	client, err := armnetwork.NewInterfacesClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	pager := client.NewListVirtualMachineScaleSetNetworkInterfacesPager(resourceGroupName, vmScaleSetName, nil)

	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			logger.Error().Err(err).Send()
			return nil, err
		}
		networkInterfaces = append(networkInterfaces, nextResult.Value...)
	}
	return
}

func getFlexibleScaleSetVmsNetworkInterfaces(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string, onlyPrimary bool, vms []*VMInfoSummary) (networkInterfaces []*armnetwork.Interface, err error) {
	logger := logging.LoggerFromCtx(ctx)

	if vms == nil {
		azVms, err := getFlexibleScaleSetVms(ctx, subscriptionId, resourceGroupName, vmScaleSetName, nil)
		if err != nil {
			err = fmt.Errorf("cannot get scale set vms: %v", err)
			logger.Error().Err(err).Send()
			return nil, err
		}
		vms = VMsToVmInfoSummary(azVms)
	}

	nicIds := make([]string, 0)
	for _, vm := range vms {
		if vm.NetworkProfile != nil && vm.NetworkProfile.NetworkInterfaces != nil {
			for _, nic := range vm.NetworkProfile.NetworkInterfaces {
				if onlyPrimary && nic.Properties.Primary == nil || !*nic.Properties.Primary {
					continue
				}
				nicIds = append(nicIds, *nic.ID)
			}
		}
	}

	credential, err := getCredential(ctx)
	if err != nil {
		logger.Error().Err(err).Send()
		return nil, err
	}

	client, err := armnetwork.NewInterfacesClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return nil, err
	}

	for _, nicId := range nicIds {
		nicIdParts := strings.Split(nicId, "/")
		nicName := nicIdParts[len(nicIdParts)-1]
		resp, err := client.Get(ctx, resourceGroupName, nicName, nil)
		if err != nil {
			logger.Error().Err(err).Send()
			return nil, err
		}
		networkInterfaces = append(networkInterfaces, &resp.Interface)
	}
	return
}

func GetScaleSetVmsNetworkPrimaryNICs(ctx context.Context, vmssParams *ScaleSetParams, forVms []*VMInfoSummary) (networkInterfaces []*armnetwork.Interface, err error) {
	var nics []*armnetwork.Interface
	if vmssParams.Flexible {
		nics, err = getFlexibleScaleSetVmsNetworkInterfaces(ctx, vmssParams.SubscriptionId, vmssParams.ResourceGroupName, vmssParams.ScaleSetName, true, forVms)
	} else {
		nics, err = getUniformScaleSetVmsNetworkInterfaces(ctx, vmssParams.SubscriptionId, vmssParams.ResourceGroupName, vmssParams.ScaleSetName)
	}
	if err != nil {
		err = fmt.Errorf("cannot get scale set vms network interfaces: %v", err)
		return nil, err
	}

	for _, ni := range nics {
		if ni.Properties == nil || ni.Properties.VirtualMachine == nil || len(ni.Properties.IPConfigurations) < 1 {
			continue
		}
		if ni.Properties.Primary == nil || !*ni.Properties.Primary {
			// get only primary NICs
			continue
		}
		networkInterfaces = append(networkInterfaces, ni)
	}
	return
}

func GetScaleSetSecondaryIps(ctx context.Context, vmssParams *ScaleSetParams) (secondaryIps []string, err error) {
	var nics []*armnetwork.Interface
	if vmssParams.Flexible {
		nics, err = getFlexibleScaleSetVmsNetworkInterfaces(ctx, vmssParams.SubscriptionId, vmssParams.ResourceGroupName, vmssParams.ScaleSetName, false, nil)
	} else {
		nics, err = getUniformScaleSetVmsNetworkInterfaces(ctx, vmssParams.SubscriptionId, vmssParams.ResourceGroupName, vmssParams.ScaleSetName)
	}
	if err != nil {
		err = fmt.Errorf("cannot get scale set vms network interfaces: %v", err)
		return nil, err
	}

	for _, nic := range nics {
		if nic.Properties == nil || nic.Properties.VirtualMachine == nil || len(nic.Properties.IPConfigurations) < 1 {
			continue
		}
		for _, ipConfig := range nic.Properties.IPConfigurations {
			isPrimary := ipConfig.Properties.Primary != nil && *ipConfig.Properties.Primary
			if !isPrimary && ipConfig.Properties.PrivateIPAddress != nil {
				secondaryIps = append(secondaryIps, *ipConfig.Properties.PrivateIPAddress)
			}
		}
	}
	return
}

func GetPublicIp(ctx context.Context, vmssParams *ScaleSetParams, prefix, clusterName, instanceIndex string) (publicIp string, err error) {
	logger := logging.LoggerFromCtx(ctx)

	if vmssParams.Flexible {
		err = errors.New("ignore getting public ip for flexible scale set VM")
		return
	}

	credential, err := getCredential(ctx)
	if err != nil {
		return
	}

	client, err := armnetwork.NewPublicIPAddressesClient(vmssParams.SubscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}
	interfaceName := fmt.Sprintf("%s-%s-backend-nic-0", prefix, clusterName)
	pager := client.NewListVirtualMachineScaleSetVMPublicIPAddressesPager(vmssParams.ResourceGroupName, vmssParams.ScaleSetName, instanceIndex, interfaceName, "ipconfig0", nil)

	for pager.More() {
		nextResult, err1 := pager.NextPage(ctx)
		if err1 != nil {
			logger.Error().Err(err1).Send()
			return "", err1
		}
		if len(nextResult.Value) > 0 {
			publicIp = *nextResult.Value[0].Properties.IPAddress
			return
		}
	}

	return
}

func GetVmsPrivateIps(ctx context.Context, vmssParams *ScaleSetParams) (vmsPrivateIps map[string]string, err error) {
	//returns compute_name to private ip map

	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msg("fetching scale set vms private ips")

	networkInterfaces, err := GetScaleSetVmsNetworkPrimaryNICs(ctx, vmssParams, nil)
	if err != nil {
		return
	}

	vmsPrivateIps = make(map[string]string)
	for _, networkInterface := range networkInterfaces {
		vmNameParts := strings.Split(*networkInterface.Properties.VirtualMachine.ID, "/")
		vmNamePartsLen := len(vmNameParts)
		vmName := fmt.Sprintf("%s_%s", vmNameParts[vmNamePartsLen-3], vmNameParts[vmNamePartsLen-1])
		if _, ok := vmsPrivateIps[vmName]; !ok {
			vmsPrivateIps[vmName] = *networkInterface.Properties.IPConfigurations[0].Properties.PrivateIPAddress
		}
	}
	return
}

func ScaleUp(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string, newSize int64) (err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msg("updating scale set vms num")

	credential, err := getCredential(ctx)
	if err != nil {
		return
	}

	client, err := armcompute.NewVirtualMachineScaleSetsClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	response, err := client.Get(ctx, resourceGroupName, vmScaleSetName, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	scaleSetCapacity := *response.SKU.Capacity
	if scaleSetCapacity >= newSize {
		logger.Info().Msgf(
			"scale set %s capacity:%d desired capacity:%d, skipping scale up", vmScaleSetName, scaleSetCapacity, newSize)
		return
	}

	_, err = client.BeginUpdate(ctx, resourceGroupName, vmScaleSetName, armcompute.VirtualMachineScaleSetUpdate{
		SKU: &armcompute.SKU{
			Capacity: &newSize,
		},
	}, nil)
	if err != nil {
		logger.Error().Err(err).Send()
	}
	return
}

func GetRoleDefinitionByRoleName(ctx context.Context, roleName, scope string) (*armauthorization.RoleDefinition, error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := getCredential(ctx)
	if err != nil {
		return nil, err
	}

	client, err := armauthorization.NewRoleDefinitionsClient(credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return nil, err
	}

	var results []*armauthorization.RoleDefinition
	filter := fmt.Sprintf("roleName eq '%s'", roleName)

	pager := client.NewListPager("/", &armauthorization.RoleDefinitionsClientListOptions{Filter: &filter})
	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			logger.Error().Err(err).Send()
			return nil, err
		}
		results = append(results, nextResult.Value...)
	}

	// filter the needed role out of all built-in ones
	var roleDefs []*armauthorization.RoleDefinition
	for _, res := range results {
		if *res.Properties.RoleName == roleName {
			roleDefs = append(roleDefs, res)
		}
	}

	if len(roleDefs) < 1 {
		err := fmt.Errorf("cannot find az role definition with name '%s'", roleName)
		logger.Error().Err(err).Send()
		return nil, err
	}
	if len(roleDefs) > 1 {
		err := fmt.Errorf("found several az role definitions with name '%s', check the name", roleName)
		logger.Error().Err(err).Send()
		return nil, err
	}
	return roleDefs[0], nil
}

func AssignStorageBlobDataContributorRoleToScaleSet(
	ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName, storageAccountName, containerName string,
) (*armauthorization.RoleAssignment, error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := getCredential(ctx)
	if err != nil {
		return nil, err
	}

	client, err := armauthorization.NewRoleAssignmentsClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return nil, err
	}

	scaleSet, err := getScaleSet(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		logger.Error().Err(err).Send()
		return nil, err
	}

	scope := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s/blobServices/default/containers/%s",
		subscriptionId,
		resourceGroupName,
		storageAccountName,
		containerName,
	)

	roleDefinition, err := GetRoleDefinitionByRoleName(ctx, "Storage Blob Data Contributor", scope)
	if err != nil {
		err = fmt.Errorf("cannot get the role definition: %v", err)
		logger.Error().Err(err).Send()
		return nil, err
	}

	var principalId *string
	if scaleSet.Identity.PrincipalID != nil {
		principalId = scaleSet.Identity.PrincipalID
	} else if len(scaleSet.Identity.UserAssignedIdentities) > 0 {
		// try user-assigned identity
		identitites := scaleSet.Identity.UserAssignedIdentities
		for _, identity := range identitites {
			principalId = identity.PrincipalID
			break
		}
	} else {
		err = errors.New("cannot find principal id for the scale set")
		logger.Error().Err(err).Send()
		return nil, err
	}

	// see https://learn.microsoft.com/en-us/rest/api/authorization/role-assignments/create
	res, err := client.Create(
		ctx,
		scope,
		uuid.New().String(), // az docs say it should be GUID
		armauthorization.RoleAssignmentCreateParameters{
			Properties: &armauthorization.RoleAssignmentProperties{
				RoleDefinitionID: roleDefinition.ID,
				PrincipalID:      principalId,
			},
		},
		nil,
	)
	if err != nil {
		var responseErr *azcore.ResponseError
		if errors.As(err, &responseErr) && responseErr.ErrorCode == "RoleAssignmentExists" {
			logger.Info().Msg("role assignment already exists")
			return nil, nil
		}

		err = fmt.Errorf("cannot create the role assignment: %v", err)
		logger.Error().Err(err).Send()
		return nil, err
	}

	return &res.RoleAssignment, nil
}

// Gets scale set
// see https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-scale-sets/get
func getScaleSet(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string) (*armcompute.VirtualMachineScaleSet, error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("Getting scale set %s info", vmScaleSetName)

	credential, err := getCredential(ctx)
	if err != nil {
		return nil, err
	}

	client, err := armcompute.NewVirtualMachineScaleSetsClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return nil, err
	}

	scaleSet, err := client.Get(ctx, resourceGroupName, vmScaleSetName, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return nil, err
	}
	return &scaleSet.VirtualMachineScaleSet, nil
}

func GetScaleSetOrNil(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string) (*armcompute.VirtualMachineScaleSet, error) {
	logger := logging.LoggerFromCtx(ctx)
	scaleSet, err := getScaleSet(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		var responseErr *azcore.ResponseError
		if errors.As(err, &responseErr) && (responseErr.ErrorCode == "ResourceNotFound" || responseErr.ErrorCode == "NotFound") {
			// scale set is not found
			return nil, nil
		}
		logger.Error().Err(err).Send()
		return nil, err
	}
	return scaleSet, nil
}

func GetScaleSetInstances(ctx context.Context, vmssParams *ScaleSetParams) (vms []*VMInfoSummary, err error) {
	if vmssParams.Flexible {
		vms, err = GetFlexibleScaleSetInstances(ctx, vmssParams.SubscriptionId, vmssParams.ResourceGroupName, vmssParams.ScaleSetName, nil)
	} else {
		vms, err = GetUniformScaleSetInstances(ctx, vmssParams.SubscriptionId, vmssParams.ResourceGroupName, vmssParams.ScaleSetName, nil)
	}
	return
}

// Gets a list of all VMs in a scale set
// see https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-scale-set-vms/list
func GetUniformScaleSetInstances(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string, expand *armcompute.ExpandTypeForListVMs) (vms []*VMInfoSummary, err error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := getCredential(ctx)
	if err != nil {
		return
	}

	client, err := armcompute.NewVirtualMachineScaleSetVMsClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	var expandStr *string
	if expand != nil {
		expandStr = (*string)(expand)
	}

	pager := client.NewListPager(
		resourceGroupName, vmScaleSetName, &armcompute.VirtualMachineScaleSetVMsClientListOptions{
			Expand: expandStr,
		})

	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			err = fmt.Errorf("failed to advance page getting images list: %v", err)
			logger.Error().Err(err).Send()
			return nil, err
		}
		vmsSummary := UniformVmssVMsToVmInfoSummary(nextResult.Value)
		vms = append(vms, vmsSummary...)
	}
	return
}

func GetFlexibleScaleSetInstances(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string, expand *armcompute.ExpandTypeForListVMs) (vms []*VMInfoSummary, err error) {
	logger := logging.LoggerFromCtx(ctx)

	azVMs, err := getFlexibleScaleSetVms(ctx, subscriptionId, resourceGroupName, vmScaleSetName, expand)
	if err != nil {
		return
	}
	vms = VMsToVmInfoSummary(azVMs)

	vmNamesToVms := make(map[string]*VMInfoSummary, len(vms))
	for _, vm := range vms {
		vmNamesToVms[vm.Name] = vm
	}

	// fetch vms ids along with protection policy info (can only be fethced via VMSS VMs client)
	credential, err := getCredential(ctx)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	client, err := armcompute.NewVirtualMachineScaleSetVMsClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	vmssVmPager := client.NewListPager(resourceGroupName, vmScaleSetName, nil)
	for vmssVmPager.More() {
		nextResult, err := vmssVmPager.NextPage(ctx)
		if err != nil {
			err = fmt.Errorf("cannot get scale set %s vms: %v", vmScaleSetName, err)
			logger.Error().Err(err).Send()
			return nil, err
		}
		for _, vmssVm := range nextResult.Value {
			vmName := *vmssVm.Name
			vm, ok := vmNamesToVms[vmName]
			if ok {
				if vmssVm.Properties != nil && vmssVm.Properties.ProtectionPolicy != nil {
					vm.ProtectionPolicy = vmssVm.Properties.ProtectionPolicy
				}
			} else {
				err = fmt.Errorf("cannot find vm %s from the flexible scale set %s", vmName, vmScaleSetName)
				logger.Error().Err(err).Send()
				return nil, err
			}
		}
	}
	return
}

func getFlexibleScaleSetVms(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string, expand *armcompute.ExpandTypeForListVMs) (vms []*armcompute.VirtualMachine, err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msg("fetching flexible scale set vms")

	credential, err := getCredential(ctx)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	client, err := armcompute.NewVirtualMachinesClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	filter := fmt.Sprintf("'virtualMachineScaleSet/id' eq '/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachineScaleSets/%s'", subscriptionId, resourceGroupName, vmScaleSetName)
	options := &armcompute.VirtualMachinesClientListOptions{
		Filter: &filter,
		Expand: expand,
	}

	pager := client.NewListPager(resourceGroupName, options)
	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			err = fmt.Errorf("cannot get flexible scale set %s vms: %v", vmScaleSetName, err)
			logger.Error().Err(err).Send()
			return nil, err
		}
		vms = append(vms, nextResult.Value...)
	}
	return
}

func GetScaleSetVmId(resourceId string) string {
	vmNameParts := strings.Split(resourceId, "/")
	vmNamePartsLen := len(vmNameParts)
	vmId := vmNameParts[vmNamePartsLen-1]
	return vmId
}

func GetScaleSetInstancesInfoFromVms(ctx context.Context, vmssParams *ScaleSetParams, vms []*VMInfoSummary) (instances []protocol.HgInstance, err error) {
	netInterfaces, err := GetScaleSetVmsNetworkPrimaryNICs(ctx, vmssParams, vms)
	if err != nil {
		return
	}
	instanceIdPrivateIp := map[string]string{}

	for _, ni := range netInterfaces {
		id := GetScaleSetVmId(*ni.Properties.VirtualMachine.ID)
		privateIp := *ni.Properties.IPConfigurations[0].Properties.PrivateIPAddress
		instanceIdPrivateIp[id] = privateIp
	}

	for _, vm := range vms {
		id := GetScaleSetVmId(vm.ID)
		// get private ip if exists
		var privateIp string
		if val, ok := instanceIdPrivateIp[id]; ok {
			privateIp = val
		}
		instanceInfo := protocol.HgInstance{
			Id:        id,
			PrivateIp: privateIp,
		}
		instances = append(instances, instanceInfo)
	}
	return
}

func GetScaleSetInstancesInfo(ctx context.Context, vmssParams *ScaleSetParams) (instances []protocol.HgInstance, err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("Getting scale set instances %s info", vmssParams.ScaleSetName)

	vms, err := GetScaleSetInstances(ctx, vmssParams)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	instances, err = GetScaleSetInstancesInfoFromVms(ctx, vmssParams, vms)
	if err != nil {
		logger.Error().Err(err).Send()
	}
	return
}

func GetScaleSetVmIndex(vmName string, flexible bool) string {
	if flexible {
		// In flexible scale sets, the vm 'name' is same as 'instanceId'
		return vmName
	}
	instanceNameParts := strings.Split(vmName, "_")
	return instanceNameParts[len(instanceNameParts)-1]
}

// NOTE: works both for Uniform and Flexible scale sets
func SetDeletionProtection(ctx context.Context, vmssParams *ScaleSetParams, instanceId string, protect bool) (err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("Setting deletion protection: %t on instanceId %s", protect, instanceId)

	credential, err := getCredential(ctx)
	if err != nil {
		return
	}

	client, err := armcompute.NewVirtualMachineScaleSetVMsClient(vmssParams.SubscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	_, err = client.BeginUpdate(
		ctx,
		vmssParams.ResourceGroupName,
		vmssParams.ScaleSetName,
		instanceId,
		armcompute.VirtualMachineScaleSetVM{
			Properties: &armcompute.VirtualMachineScaleSetVMProperties{
				ProtectionPolicy: &armcompute.VirtualMachineScaleSetVMProtectionPolicy{
					ProtectFromScaleSetActions: &protect,
				},
			},
		},
		nil)

	return
}

func RetrySetDeletionProtectionAndReport(
	ctx context.Context, vmssParams *ScaleSetParams, stateParams BlobObjParams, instanceId, hostName string,
	maxAttempts int, sleepInterval time.Duration,
) (err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("Setting deletion protection on %s", hostName)
	counter := 0
	for {
		err = SetDeletionProtection(ctx, vmssParams, instanceId, true)
		if err == nil {
			msg := "Deletion protection was set successfully"
			logger.Info().Msg(msg)
			ReportMsg(ctx, hostName, stateParams, "progress", msg)
			break
		}

		if protectionErr, ok := err.(*azcore.ResponseError); ok && protectionErr.ErrorCode == "AuthorizationFailed" {
			counter++
			// deletion protection invoked by terminate function
			if maxAttempts == 0 {
				msg := "Deletion protection set authorization isn't ready, will retry on next scale down workflow"
				ReportMsg(ctx, hostName, stateParams, "debug", msg)
				return
			}

			if counter > maxAttempts {
				break
			}
			msg := fmt.Sprintf("Deletion protection set authorization isn't ready, going to sleep for %s", sleepInterval)
			logger.Info().Msg(msg)
			ReportMsg(ctx, hostName, stateParams, "debug", msg)
			time.Sleep(sleepInterval)
		} else {
			break
		}
	}
	if err != nil {
		logger.Error().Err(err).Send()
		ReportMsg(ctx, hostName, stateParams, "error", err.Error())
	}
	return
}

func ReportMsg(ctx context.Context, hostName string, stateParams BlobObjParams, reportType, message string) {
	reportObj := protocol.Report{Type: reportType, Hostname: hostName, Message: message}
	_ = UpdateStateReporting(ctx, stateParams, reportObj)
}

func GetWekaAdminPassword(ctx context.Context, keyVaultUri string) (password string, err error) {
	return GetKeyVaultValue(ctx, keyVaultUri, WekaAdminPasswordKey)
}

func GetWekaDeploymentPassword(ctx context.Context, keyVaultUri string) (password string, err error) {
	return GetKeyVaultValue(ctx, keyVaultUri, WekaDeploymentPasswordKey)
}

// Get Weka deployment password if exists, otherwise get admin password
func GetWekaClusterCredentials(ctx context.Context, keyVaultUri string) (protocol.ClusterCreds, error) {
	usename := WekaDeploymentUsername
	password, err := GetWekaDeploymentPassword(ctx, keyVaultUri)

	var responseErr *azcore.ResponseError
	if err != nil && errors.As(err, &responseErr) && responseErr.ErrorCode == "SecretNotFound" || err == nil && password == "" {
		usename = WekaAdminUsername
		password, err = GetWekaAdminPassword(ctx, keyVaultUri)
	}

	credentials := protocol.ClusterCreds{
		Username: usename,
		Password: password,
	}
	return credentials, err
}

func SetWekaDeploymentPassword(ctx context.Context, keyVaultUri, password string) (err error) {
	return SetKeyVaultValue(ctx, keyVaultUri, WekaDeploymentPasswordKey, password)
}

func SetWekaAdminPassword(ctx context.Context, keyVaultUri, password string) (err error) {
	return SetKeyVaultValue(ctx, keyVaultUri, WekaAdminPasswordKey, password)
}

func GetVmScaleSetName(prefix, clusterName string) string {
	return fmt.Sprintf("%s-%s-vmss", prefix, clusterName)
}

type InstanceIdsSet map[string]types.Nilt

func GetInstanceIpsSet(scaleResponse protocol.ScaleResponse) InstanceIdsSet {
	instanceIpsSet := make(InstanceIdsSet)
	for _, instance := range scaleResponse.Hosts {
		instanceIpsSet[instance.PrivateIp] = types.Nilv
	}
	return instanceIpsSet
}

func FilterSpecificScaleSetInstances(ctx context.Context, allVms []*VMInfoSummary, instanceIds []string) (vms []*VMInfoSummary, err error) {
	instanceIdsSet := make(InstanceIdsSet)
	for _, instanceId := range instanceIds {
		instanceIdsSet[instanceId] = types.Nilv
	}

	for _, vm := range allVms {
		if _, ok := instanceIdsSet[GetScaleSetVmId(vm.ID)]; ok {
			vms = append(vms, vm)
		}
	}

	return
}

func TerminateScaleSetInstances(ctx context.Context, vmssParams *ScaleSetParams, terminateInstanceIds []string) (terminatedInstances []string, errs []error) {
	logger := logging.LoggerFromCtx(ctx)

	if len(terminateInstanceIds) == 0 {
		return
	}
	for _, instanceId := range terminateInstanceIds {
		err := SetDeletionProtection(ctx, vmssParams, instanceId, false)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		logger.Info().Msgf("Deleting instanceId %s", instanceId)
		if vmssParams.Flexible {
			err = deleteFlexibleScaleSetVM(ctx, vmssParams.SubscriptionId, vmssParams.ResourceGroupName, vmssParams.ScaleSetName, instanceId)
		} else {
			err = deleteUniformScaleSetVM(ctx, vmssParams.SubscriptionId, vmssParams.ResourceGroupName, vmssParams.ScaleSetName, instanceId)
		}
		if err != nil {
			logger.Error().Err(err).Send()
			errs = append(errs, err)
			continue
		}
		terminatedInstances = append(terminatedInstances, instanceId)
	}

	return
}

func deleteFlexibleScaleSetVM(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName, instanceId string) (err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("Deleting instanceId %s from Flexible VMSS %s", instanceId, vmScaleSetName)

	credential, err := getCredential(ctx)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	client, err := armcompute.NewVirtualMachinesClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	force := true
	_, err = client.BeginDelete(ctx, resourceGroupName, instanceId, &armcompute.VirtualMachinesClientBeginDeleteOptions{
		ForceDeletion: &force,
	})
	if err != nil {
		logger.Error().Err(err).Send()
	}
	return
}

func deleteUniformScaleSetVM(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName, instanceId string) (err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("Deleting instanceId %s from Uniform VMSS %s", instanceId, vmScaleSetName)

	credential, err := getCredential(ctx)
	if err != nil {
		return
	}

	client, err := armcompute.NewVirtualMachineScaleSetVMsClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	force := true
	_, err = client.BeginDelete(
		ctx,
		resourceGroupName,
		vmScaleSetName,
		instanceId,
		&armcompute.VirtualMachineScaleSetVMsClientBeginDeleteOptions{
			ForceDeletion: &force,
		},
	)
	if err != nil {
		logger.Error().Err(err).Send()
	}
	return
}

func UpdateStateReporting(ctx context.Context, stateParams BlobObjParams, report protocol.Report) (err error) {
	leaseId, err := LockContainer(ctx, stateParams.StorageName, stateParams.ContainerName)
	if err != nil {
		return
	}
	defer UnlockContainer(ctx, stateParams.StorageName, stateParams.ContainerName, leaseId)

	return UpdateStateReportingWithoutLocking(ctx, stateParams, report)
}

func AddClusterUpdate(ctx context.Context, stateParams BlobObjParams, update protocol.Update) (err error) {
	leaseId, err := LockContainer(ctx, stateParams.StorageName, stateParams.ContainerName)
	if err != nil {
		return
	}
	defer UnlockContainer(ctx, stateParams.StorageName, stateParams.ContainerName, leaseId)

	state, err := ReadState(ctx, stateParams)
	if err != nil {
		return
	}

	reportLib.AddClusterUpdate(update, &state)

	err = WriteState(ctx, stateParams, state)
	if err != nil {
		err = fmt.Errorf("failed addind cluster update to state")
		return
	}
	return
}

func UpdateStateReportingWithoutLocking(ctx context.Context, stateParams BlobObjParams, report protocol.Report) (err error) {
	state, err := ReadState(ctx, stateParams)
	if err != nil {
		return
	}
	err = reportLib.UpdateReport(report, &state)
	if err != nil {
		err = fmt.Errorf("failed updating state report")
		return
	}
	err = WriteState(ctx, stateParams, state)
	if err != nil {
		err = fmt.Errorf("failed updating state report")
		return
	}
	return
}

func GetInstancePowerState(instance *VMInfoSummary) (powerState string) {
	prefix := "PowerState/"
	for _, status := range instance.InstanceViewStatuses {
		if strings.HasPrefix(*status.Code, prefix) {
			powerState = strings.TrimPrefix(*status.Code, prefix)
			return
		}
	}
	return
}

func GetInstanceProvisioningState(instance *VMInfoSummary) (provisioningState string) {
	provisioningState = "unknown"
	if instance.ProvisioningState != nil {
		provisioningState = *instance.ProvisioningState
	}
	return strings.ToLower(provisioningState)
}

func GetUnhealthyInstancesToTerminate(ctx context.Context, scaleSetVms []*VMInfoSummary) (toTerminate []string) {
	logger := logging.LoggerFromCtx(ctx)

	for _, vm := range scaleSetVms {
		if vm.VMHealth == nil {
			continue
		}
		healthStatus := *vm.VMHealth.Status.Code
		if healthStatus == "HealthState/unhealthy" {
			instancePowerState := GetInstancePowerState(vm)
			instanceProvisioningState := GetInstanceProvisioningState(vm)
			logger.Debug().Msgf("instance power state: %s, provisioning state: %s", instancePowerState, instanceProvisioningState)
			if instancePowerState == "stopped" || instanceProvisioningState == "failed" {
				toTerminate = append(toTerminate, GetScaleSetVmId(vm.ID))
			}
		}
	}

	logger.Info().Msgf("found %d unhealthy stopped instances to terminate: %s", len(toTerminate), toTerminate)
	return
}

func GetScaleSetVmsExpandedView(ctx context.Context, p *ScaleSetParams) ([]*VMInfoSummary, error) {
	expand := armcompute.ExpandTypeForListVMsInstanceView
	if p.Flexible {
		return GetFlexibleScaleSetInstances(ctx, p.SubscriptionId, p.ResourceGroupName, p.ScaleSetName, &expand)
	} else {
		return GetUniformScaleSetInstances(ctx, p.SubscriptionId, p.ResourceGroupName, p.ScaleSetName, &expand)
	}
}

func GetAzureInstanceNameCmd() string {
	return "curl -s -H Metadata:true --noproxy * http://169.254.169.254/metadata/instance?api-version=2021-02-01 | jq '.compute.name' | cut -c2- | rev | cut -c2- | rev"
}

func ReadVmssConfig(ctx context.Context, vmssConfigStr string) (vmssConfig VMSSConfig, err error) {
	logger := logging.LoggerFromCtx(ctx)

	if vmssConfigStr == "" {
		err = fmt.Errorf("vmss config is not set")
		logger.Error().Err(err).Msg("cannot read vmss config")
		return
	}

	asByteArray := []byte(vmssConfigStr)
	err = json.Unmarshal(asByteArray, &vmssConfig)
	if err != nil {
		logger.Error().Err(err).Msg("cannot unmarshal vmss config")
		return
	}

	// calculate hash of the config (used to identify vmss config changes)
	hash := sha256.Sum256(asByteArray)
	hashStr := fmt.Sprintf("%x", hash)

	// take first 16 characters of the hash
	vmssConfig.ConfigHash = hashStr[:16]
	return
}

func GetVmssConfig(ctx context.Context, resourceGroupName string, scaleSet *armcompute.VirtualMachineScaleSet) *VMSSConfig {
	var identityIds []string
	if scaleSet.Identity != nil && scaleSet.Identity.UserAssignedIdentities != nil {
		for identityId := range scaleSet.Identity.UserAssignedIdentities {
			identityIds = append(identityIds, identityId)
		}
	}

	var identityType string
	if scaleSet.Identity != nil {
		identityType = string(*scaleSet.Identity.Type)
	}

	var sshPublicKey string
	if scaleSet.Properties.VirtualMachineProfile.OSProfile.LinuxConfiguration.SSH.PublicKeys != nil {
		sshPublicKey = *scaleSet.Properties.VirtualMachineProfile.OSProfile.LinuxConfiguration.SSH.PublicKeys[0].KeyData
	}

	var primaryNic *PrimaryNIC
	var secondaryNics *SecondaryNICs

	for _, nic := range scaleSet.Properties.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations {
		ipConfigs := make([]IPConfiguration, len(nic.Properties.IPConfigurations))

		for i, ipConfig := range nic.Properties.IPConfigurations {
			loadBalancerBackendAddressPoolIDs := make([]string, len(ipConfig.Properties.LoadBalancerBackendAddressPools))
			for j, loadBalancerBackendAddressPool := range ipConfig.Properties.LoadBalancerBackendAddressPools {
				loadBalancerBackendAddressPoolIDs[j] = *loadBalancerBackendAddressPool.ID
			}
			ipConfigs[i] = IPConfiguration{
				LoadBalancerBackendAddressPoolIDs: loadBalancerBackendAddressPoolIDs,
				SubnetID:                          *ipConfig.Properties.Subnet.ID,
				Primary:                           *ipConfig.Properties.Primary,
			}
			if ipConfig.Properties.PublicIPAddressConfiguration != nil {
				ipConfigs[i].PublicIPAddress = &PublicIPAddress{
					Assign:          true,
					DomainNameLabel: *ipConfig.Properties.PublicIPAddressConfiguration.Properties.DNSSettings.DomainNameLabel,
					Name:            *ipConfig.Properties.PublicIPAddressConfiguration.Name,
				}
			}
		}

		if nic.Properties.Primary != nil && *nic.Properties.Primary {
			primaryNic = &PrimaryNIC{
				EnableAcceleratedNetworking: *nic.Properties.EnableAcceleratedNetworking,
				Name:                        *nic.Name,
				NetworkSecurityGroupID:      *nic.Properties.NetworkSecurityGroup.ID,
				IPConfigurations:            ipConfigs,
			}
		} else if secondaryNics == nil {
			nicNameParts := strings.Split(*nic.Name, "-")
			// remove the last part of the nic name which is the index
			namePrefix := strings.Join(nicNameParts[:len(nicNameParts)-1], "-")

			secondaryNics = &SecondaryNICs{
				EnableAcceleratedNetworking: *nic.Properties.EnableAcceleratedNetworking,
				NamePrefix:                  namePrefix,
				IPConfigurations:            ipConfigs,
				NetworkSecurityGroupID:      *nic.Properties.NetworkSecurityGroup.ID,
				Number:                      len(scaleSet.Properties.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations) - 1,
			}
		}
		if primaryNic != nil && secondaryNics != nil {
			break
		}
	}

	var customData string
	if scaleSet.Properties.VirtualMachineProfile.OSProfile.CustomData != nil {
		customData = *scaleSet.Properties.VirtualMachineProfile.OSProfile.CustomData
	}

	var ppg *string
	if scaleSet.Properties.ProximityPlacementGroup != nil {
		ppg = scaleSet.Properties.ProximityPlacementGroup.ID
		upperCaseRg := strings.ToUpper(resourceGroupName)
		val := strings.Replace(*ppg, upperCaseRg, resourceGroupName, 1)
		ppg = &val
	}

	var sourceImageID string
	if scaleSet.Properties.VirtualMachineProfile.StorageProfile.ImageReference.CommunityGalleryImageID != nil {
		sourceImageID = *scaleSet.Properties.VirtualMachineProfile.StorageProfile.ImageReference.CommunityGalleryImageID
	} else {
		sourceImageID = *scaleSet.Properties.VirtualMachineProfile.StorageProfile.ImageReference.ID
	}

	tags := PtrMapToStrMap(scaleSet.Tags)
	configHash := ""
	if val, ok := tags["config_hash"]; ok {
		configHash = val
	}

	var healthProbeId string
	if scaleSet.Properties.VirtualMachineProfile.NetworkProfile.HealthProbe != nil {
		healthProbeId = *scaleSet.Properties.VirtualMachineProfile.NetworkProfile.HealthProbe.ID
	}

	vmssConfig := &VMSSConfig{
		Name:              *scaleSet.Name,
		Location:          *scaleSet.Location,
		Zones:             PtrArrToStrArray(scaleSet.Zones),
		ResourceGroupName: resourceGroupName,
		SKU:               *scaleSet.SKU.Name,
		SourceImageID:     sourceImageID,
		Tags:              tags,

		UpgradeMode:          string(*scaleSet.Properties.UpgradePolicy.Mode),
		OrchestrationMode:    string(*scaleSet.Properties.OrchestrationMode),
		HealthProbeID:        healthProbeId,
		Overprovision:        *scaleSet.Properties.Overprovision,
		SinglePlacementGroup: *scaleSet.Properties.SinglePlacementGroup,

		Identity: Identity{
			IdentityIDs: identityIds,
			Type:        identityType,
		},
		AdminUsername:      *scaleSet.Properties.VirtualMachineProfile.OSProfile.AdminUsername,
		SshPublicKey:       sshPublicKey,
		ComputerNamePrefix: *scaleSet.Properties.VirtualMachineProfile.OSProfile.ComputerNamePrefix,
		CustomData:         customData,

		DisablePasswordAuthentication: *scaleSet.Properties.VirtualMachineProfile.OSProfile.LinuxConfiguration.DisablePasswordAuthentication,
		ProximityPlacementGroupID:     ppg,

		OSDisk: OSDisk{
			Caching:            string(*scaleSet.Properties.VirtualMachineProfile.StorageProfile.OSDisk.Caching),
			StorageAccountType: string(*scaleSet.Properties.VirtualMachineProfile.StorageProfile.OSDisk.ManagedDisk.StorageAccountType),
			SizeGB:             scaleSet.Properties.VirtualMachineProfile.StorageProfile.OSDisk.DiskSizeGB,
		},
		DataDisk: DataDisk{
			Caching:            string(*scaleSet.Properties.VirtualMachineProfile.StorageProfile.DataDisks[0].Caching),
			CreateOption:       string(*scaleSet.Properties.VirtualMachineProfile.StorageProfile.DataDisks[0].CreateOption),
			DiskSizeGB:         *scaleSet.Properties.VirtualMachineProfile.StorageProfile.DataDisks[0].DiskSizeGB,
			Lun:                *scaleSet.Properties.VirtualMachineProfile.StorageProfile.DataDisks[0].Lun,
			StorageAccountType: string(*scaleSet.Properties.VirtualMachineProfile.StorageProfile.DataDisks[0].ManagedDisk.StorageAccountType),
		},
		PrimaryNIC:    *primaryNic,
		SecondaryNICs: secondaryNics,
		ConfigHash:    configHash,
	}
	return vmssConfig
}

func CreateOrUpdateVmss(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName, configHash string, config VMSSConfig, vmssSize int) (id *string, err error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := getCredential(ctx)
	if err != nil {
		return
	}

	client, err := armcompute.NewVirtualMachineScaleSetsClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	config.Tags["config_hash"] = configHash
	config.Tags["config_applied_at"] = time.Now().Format(time.RFC3339)
	size := int64(vmssSize)
	forceDeletion := false
	sshKeyPath := fmt.Sprintf("/home/%s/.ssh/authorized_keys", config.AdminUsername)

	tags := StrMapToPtrMap(config.Tags)
	zones := StrArrToPtrArray(config.Zones)

	identities := make(map[string]*armcompute.UserAssignedIdentitiesValue)
	for _, identityID := range config.Identity.IdentityIDs {
		identities[identityID] = &armcompute.UserAssignedIdentitiesValue{}
	}

	var osDiskSizeGb *int32
	if config.OSDisk.SizeGB != nil {
		osDiskSizeGb = config.OSDisk.SizeGB
	}

	identityType, err := ToEnumStrValue[armcompute.ResourceIdentityType](config.Identity.Type, armcompute.PossibleResourceIdentityTypeValues())
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	upgradeMode, err := ToEnumStrValue[armcompute.UpgradeMode](config.UpgradeMode, armcompute.PossibleUpgradeModeValues())
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	orchestrationMode, err := ToEnumStrValue[armcompute.OrchestrationMode](config.OrchestrationMode, armcompute.PossibleOrchestrationModeValues())
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	osDiskCaching, err := ToEnumStrValue[armcompute.CachingTypes](config.OSDisk.Caching, armcompute.PossibleCachingTypesValues())
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	osDiskCreateOption := armcompute.DiskCreateOptionTypesFromImage
	osDiskStorageAccountType, err := ToEnumStrValue[armcompute.StorageAccountTypes](config.OSDisk.StorageAccountType, armcompute.PossibleStorageAccountTypesValues())
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}
	dataDiskCreateOption, err := ToEnumStrValue[armcompute.DiskCreateOptionTypes](config.DataDisk.CreateOption, armcompute.PossibleDiskCreateOptionTypesValues())
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	dataDiskCaching, err := ToEnumStrValue[armcompute.CachingTypes](config.DataDisk.Caching, armcompute.PossibleCachingTypesValues())
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}
	dataDiskStorageAccountType, err := ToEnumStrValue[armcompute.StorageAccountTypes](config.DataDisk.StorageAccountType, armcompute.PossibleStorageAccountTypesValues())
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	var ppgSubResource *armcompute.SubResource
	if config.ProximityPlacementGroupID != nil {
		ppgSubResource = &armcompute.SubResource{
			ID: config.ProximityPlacementGroupID,
		}
	}

	imageReference := &armcompute.ImageReference{}
	sourceImageIdLower := strings.ToLower(config.SourceImageID)
	if strings.HasPrefix(sourceImageIdLower, "/communitygalleries") {
		imageReference.CommunityGalleryImageID = &config.SourceImageID
	} else {
		imageReference.ID = &config.SourceImageID
	}

	var nics []*armcompute.VirtualMachineScaleSetNetworkConfiguration

	primaryNicConfig := getPrimaryNicConfig(&config.PrimaryNIC)
	nics = append(nics, primaryNicConfig)

	if config.SecondaryNICs != nil {
		secondaryNicsConfig := getSecondaryNicsConfig(config.SecondaryNICs)
		nics = append(nics, secondaryNicsConfig...)
	}

	var healthProbe *armcompute.APIEntityReference
	if config.HealthProbeID != "" {
		healthProbe = &armcompute.APIEntityReference{
			ID: &config.HealthProbeID,
		}
	}

	vmss := armcompute.VirtualMachineScaleSet{
		Location: &config.Location,
		Identity: &armcompute.VirtualMachineScaleSetIdentity{
			Type:                   identityType,
			UserAssignedIdentities: identities,
		},
		SKU: &armcompute.SKU{
			Name:     &config.SKU,
			Capacity: &size,
		},
		Tags:  tags,
		Zones: zones,
		Properties: &armcompute.VirtualMachineScaleSetProperties{
			Overprovision: &config.Overprovision,
			UpgradePolicy: &armcompute.UpgradePolicy{
				Mode: upgradeMode,
			},
			SinglePlacementGroup: &config.SinglePlacementGroup,
			OrchestrationMode:    orchestrationMode,
			ScaleInPolicy: &armcompute.ScaleInPolicy{
				ForceDeletion: &forceDeletion,
			},
			ProximityPlacementGroup: ppgSubResource,
			VirtualMachineProfile: &armcompute.VirtualMachineScaleSetVMProfile{
				OSProfile: &armcompute.VirtualMachineScaleSetOSProfile{
					AdminUsername:      &config.AdminUsername,
					ComputerNamePrefix: &config.ComputerNamePrefix,
					LinuxConfiguration: &armcompute.LinuxConfiguration{
						DisablePasswordAuthentication: &config.DisablePasswordAuthentication,
						SSH: &armcompute.SSHConfiguration{
							PublicKeys: []*armcompute.SSHPublicKey{
								{
									KeyData: &config.SshPublicKey,
									Path:    &sshKeyPath,
								},
							},
						},
					},
					CustomData: &config.CustomData,
				},
				StorageProfile: &armcompute.VirtualMachineScaleSetStorageProfile{
					OSDisk: &armcompute.VirtualMachineScaleSetOSDisk{
						CreateOption: &osDiskCreateOption,
						Caching:      osDiskCaching,
						DiskSizeGB:   osDiskSizeGb,
						ManagedDisk: &armcompute.VirtualMachineScaleSetManagedDiskParameters{
							StorageAccountType: osDiskStorageAccountType,
						},
					},
					ImageReference: imageReference,
					DataDisks: []*armcompute.VirtualMachineScaleSetDataDisk{
						{
							Lun:          &config.DataDisk.Lun,
							CreateOption: dataDiskCreateOption,
							Caching:      dataDiskCaching,
							DiskSizeGB:   &config.DataDisk.DiskSizeGB,
							ManagedDisk: &armcompute.VirtualMachineScaleSetManagedDiskParameters{
								StorageAccountType: dataDiskStorageAccountType,
							},
						},
					},
				},
				NetworkProfile: &armcompute.VirtualMachineScaleSetNetworkProfile{
					HealthProbe:                    healthProbe,
					NetworkInterfaceConfigurations: nics,
				},
			},
		},
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, vmScaleSetName, vmss, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}
	resp, err := poller.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{Frequency: time.Second})
	if err != nil {
		err = fmt.Errorf("cannot create/update vmss: %v", err)
		return
	}
	id = resp.VirtualMachineScaleSet.ID
	logger.Info().Msgf("vmss %s created/updated successfully", *id)
	return
}

func getPrimaryNicConfig(primaryNic *PrimaryNIC) *armcompute.VirtualMachineScaleSetNetworkConfiguration {
	var loadBalancerAddrPoolIds []*armcompute.SubResource
	for _, lbId := range primaryNic.IPConfigurations[0].LoadBalancerBackendAddressPoolIDs {
		loadBalancerAddrPoolIds = append(loadBalancerAddrPoolIds, &armcompute.SubResource{ID: &lbId})
	}

	var publicIPConfig *armcompute.VirtualMachineScaleSetPublicIPAddressConfiguration
	if primaryNic.IPConfigurations[0].PublicIPAddress.Assign {
		publicIPConfig = &armcompute.VirtualMachineScaleSetPublicIPAddressConfiguration{
			Name: &primaryNic.IPConfigurations[0].PublicIPAddress.Name,
			Properties: &armcompute.VirtualMachineScaleSetPublicIPAddressConfigurationProperties{
				DNSSettings: &armcompute.VirtualMachineScaleSetPublicIPAddressConfigurationDNSSettings{
					DomainNameLabel: &primaryNic.IPConfigurations[0].PublicIPAddress.DomainNameLabel,
				},
			},
		}
	}

	ipConfigName := "ipconfig0"

	primaryNicConfig := armcompute.VirtualMachineScaleSetNetworkConfiguration{
		Name: &primaryNic.Name,
		Properties: &armcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
			Primary:                     TruePtr(),
			EnableAcceleratedNetworking: &primaryNic.EnableAcceleratedNetworking,
			NetworkSecurityGroup: &armcompute.SubResource{
				ID: &primaryNic.NetworkSecurityGroupID,
			},
			IPConfigurations: []*armcompute.VirtualMachineScaleSetIPConfiguration{
				{
					Name: &ipConfigName,
					Properties: &armcompute.VirtualMachineScaleSetIPConfigurationProperties{
						Primary: &primaryNic.IPConfigurations[0].Primary,
						Subnet: &armcompute.APIEntityReference{
							ID: &primaryNic.IPConfigurations[0].SubnetID,
						},
						LoadBalancerBackendAddressPools: loadBalancerAddrPoolIds,
						PublicIPAddressConfiguration:    publicIPConfig,
					},
				},
			},
		},
	}
	return &primaryNicConfig
}

func getSecondaryNicsConfig(secondaryNics *SecondaryNICs) []*armcompute.VirtualMachineScaleSetNetworkConfiguration {
	nicsConfigs := make([]*armcompute.VirtualMachineScaleSetNetworkConfiguration, secondaryNics.Number)

	var loadBalancerAddrPoolIds []*armcompute.SubResource
	for _, lbId := range secondaryNics.IPConfigurations[0].LoadBalancerBackendAddressPoolIDs {
		loadBalancerAddrPoolIds = append(loadBalancerAddrPoolIds, &armcompute.SubResource{ID: &lbId})
	}

	for i := 0; i < secondaryNics.Number; i++ {
		nicName := fmt.Sprintf("%s-%d", secondaryNics.NamePrefix, i+1)
		ipConfigName := fmt.Sprintf("%s%d", "ipconfig", i+1)

		nicConfig := armcompute.VirtualMachineScaleSetNetworkConfiguration{
			Name: &nicName,
			Properties: &armcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
				EnableAcceleratedNetworking: &secondaryNics.EnableAcceleratedNetworking,
				NetworkSecurityGroup: &armcompute.SubResource{
					ID: &secondaryNics.NetworkSecurityGroupID,
				},
				IPConfigurations: []*armcompute.VirtualMachineScaleSetIPConfiguration{
					{
						Name: &ipConfigName,
						Properties: &armcompute.VirtualMachineScaleSetIPConfigurationProperties{
							Primary: &secondaryNics.IPConfigurations[0].Primary,
							Subnet: &armcompute.APIEntityReference{
								ID: &secondaryNics.IPConfigurations[0].SubnetID,
							},
							LoadBalancerBackendAddressPools: loadBalancerAddrPoolIds,
						},
					},
				},
			},
		}
		nicsConfigs[i] = &nicConfig
	}
	return nicsConfigs
}

func GetCurrentScaleSetConfiguration(ctx context.Context, vmssParams *ScaleSetParams) (config *VMSSConfig, err error) {
	logger := logging.LoggerFromCtx(ctx)

	scaleSet, err := GetScaleSetOrNil(ctx, vmssParams.SubscriptionId, vmssParams.ResourceGroupName, vmssParams.ScaleSetName)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}
	if scaleSet == nil {
		return nil, nil
	}

	config = GetVmssConfig(ctx, vmssParams.ResourceGroupName, scaleSet)
	return
}

func UpdateTagsOnVm(ctx context.Context, subscriptionId, resourceGroupName, vmName string, tags map[string]string) error {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := getCredential(ctx)
	if err != nil {
		return err
	}

	client, err := armcompute.NewVirtualMachinesClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return err
	}

	// get current tags
	vm, err := client.Get(ctx, resourceGroupName, vmName, nil)
	if err != nil {
		err = fmt.Errorf("cannot get vm %s: %v", vmName, err)
		logger.Error().Err(err).Send()
		return err
	}

	vmTags := PtrMapToStrMap(vm.Tags)
	// merge current tags with new tags
	for k, v := range tags {
		vmTags[k] = v
	}

	params := &armcompute.VirtualMachineUpdate{
		Tags: StrMapToPtrMap(vmTags),
	}

	_, err = client.BeginUpdate(ctx, resourceGroupName, vmName, *params, nil)
	if err != nil {
		err = fmt.Errorf("cannot update tags on vm %s: %v", vmName, err)
		logger.Error().Err(err).Send()
		return err
	}
	return nil
}
