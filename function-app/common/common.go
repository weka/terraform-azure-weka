package common

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/rs/zerolog/log"
	"io"
	"strings"
	"time"
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

type ClusterState struct {
	InitialSize int      `json:"initial_size"`
	DesiredSize int      `json:"desired_size"`
	Instances   []string `json:"instances"`
	Clusterized bool     `json:"clusterized"`
}

func leaseContainer(subscriptionId, resourceGroupName, storageAccountName, containerName string, leaseIdIn *string, action armstorage.LeaseContainerRequestAction) (leaseIdOut *string, err error) {
	ctx := context.Background()

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Error().Msgf("azidentity.NewDefaultAzureCredential: %s", err)
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
					log.Error().Msgf("%s", err)
					return
				}
				log.Debug().Msg("lease in use, will retry in 1 sec")
				time.Sleep(time.Second)
			} else {
				log.Error().Msgf("%s", err)
				return
			}
		} else {
			leaseIdOut = lease.LeaseID
			return
		}
	}

	log.Error().Msgf("%s", err)
	return
}

func LockContainer(subscriptionId, resourceGroupName, storageAccountName, containerName string) (*string, error) {
	log.Debug().Msgf("locking %s", containerName)
	return leaseContainer(subscriptionId, resourceGroupName, storageAccountName, containerName, nil, armstorage.LeaseContainerRequestActionAcquire)
}

func UnlockContainer(subscriptionId, resourceGroupName, storageAccountName, containerName string, leaseId *string) (*string, error) {
	log.Debug().Msgf("unlocking %s", containerName)
	return leaseContainer(subscriptionId, resourceGroupName, storageAccountName, containerName, leaseId, armstorage.LeaseContainerRequestActionRelease)
}

func ReadBlobObject(stateStorageName, containerName, blobName string) (state []byte, err error) {
	ctx := context.Background()

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Error().Msgf("azidentity.NewDefaultAzureCredential: %s", err)
		return
	}

	blobClient, err := azblob.NewClient(getBlobUrl(stateStorageName), credential, nil)
	if err != nil {
		log.Error().Msgf("azblob.NewClient: %s", err)
		return
	}

	downloadResponse, err := blobClient.DownloadStream(ctx, containerName, blobName, nil)
	if err != nil {
		log.Error().Msgf("blobClient.DownloadStream: %s", err)
		return
	}

	state, err = io.ReadAll(downloadResponse.Body)
	if err != nil {
		log.Error().Msgf("%s", err)
	}

	return

}

func ReadState(stateStorageName, containerName string) (state ClusterState, err error) {
	stateAsByteArray, err := ReadBlobObject(stateStorageName, containerName, "state")
	if err != nil {
		return
	}
	err = json.Unmarshal(stateAsByteArray, &state)
	if err != nil {
		log.Error().Msgf("%s", err)
		return
	}

	return
}

func WriteBlobObject(stateStorageName, containerName, blobName string, state []byte) (err error) {
	ctx := context.Background()

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Error().Msgf("%s", err)
		return
	}

	blobClient, err := azblob.NewClient(getBlobUrl(stateStorageName), credential, nil)
	if err != nil {
		log.Error().Msgf("%s", err)
		return
	}

	_, err = blobClient.UploadBuffer(ctx, containerName, blobName, state, &azblob.UploadBufferOptions{})

	return

}

func WriteState(stateStorageName, containerName string, state ClusterState) (err error) {
	stateAsByteArray, err := json.Marshal(state)
	if err != nil {
		log.Error().Msgf("%s", err)
		return
	}

	err = WriteBlobObject(stateStorageName, containerName, "state", stateAsByteArray)
	return
}

func getBlobUrl(storageName string) string {
	return fmt.Sprintf("https://%s.blob.core.windows.net/", storageName)
}

func AddInstanceToState(subscriptionId, resourceGroupName, stateStorageName, stateContainerName, newInstance string) (state ClusterState, err error) {
	leaseId, err := LockContainer(subscriptionId, resourceGroupName, stateStorageName, stateContainerName)
	if err != nil {
		return
	}

	state, err = ReadState(stateStorageName, stateContainerName)
	if err != nil {
		return
	}

	if len(state.Instances) >= state.InitialSize {
		err = errors.New("cluster size is already satisfied")
	} else {
		state.Instances = append(state.Instances, newInstance)
		err = WriteState(stateStorageName, stateContainerName, state)
	}
	_, err2 := UnlockContainer(subscriptionId, resourceGroupName, stateStorageName, stateContainerName, leaseId)
	if err2 != nil {
		if err == nil {
			err = err2
		}
		log.Error().Msgf("unlocking %s failed", stateStorageName)
	}
	return
}

func UpdateClusterized(subscriptionId, resourceGroupName, stateStorageName, stateContainerName string) (state ClusterState, err error) {
	leaseId, err := LockContainer(subscriptionId, resourceGroupName, stateStorageName, stateContainerName)
	if err != nil {
		return
	}

	state, err = ReadState(stateStorageName, stateContainerName)
	if err != nil {
		return
	}

	state.Clusterized = true
	err = WriteState(stateStorageName, stateContainerName, state)
	_, err2 := UnlockContainer(subscriptionId, resourceGroupName, stateStorageName, stateContainerName, leaseId)
	if err2 != nil {
		if err == nil {
			err = err2
		}
		log.Error().Msgf("unlocking %s failed", stateStorageName)
	}
	return
}

func CreateStorageAccount(subscriptionId, resourceGroupName, obsName, location string) (accessKey string, err error) {
	log.Info().Msgf("creating storage account: %s", obsName)
	ctx := context.Background()
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Error().Msgf("%s", err)
		return
	}

	client, err := armstorage.NewAccountsClient(subscriptionId, credential, nil)
	if err != nil {
		log.Error().Msgf("%s", err)
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
				log.Debug().Msgf("storage account %s already exists", obsName)
				err = nil
			} else {
				log.Error().Msgf("storage creation failed: %s", err)
				return
			}
		} else {
			log.Error().Msgf("storage creation failed: %s", err)
			return
		}
	}

	for i := 0; i < 10; i++ {
		accessKey, err = getStorageAccountAccessKey(subscriptionId, resourceGroupName, obsName)

		if err != nil {
			if azerr, ok := err.(*azcore.ResponseError); ok {
				if azerr.ErrorCode == "StorageAccountIsNotProvisioned" {
					log.Debug().Msgf("new storage account is not ready will retry in 1M")
					time.Sleep(time.Minute)
				} else {
					log.Error().Msgf("%s", err)
					return
				}
			} else {
				log.Error().Msgf("%s", err)
				return
			}
		} else {
			log.Debug().Msgf("storage account '%s' is ready for use", obsName)
			break
		}
	}

	return
}

func getStorageAccountAccessKey(subscriptionId, resourceGroupName, obsName string) (accessKey string, err error) {
	ctx := context.Background()
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Error().Msgf("%s", err)
		return
	}

	client, err := armstorage.NewAccountsClient(subscriptionId, credential, nil)
	keys, err := client.ListKeys(ctx, resourceGroupName, obsName, nil)
	if err != nil {
		log.Error().Msgf("%s", err)
		return
	}
	accessKey = *keys.Keys[0].Value
	return
}

func CreateContainer(storageAccountName, containerName string) (err error) {
	log.Info().Msgf("creating obs container %s in storage account %s", containerName, storageAccountName)
	ctx := context.Background()

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Error().Msgf("%s", err)
		return
	}

	blobClient, err := azblob.NewClient(getBlobUrl(storageAccountName), credential, nil)
	if err != nil {
		log.Error().Msgf("%s", err)
		return
	}

	_, err = blobClient.CreateContainer(ctx, containerName, nil)
	if err != nil {
		if azerr, ok := err.(*azcore.ResponseError); ok {
			if azerr.ErrorCode == "ContainerAlreadyExists" {
				log.Info().Msgf("obs container %s already exists", containerName)
				err = nil
				return
			}
		}
		log.Error().Msgf("obs container creation failed: %s", err)
	}
	return
}

func GetKeyVaultValue(keyVaultUri, secretName string) (secret string, err error) {
	log.Info().Msgf("fetching key vault secret: %s", secretName)
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Error().Msgf("%s", err)
		return
	}

	ctx := context.Background()
	client, err := azsecrets.NewClient(keyVaultUri, credential, nil)
	resp, err := client.GetSecret(ctx, secretName, "", nil)
	if err != nil {
		log.Error().Msgf("%s", err)
		return
	}

	secret = *resp.Value

	return
}

func GetVmsPrivateIps(subscriptionId, resourceGroupName string, vmNames []string) (privateIps []string, err error) {
	log.Info().Msg("fetching cluster vms private ips")
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Error().Msgf("%s", err)
		return
	}

	vmClient, err := armcompute.NewVirtualMachinesClient(subscriptionId, credential, nil)
	if err != nil {
		log.Error().Msgf("%s", err)
		return
	}

	nicClient, err := armnetwork.NewInterfacesClient(subscriptionId, credential, nil)
	if err != nil {
		log.Error().Msgf("%s", err)
		return
	}

	ctx := context.Background()
	for _, vmName := range vmNames {
		vmResponse, err2 := vmClient.Get(ctx, resourceGroupName, vmName, nil)
		if err2 != nil {
			err = err2
			log.Error().Msgf("failed fetching vm: %s", err)
			return
		}

		privateIp := ""
		for _, nicRef := range vmResponse.Properties.NetworkProfile.NetworkInterfaces {
			nicID, err2 := arm.ParseResourceID(*nicRef.ID)
			if err2 != nil {
				err = err2
				log.Error().Msgf("%s", err)
				return
			}
			nic, err2 := nicClient.Get(ctx, nicID.ResourceGroupName, nicID.Name, nil)
			if err2 != nil {
				err = err2
				log.Error().Msgf("failed fetching vm nic: %s", err)
				return
			}

			for _, ipCfg := range nic.Properties.IPConfigurations {
				if ipCfg.Properties.PrivateIPAddress != nil {
					privateIp = *ipCfg.Properties.PrivateIPAddress
					break
				}
			}
			if privateIp != "" {
				privateIps = append(privateIps, privateIp)
				break
			}
		}
	}

	return

}
