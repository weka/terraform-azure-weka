package common

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/lease"
	"github.com/google/uuid"
	"github.com/weka/go-cloud-lib/lib/types"
	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
	reportLib "github.com/weka/go-cloud-lib/report"
)

const WekaAdminUsername = "admin"

type InvokeRequest struct {
	Data     map[string]json.RawMessage
	Metadata map[string]interface{}
}

type InvokeResponse struct {
	Outputs     map[string]interface{}
	Logs        []string
	ReturnValue interface{}
}

const FindDrivesScript = `
import json
import sys
for d in json.load(sys.stdin)['disks']:
	if d['isRotational'] or 'nvme' not in d['devPath']: continue
	print(d['devPath'])
`

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
	resData["body"] = err.Error()

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

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Msgf("azidentity.NewDefaultAzureCredential: %s", err)
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

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Msgf("azidentity.NewDefaultAzureCredential: %s", err)
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

func ReadBlobObject(ctx context.Context, storageName, containerName, blobName string) (state []byte, err error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Msgf("azidentity.NewDefaultAzureCredential: %s", err)
		return
	}

	blobClient, err := azblob.NewClient(getBlobUrl(storageName), credential, nil)
	if err != nil {
		logger.Error().Msgf("azblob.NewClient: %s", err)
		return
	}

	downloadResponse, err := blobClient.DownloadStream(ctx, containerName, blobName, nil)
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

func ReadState(ctx context.Context, stateStorageName, containerName string) (state protocol.ClusterState, err error) {
	logger := logging.LoggerFromCtx(ctx)

	stateAsByteArray, err := ReadBlobObject(ctx, stateStorageName, containerName, "state")
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

func WriteBlobObject(ctx context.Context, storageName, containerName, blobName string, state []byte) (err error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	blobClient, err := azblob.NewClient(getBlobUrl(storageName), credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	_, err = blobClient.UploadBuffer(ctx, containerName, blobName, state, &azblob.UploadBufferOptions{})

	return

}

func WriteState(ctx context.Context, stateStorageName, containerName string, state protocol.ClusterState) (err error) {
	logger := logging.LoggerFromCtx(ctx)

	stateAsByteArray, err := json.Marshal(state)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	err = WriteBlobObject(ctx, stateStorageName, containerName, "state", stateAsByteArray)
	return
}

func getBlobUrl(storageName string) string {
	return fmt.Sprintf("https://%s.blob.core.windows.net/", storageName)
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

func AddInstanceToState(ctx context.Context, subscriptionId, resourceGroupName, stateStorageName, stateContainerName, newInstance string) (state protocol.ClusterState, err error) {
	logger := logging.LoggerFromCtx(ctx)

	leaseId, err := LockContainer(ctx, stateStorageName, stateContainerName)
	if err != nil {
		return
	}
	defer UnlockContainer(ctx, stateStorageName, stateContainerName, leaseId)

	state, err = ReadState(ctx, stateStorageName, stateContainerName)
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
		err = WriteState(ctx, stateStorageName, stateContainerName, state)
	}
	return
}

func UpdateClusterized(ctx context.Context, subscriptionId, resourceGroupName, stateStorageName, stateContainerName string) (state protocol.ClusterState, err error) {
	logger := logging.LoggerFromCtx(ctx)

	leaseId, err := LockContainer(ctx, stateStorageName, stateContainerName)
	if err != nil {
		return
	}
	defer UnlockContainer(ctx, stateStorageName, stateContainerName, leaseId)

	state, err = ReadState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		return
	}

	state.Instances = []string{}
	state.Clusterized = true

	err = WriteState(ctx, stateStorageName, stateContainerName, state)

	logger.Info().Msg("State updated to 'clusterized'")
	return
}

func CreateStorageAccount(ctx context.Context, subscriptionId, resourceGroupName, obsName, location string) (accessKey string, err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("creating storage account: %s", obsName)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	client, err := armstorage.NewAccountsClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}
	skuName := armstorage.SKUNameStandardZRS
	kind := armstorage.KindStorageV2
	_, err = client.BeginCreate(ctx, resourceGroupName, obsName, armstorage.AccountCreateParameters{
		Kind:     &kind,
		Location: &location,
		SKU: &armstorage.SKU{
			Name: &skuName,
		},
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

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Err(err).Send()
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

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Err(err).Send()
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

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	client, err := azsecrets.NewClient(keyVaultUri, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}
	resp, err := client.GetSecret(ctx, secretName, "", nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	secret = *resp.Value

	return
}

// Gets all network interfaces in a VM scale set
// see https://learn.microsoft.com/en-us/rest/api/virtualnetwork/network-interface-in-vm-ss/list-virtual-machine-scale-set-network-interfaces
func getScaleSetVmsNetworkInterfaces(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string) (networkInterfaces []*armnetwork.Interface, err error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Err(err).Send()
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

func GetScaleSetVmsNetworkPrimaryNICs(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string) (networkInterfaces []*armnetwork.Interface, err error) {
	nics, err := getScaleSetVmsNetworkInterfaces(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
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

func GetPublicIp(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName, prefix, clusterName, instanceIndex string) (publicIp string, err error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	client, err := armnetwork.NewPublicIPAddressesClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}
	interfaceName := fmt.Sprintf("%s-%s-backend-nic", prefix, clusterName)
	pager := client.NewListVirtualMachineScaleSetVMPublicIPAddressesPager(resourceGroupName, vmScaleSetName, instanceIndex, interfaceName, "ipconfig1", nil)

	for pager.More() {
		nextResult, err1 := pager.NextPage(ctx)
		if err1 != nil {
			logger.Error().Err(err1).Send()
			return "", err1
		}
		publicIp = *nextResult.Value[0].Properties.IPAddress
		return
	}
	return
}

func GetVmsPrivateIps(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string) (vmsPrivateIps map[string]string, err error) {
	//returns compute_name to private ip map

	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msg("fetching scale set vms private ips")

	networkInterfaces, err := GetScaleSetVmsNetworkPrimaryNICs(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
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

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Err(err).Send()
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

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return nil, err
	}

	client, err := armauthorization.NewRoleDefinitionsClient(cred, nil)
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

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return nil, err
	}

	client, err := armauthorization.NewRoleAssignmentsClient(subscriptionId, cred, nil)
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

type ScaleSetInfo struct {
	Id            string
	Name          string
	AdminUsername string
	AdminPassword string
	Capacity      int
	VMSize        string
}

// Gets scale set
// see https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-scale-sets/get
func getScaleSet(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string) (*armcompute.VirtualMachineScaleSet, error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("Getting scale set %s info", vmScaleSetName)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Err(err).Send()
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
		if errors.As(err, &responseErr) && responseErr.ErrorCode == "ResourceNotFound" {
			// scale set is not found
			return nil, nil
		}
		logger.Error().Err(err).Send()
		return nil, err
	}
	return scaleSet, nil
}

// Gets single scale set info
func GetScaleSetInfo(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName, keyVaultUri string) (*ScaleSetInfo, error) {
	logger := logging.LoggerFromCtx(ctx)

	scaleSet, err := getScaleSet(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		logger.Error().Err(err).Send()
		return nil, err
	}

	wekaPassword, err := GetWekaClusterPassword(ctx, keyVaultUri)
	if err != nil {
		logger.Error().Err(err).Send()
		return nil, err
	}

	scaleSetInfo := ScaleSetInfo{
		Id:            *scaleSet.ID,
		Name:          *scaleSet.Name,
		AdminUsername: WekaAdminUsername,
		AdminPassword: wekaPassword,
		Capacity:      int(*scaleSet.SKU.Capacity),
		VMSize:        *scaleSet.SKU.Name,
	}
	return &scaleSetInfo, err
}

// Gets a list of all VMs in a scale set
// see https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-scale-set-vms/list
func GetScaleSetInstances(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string, expand *string) (vms []*armcompute.VirtualMachineScaleSetVM, err error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	client, err := armcompute.NewVirtualMachineScaleSetVMsClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	pager := client.NewListPager(
		resourceGroupName, vmScaleSetName, &armcompute.VirtualMachineScaleSetVMsClientListOptions{
			Expand: expand,
		})

	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			err = fmt.Errorf("failed to advance page getting images list: %v", err)
			logger.Error().Err(err).Send()
			return nil, err
		}
		vms = append(vms, nextResult.Value...)
	}
	return
}

type ScaleSetInstanceInfo struct {
	Id        string
	PrivateIp string
}

func GetScaleSetVmId(resourceId string) string {
	vmNameParts := strings.Split(resourceId, "/")
	vmNamePartsLen := len(vmNameParts)
	vmId := vmNameParts[vmNamePartsLen-1]
	return vmId
}

func GetScaleSetInstancesInfo(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string) (instances []ScaleSetInstanceInfo, err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("Getting scale set instances %s info", vmScaleSetName)

	netInterfaces, err := GetScaleSetVmsNetworkPrimaryNICs(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		return
	}
	instanceIdPrivateIp := map[string]string{}

	for _, ni := range netInterfaces {
		id := GetScaleSetVmId(*ni.Properties.VirtualMachine.ID)
		privateIp := *ni.Properties.IPConfigurations[0].Properties.PrivateIPAddress
		instanceIdPrivateIp[id] = privateIp
	}

	vms, err := GetScaleSetInstances(ctx, subscriptionId, resourceGroupName, vmScaleSetName, nil)
	if err != nil {
		return
	}
	for _, vm := range vms {
		id := GetScaleSetVmId(*vm.ID)
		// get private ip if exists
		var privateIp string
		if val, ok := instanceIdPrivateIp[id]; ok {
			privateIp = val
		}
		instanceInfo := ScaleSetInstanceInfo{
			Id:        id,
			PrivateIp: privateIp,
		}
		instances = append(instances, instanceInfo)
	}
	return
}

func GetScaleSetVmIndex(vmName string) string {
	instanceNameParts := strings.Split(vmName, "_")
	return instanceNameParts[len(instanceNameParts)-1]
}

func SetDeletionProtection(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName, instanceId string, protect bool) (err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("Setting deletion protection: %t on instanceId %s", protect, instanceId)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	client, err := armcompute.NewVirtualMachineScaleSetVMsClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	_, err = client.BeginUpdate(
		ctx,
		resourceGroupName,
		vmScaleSetName,
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
	ctx context.Context, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, vmScaleSetName, instanceId, hostName string,
	maxAttempts int, sleepInterval time.Duration,
) (err error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Info().Msgf("Setting deletion protection on %s", hostName)
	counter := 0
	for {
		err = SetDeletionProtection(ctx, subscriptionId, resourceGroupName, vmScaleSetName, instanceId, true)
		if err == nil {
			msg := "Deletion protection was set successfully"
			logger.Info().Msg(msg)
			ReportMsg(ctx, hostName, stateContainerName, stateStorageName, "progress", msg)
			break
		}

		if protectionErr, ok := err.(*azcore.ResponseError); ok && protectionErr.ErrorCode == "AuthorizationFailed" {
			counter++
			// deletion protection invoked by terminate function
			if maxAttempts == 0 {
				msg := "Deletion protection set authorization isn't ready, will retry on next scale down workflow"
				ReportMsg(ctx, hostName, stateContainerName, stateStorageName, "debug", msg)
				return
			}

			if counter > maxAttempts {
				break
			}
			msg := fmt.Sprintf("Deletion protection set authorization isn't ready, going to sleep for %s", sleepInterval)
			logger.Info().Msg(msg)
			ReportMsg(ctx, hostName, stateContainerName, stateStorageName, "debug", msg)
			time.Sleep(sleepInterval)
		} else {
			break
		}
	}
	if err != nil {
		logger.Error().Err(err).Send()
		ReportMsg(ctx, hostName, stateContainerName, stateStorageName, "error", err.Error())
	}
	return
}

func ReportMsg(ctx context.Context, hostName, stateContainerName, stateStorageName, reportType, message string) {
	reportObj := protocol.Report{Type: reportType, Hostname: hostName, Message: message}
	_ = UpdateStateReporting(ctx, stateContainerName, stateStorageName, reportObj)
}

func GetWekaClusterPassword(ctx context.Context, keyVaultUri string) (password string, err error) {
	return GetKeyVaultValue(ctx, keyVaultUri, "weka-password")
}

func GetVmScaleSetName(prefix, clusterName string) string {
	return fmt.Sprintf("%s-%s-vmss", prefix, clusterName)
}

func GetScaleSetInstanceIds(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string) (instanceIds []string, err error) {
	vms, err := GetScaleSetInstances(ctx, subscriptionId, resourceGroupName, vmScaleSetName, nil)
	if err != nil {
		return
	}

	for _, vm := range vms {
		instanceIds = append(instanceIds, GetScaleSetVmId(*vm.ID))
	}

	return
}

type InstanceIdsSet map[string]types.Nilt

func GetInstanceIpsSet(scaleResponse protocol.ScaleResponse) InstanceIdsSet {
	instanceIpsSet := make(InstanceIdsSet)
	for _, instance := range scaleResponse.Hosts {
		instanceIpsSet[instance.PrivateIp] = types.Nilv
	}
	return instanceIpsSet
}

func FilterSpecificScaleSetInstances(ctx context.Context, allVms []*armcompute.VirtualMachineScaleSetVM, instanceIds []string) (vms []*armcompute.VirtualMachineScaleSetVM, err error) {
	instanceIdsSet := make(InstanceIdsSet)
	for _, instanceId := range instanceIds {
		instanceIdsSet[instanceId] = types.Nilv
	}

	for _, vm := range allVms {
		if _, ok := instanceIdsSet[GetScaleSetVmId(*vm.ID)]; ok {
			vms = append(vms, vm)
		}
	}

	return
}

func TerminateScaleSetInstances(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string, terminateInstanceIds []string) (terminatedInstances []string, errs []error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	client, err := armcompute.NewVirtualMachineScaleSetVMsClient(subscriptionId, credential, nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	if len(terminateInstanceIds) == 0 {
		return
	}
	force := true
	for _, instanceId := range terminateInstanceIds {
		err = SetDeletionProtection(ctx, subscriptionId, resourceGroupName, vmScaleSetName, instanceId, false)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		logger.Info().Msgf("Deleting instanceId %s", instanceId)
		_, err = client.BeginDelete(ctx, resourceGroupName, vmScaleSetName, instanceId, &armcompute.VirtualMachineScaleSetVMsClientBeginDeleteOptions{
			ForceDeletion: &force,
		})
		if err != nil {
			logger.Error().Err(err).Send()
			errs = append(errs, err)
			continue
		}
		terminatedInstances = append(terminatedInstances, instanceId)
	}

	return
}

func UpdateStateReporting(ctx context.Context, stateContainerName, stateStorageName string, report protocol.Report) (err error) {
	leaseId, err := LockContainer(ctx, stateStorageName, stateContainerName)
	if err != nil {
		return
	}
	defer UnlockContainer(ctx, stateStorageName, stateContainerName, leaseId)

	return UpdateStateReportingWithoutLocking(ctx, stateContainerName, stateStorageName, report)
}

func AddClusterUpdate(ctx context.Context, stateContainerName, stateStorageName string, update protocol.Update) (err error) {
	leaseId, err := LockContainer(ctx, stateStorageName, stateContainerName)
	if err != nil {
		return
	}
	defer UnlockContainer(ctx, stateStorageName, stateContainerName, leaseId)

	state, err := ReadState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		return
	}

	reportLib.AddClusterUpdate(update, &state)

	err = WriteState(ctx, stateStorageName, stateContainerName, state)
	if err != nil {
		err = fmt.Errorf("failed addind cluster update to state")
		return
	}
	return
}

func UpdateStateReportingWithoutLocking(ctx context.Context, stateContainerName, stateStorageName string, report protocol.Report) (err error) {
	state, err := ReadState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		return
	}
	err = reportLib.UpdateReport(report, &state)
	if err != nil {
		err = fmt.Errorf("failed updating state report")
		return
	}
	err = WriteState(ctx, stateStorageName, stateContainerName, state)
	if err != nil {
		err = fmt.Errorf("failed updating state report")
		return
	}
	return
}

func GetInstancePowerState(instance *armcompute.VirtualMachineScaleSetVM) (powerState string) {
	prefix := "PowerState/"
	for _, status := range instance.Properties.InstanceView.Statuses {
		if strings.HasPrefix(*status.Code, prefix) {
			powerState = strings.TrimPrefix(*status.Code, prefix)
			return
		}
	}
	return
}

func GetInstanceProvisioningState(instance *armcompute.VirtualMachineScaleSetVM) (provisioningState string) {
	provisioningState = "unknown"
	if instance.Properties.ProvisioningState != nil {
		provisioningState = *instance.Properties.ProvisioningState
	}
	return strings.ToLower(provisioningState)
}

func GetUnhealthyInstancesToTerminate(ctx context.Context, scaleSetVms []*armcompute.VirtualMachineScaleSetVM) (toTerminate []string) {
	logger := logging.LoggerFromCtx(ctx)

	for _, vm := range scaleSetVms {
		if vm.Properties.InstanceView == nil || vm.Properties.InstanceView.VMHealth == nil {
			continue
		}
		healthStatus := *vm.Properties.InstanceView.VMHealth.Status.Code
		if healthStatus == "HealthState/unhealthy" {
			instancePowerState := GetInstancePowerState(vm)
			instanceProvisioningState := GetInstanceProvisioningState(vm)
			logger.Debug().Msgf("instance power state: %s, provisioning state: %s", instancePowerState, instanceProvisioningState)
			if instancePowerState == "stopped" || instanceProvisioningState == "failed" {
				toTerminate = append(toTerminate, GetScaleSetVmId(*vm.ID))
			}
		}
	}

	logger.Info().Msgf("found %d unhealthy stopped instances to terminate: %s", len(toTerminate), toTerminate)
	return
}

func GetScaleSetVmsExpandedView(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string) ([]*armcompute.VirtualMachineScaleSetVM, error) {
	expand := "instanceView"
	return GetScaleSetInstances(ctx, subscriptionId, resourceGroupName, vmScaleSetName, &expand)
}

func GetAzureInstanceNameCmd() string {
	return "curl -s -H Metadata:true --noproxy * http://169.254.169.254/metadata/instance?api-version=2021-02-01 | jq '.compute.name' | cut -c2- | rev | cut -c2- | rev"
}

func ReadVmssConfig(ctx context.Context, storageName, containerName string) (vmssConfig VMSSConfig, err error) {
	logger := logging.LoggerFromCtx(ctx)

	asByteArray, err := ReadBlobObject(ctx, storageName, containerName, "vmss-config")
	if err != nil {
		return
	}

	// calculate hash of the config (used to identify vmss config changes)
	hash := sha256.Sum256(asByteArray)
	hashStr := fmt.Sprintf("%x", hash)

	err = json.Unmarshal(asByteArray, &vmssConfig)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

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
		HealthProbeID:        *scaleSet.Properties.VirtualMachineProfile.NetworkProfile.HealthProbe.ID,
		Overprovision:        *scaleSet.Properties.Overprovision,
		SinglePlacementGroup: *scaleSet.Properties.SinglePlacementGroup,

		Identity: Identity{
			IdentityIDs: identityIds,
			Type:        string(*scaleSet.Identity.Type),
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
		SecondaryNICs: *secondaryNics,
		ConfigHash:    configHash,
	}
	return vmssConfig
}

func CreateOrUpdateVmss(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName, configHash string, config VMSSConfig, vmssSize int) (id *string, err error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Err(err).Send()
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

	secondaryNicsConfig := getSecondaryNicsConfig(&config.SecondaryNICs)
	nics = append(nics, secondaryNicsConfig...)

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
			ProximityPlacementGroup: &armcompute.SubResource{
				ID: config.ProximityPlacementGroupID,
			},
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
					HealthProbe: &armcompute.APIEntityReference{
						ID: &config.HealthProbeID,
					},
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

func GetCurrentScaleSetConfiguration(ctx context.Context, subscriptionId, resourceGroupName, vmScaleSetName string) (config *VMSSConfig, err error) {
	logger := logging.LoggerFromCtx(ctx)

	scaleSet, err := GetScaleSetOrNil(ctx, subscriptionId, resourceGroupName, vmScaleSetName)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}
	if scaleSet == nil {
		return nil, nil
	}

	config = GetVmssConfig(ctx, resourceGroupName, scaleSet)
	return
}
