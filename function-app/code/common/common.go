package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/google/uuid"
	"github.com/weka/go-cloud-lib/lib/types"
	"github.com/weka/go-cloud-lib/logging"
	"github.com/weka/go-cloud-lib/protocol"
	reportLib "github.com/weka/go-cloud-lib/report"
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

const FindDrivesScript = `
import json
import sys
for d in json.load(sys.stdin)['disks']:
	if d['isRotational'] or 'nvme' not in d['devPath']: continue
	print(d['devPath'])
`

func leaseContainer(ctx context.Context, subscriptionId, resourceGroupName, storageAccountName, containerName string, leaseIdIn *string, action armstorage.LeaseContainerRequestAction) (leaseIdOut *string, err error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Msgf("azidentity.NewDefaultAzureCredential: %s", err)
		return
	}

	containerClient, err := armstorage.NewBlobContainersClient(subscriptionId, credential, nil)
	duration := int32(60)
	for i := 1; i < 1000; i++ {
		lease, err2 := containerClient.Lease(ctx, resourceGroupName, storageAccountName, containerName,
			&armstorage.BlobContainersClientLeaseOptions{
				Parameters: &armstorage.LeaseContainerRequest{
					Action:        &action,
					LeaseDuration: &duration,
					LeaseID:       leaseIdIn,
				},
			})
		err = err2
		if err != nil {
			if leaseErr, ok := err.(*azcore.ResponseError); ok && leaseErr.ErrorCode == "ContainerOperationFailure" {
				buf := new(bytes.Buffer)
				buf.ReadFrom(leaseErr.RawResponse.Body)
				if !strings.Contains(buf.String(), "LeaseAlreadyPresent") {
					logger.Error().Err(err).Send()
					return
				}
				logger.Debug().Msg("lease in use, will retry in 1 sec")
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

func LockContainer(ctx context.Context, subscriptionId, resourceGroupName, storageAccountName, containerName string) (*string, error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Debug().Msgf("locking %s", containerName)
	return leaseContainer(ctx, subscriptionId, resourceGroupName, storageAccountName, containerName, nil, armstorage.LeaseContainerRequestActionAcquire)
}

func UnlockContainer(ctx context.Context, subscriptionId, resourceGroupName, storageAccountName, containerName string, leaseId *string) (*string, error) {
	logger := logging.LoggerFromCtx(ctx)
	logger.Debug().Msgf("unlocking %s", containerName)
	return leaseContainer(ctx, subscriptionId, resourceGroupName, storageAccountName, containerName, leaseId, armstorage.LeaseContainerRequestActionRelease)
}

func ReadBlobObject(ctx context.Context, stateStorageName, containerName, blobName string) (state []byte, err error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Msgf("azidentity.NewDefaultAzureCredential: %s", err)
		return
	}

	blobClient, err := azblob.NewClient(getBlobUrl(stateStorageName), credential, nil)
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

func WriteBlobObject(ctx context.Context, stateStorageName, containerName, blobName string, state []byte) (err error) {
	logger := logging.LoggerFromCtx(ctx)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		logger.Error().Err(err).Send()
		return
	}

	blobClient, err := azblob.NewClient(getBlobUrl(stateStorageName), credential, nil)
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

type ShutdownRequired struct {
	Message string
}

func (e *ShutdownRequired) Error() string {
	return e.Message
}

func AddInstanceToState(ctx context.Context, subscriptionId, resourceGroupName, stateStorageName, stateContainerName, newInstance string) (state protocol.ClusterState, err error) {
	logger := logging.LoggerFromCtx(ctx)

	leaseId, err := LockContainer(ctx, subscriptionId, resourceGroupName, stateStorageName, stateContainerName)
	if err != nil {
		return
	}

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
	_, err2 := UnlockContainer(ctx, subscriptionId, resourceGroupName, stateStorageName, stateContainerName, leaseId)
	if err2 != nil {
		if err == nil {
			err = err2
		}
		logger.Error().Msgf("unlocking %s failed", stateStorageName)
	}
	return
}

func UpdateClusterized(ctx context.Context, subscriptionId, resourceGroupName, stateStorageName, stateContainerName string) (state protocol.ClusterState, err error) {
	logger := logging.LoggerFromCtx(ctx)

	leaseId, err := LockContainer(ctx, subscriptionId, resourceGroupName, stateStorageName, stateContainerName)
	if err != nil {
		return
	}

	state, err = ReadState(ctx, stateStorageName, stateContainerName)
	if err != nil {
		return
	}

	state.Instances = []string{}
	state.Clusterized = true
	err = WriteState(ctx, stateStorageName, stateContainerName, state)
	_, err2 := UnlockContainer(ctx, subscriptionId, resourceGroupName, stateStorageName, stateContainerName, leaseId)
	if err2 != nil {
		if err == nil {
			err = err2
		}
		logger.Error().Msgf("unlocking %s failed", stateStorageName)
	}
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

	// see https://learn.microsoft.com/en-us/rest/api/authorization/role-assignments/create
	res, err := client.Create(
		ctx,
		scope,
		uuid.New().String(), // az docs say it should be GUID
		armauthorization.RoleAssignmentCreateParameters{
			Properties: &armauthorization.RoleAssignmentProperties{
				RoleDefinitionID: roleDefinition.ID,
				PrincipalID:      scaleSet.Identity.PrincipalID,
			},
		},
		nil,
	)
	if err != nil {
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
		AdminUsername: "admin",
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
			ReportMsg(ctx, hostName, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, "progress", msg)
			break
		}

		if protectionErr, ok := err.(*azcore.ResponseError); ok && protectionErr.ErrorCode == "AuthorizationFailed" {
			counter++
			// deletion protection invoked by terminate function
			if maxAttempts == 0 {
				msg := fmt.Sprintf("Deletion protection set authorization isn't ready, will retry on next scale down workflow")
				ReportMsg(ctx, hostName, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, "debug", msg)
				return
			}

			if counter > maxAttempts {
				break
			}
			msg := fmt.Sprintf("Deletion protection set authorization isn't ready, going to sleep for %s", sleepInterval)
			logger.Info().Msg(msg)
			ReportMsg(ctx, hostName, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, "debug", msg)
			time.Sleep(sleepInterval)
		} else {
			break
		}
	}
	if err != nil {
		logger.Error().Err(err).Send()
		ReportMsg(ctx, hostName, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, "error", err.Error())
	}
	return
}

func ReportMsg(ctx context.Context, hostName, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, reportType, message string) {
	reportObj := protocol.Report{Type: reportType, Hostname: hostName, Message: message}
	_ = UpdateStateReporting(ctx, subscriptionId, resourceGroupName, stateContainerName, stateStorageName, reportObj)
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

func UpdateStateReporting(ctx context.Context, subscriptionId, resourceGroupName, stateContainerName, stateStorageName string, report protocol.Report) (err error) {
	logger := logging.LoggerFromCtx(ctx)

	leaseId, err := LockContainer(ctx, subscriptionId, resourceGroupName, stateStorageName, stateContainerName)
	if err != nil {
		return
	}

	err = UpdateStateReportingWithoutLocking(ctx, stateContainerName, stateStorageName, report)

	_, err2 := UnlockContainer(ctx, subscriptionId, resourceGroupName, stateStorageName, stateContainerName, leaseId)
	if err2 != nil {
		if err == nil {
			err = err2
		}
		logger.Error().Msgf("unlocking %s failed", stateStorageName)
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
