package common

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/weka/go-cloud-lib/protocol"
)

type Identity struct {
	IdentityIDs []string `json:"identity_ids"`
	Type        string   `json:"type"`
}

type DataDisk struct {
	Caching            string `json:"caching"`
	CreateOption       string `json:"create_option"`
	DiskSizeGB         int32  `json:"disk_size_gb"`
	Lun                int32  `json:"lun"`
	StorageAccountType string `json:"storage_account_type"`
}

type OSDisk struct {
	Caching            string `json:"caching"`
	StorageAccountType string `json:"storage_account_type"`
	DiskSizeGB         *int32 `json:"disk_size_gb,omitempty"`
}

type PublicIPAddress struct {
	Assign          bool   `json:"assign"`
	DomainNameLabel string `json:"domain_name_label"`
	Name            string `json:"name"`
}

type IPConfiguration struct {
	LoadBalancerBackendAddressPoolIDs []string         `json:"load_balancer_backend_address_pool_ids"`
	Primary                           bool             `json:"primary"`
	PublicIPAddress                   *PublicIPAddress `json:"public_ip_address,omitempty"`
	SubnetID                          string           `json:"subnet_id"`
}

type PrimaryNIC struct {
	EnableAcceleratedNetworking bool              `json:"enable_accelerated_networking"`
	IPConfigurations            []IPConfiguration `json:"ip_configurations"`
	Name                        string            `json:"name"`
	NetworkSecurityGroupID      string            `json:"network_security_group_id"`
}

type SecondaryNICs struct {
	EnableAcceleratedNetworking bool              `json:"enable_accelerated_networking"`
	IPConfigurations            []IPConfiguration `json:"ip_configurations"`
	NamePrefix                  string            `json:"name_prefix"`
	NetworkSecurityGroupID      string            `json:"network_security_group_id"`
	Number                      int               `json:"number"`
}

type VMSSConfig struct {
	Name              string            `json:"name"`
	Location          string            `json:"location"`
	Zones             []string          `json:"zones"`
	ResourceGroupName string            `json:"resource_group_name"`
	SKU               string            `json:"sku"`
	SourceImageID     string            `json:"source_image_id"`
	Tags              map[string]string `json:"tags"`

	UpgradeMode          string `json:"upgrade_mode"`
	OrchestrationMode    string `json:"orchestration_mode"`
	HealthProbeID        string `json:"health_probe_id"`
	Overprovision        bool   `json:"overprovision"`
	SinglePlacementGroup bool   `json:"single_placement_group"`

	Identity           Identity `json:"identity"`
	AdminUsername      string   `json:"admin_username"`
	SshPublicKey       string   `json:"ssh_public_key"`
	ComputerNamePrefix string   `json:"computer_name_prefix"`
	UserData           string   `json:"user_data"`

	DisablePasswordAuthentication bool    `json:"disable_password_authentication"`
	ProximityPlacementGroupID     *string `json:"proximity_placement_group_id,omitempty"`

	OSDisk        OSDisk         `json:"os_disk"`
	DataDisk      DataDisk       `json:"data_disk"`
	PrimaryNIC    PrimaryNIC     `json:"primary_nic"`
	SecondaryNICs *SecondaryNICs `json:"secondary_nics"`

	// ignore the following fields when marshaling to json
	ConfigHash string `json:"-"`
}

// Compares two vmss configs - works with copies of VMSSConfig structs
// NOTES:
// - does not compare "config_hash" and "config_applied_at" tags, and names which include version
func VmssConfigsDiff(old, new VMSSConfig) string {
	old.UserData, new.UserData = "", "" // ignore user data when generating diff (as it may contain sensitive data)
	old.Tags["config_hash"], new.Tags["config_hash"] = "", ""
	old.Tags["config_applied_at"], new.Tags["config_applied_at"] = "", ""
	old.ComputerNamePrefix, new.ComputerNamePrefix = "", ""
	old.Name, new.Name = "", ""
	old.ConfigHash, new.ConfigHash = "", ""

	if len(old.PrimaryNIC.IPConfigurations) == len(new.PrimaryNIC.IPConfigurations) {
		for i := range old.PrimaryNIC.IPConfigurations {
			if old.PrimaryNIC.IPConfigurations[i].PublicIPAddress != nil {
				old.PrimaryNIC.IPConfigurations[i].PublicIPAddress.DomainNameLabel = new.PrimaryNIC.IPConfigurations[i].PublicIPAddress.DomainNameLabel
			}
		}
	}

	if new.OSDisk.DiskSizeGB == nil {
		old.OSDisk.DiskSizeGB = nil
	}

	return cmp.Diff(old, new) // arguments order: (want, got)
}

type VMSSStateVerbose struct {
	VmssName          string            `json:"vmss_name"`
	TargetConfig      VMSSConfig        `json:"target_config"`
	CurrentConfig     *VMSSConfig       `json:"current_config,omitempty"`
	TargetConfigHash  string            `json:"target_config_hash"`
	CurrentConfigHash string            `json:"current_config_hash"`
	NeedUpdate        bool              `json:"need_update"`
	UpdatesLog        []protocol.Update `json:"updates,omitempty"`
}

func ToEnumStrValue[T interface{ ~string }](val string, possibleEnumValues []T) (*T, error) {
	for _, enumVal := range possibleEnumValues {
		if val == string(enumVal) {
			return &enumVal, nil
		}
	}
	err := fmt.Errorf("invalid value %s, possible values are %v", val, possibleEnumValues)
	return nil, err
}

func TruePtr() *bool {
	b := true
	return &b
}

func FalsePtr() *bool {
	b := false
	return &b
}

func PtrArrToStrArray(arr []*string) []string {
	result := make([]string, len(arr))
	for i, s := range arr {
		result[i] = *s
	}
	return result
}

func PtrMapToStrMap(m map[string]*string) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = *v
	}
	return result
}

func StrArrToPtrArray(arr []string) []*string {
	result := make([]*string, len(arr))
	for i, s := range arr {
		copyS := s
		result[i] = &copyS
	}
	return result
}

func StrMapToPtrMap(m map[string]string) map[string]*string {
	result := make(map[string]*string, len(m))
	for k, v := range m {
		copyV := v
		result[k] = &copyV
	}
	return result
}
