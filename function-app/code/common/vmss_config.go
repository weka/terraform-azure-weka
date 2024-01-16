package common

import (
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
)

type Identity struct {
	IdentityIDs []string `json:"identity_ids"`
	Type        string   `json:"type"`
}

type AdminSSHKey struct {
	PublicKey string `json:"public_key"`
	Username  string `json:"username"`
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
	SizeGB             *int32 `json:"size_gb,omitempty"`
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
	Primary                     bool              `json:"primary"`
}

type SecondaryNICs struct {
	EnableAcceleratedNetworking bool              `json:"enable_accelerated_networking"`
	IPConfigurations            []IPConfiguration `json:"ip_configurations"`
	NamePrefix                  string            `json:"name_prefix"`
	NetworkSecurityGroupID      string            `json:"network_security_group_id"`
	Number                      int               `json:"number"`
	Primary                     bool              `json:"primary"`
}

type VMSSConfig struct {
	Name              string             `json:"name"`
	Location          string             `json:"location"`
	Zones             []string           `json:"zones"`
	ResourceGroupName string             `json:"resource_group_name"`
	SKU               string             `json:"sku"`
	SourceImageID     string             `json:"source_image_id"`
	Tags              map[string]*string `json:"tags"`

	UpgradeMode          string `json:"upgrade_mode"`
	OrchestrationMode    string `json:"orchestration_mode"`
	HealthProbeID        string `json:"health_probe_id"`
	Overprovision        bool   `json:"overprovision"`
	SinglePlacementGroup bool   `json:"single_placement_group"`

	Identity           Identity    `json:"identity"`
	AdminUsername      string      `json:"admin_username"`
	AdminSSHKey        AdminSSHKey `json:"admin_ssh_key"`
	ComputerNamePrefix string      `json:"computer_name_prefix"`
	CustomData         string      `json:"custom_data"`

	DisablePasswordAuthentication bool   `json:"disable_password_authentication"`
	ProximityPlacementGroupID     string `json:"proximity_placement_group_id"`

	OSDisk        OSDisk        `json:"os_disk"`
	DataDisk      DataDisk      `json:"data_disk"`
	PrimaryNIC    PrimaryNIC    `json:"primary_nic"`
	SecondaryNICs SecondaryNICs `json:"secondary_nics"`
}

func VmssConfigsDiff(old, new *VMSSConfig) string {
	return cmp.Diff(new, old) // arguments order: (want, got)
}

type RefreshStatus string

const (
	RefreshNone       RefreshStatus = "none"
	RefreshInProgress RefreshStatus = "in_progress"
	RefreshNeeded     RefreshStatus = "needed"
)

type VMSSState struct {
	VmssVersion   uint16        `json:"vmss_version"`
	RefreshStatus RefreshStatus `json:"refresh_status"`
	CurrentConfig *VMSSConfig   `json:"current_config,omitempty"`
}

func GetRefreshVmssName(outdatedVmssName string, currentVmssVersion uint16) string {
	versionStr := fmt.Sprintf("-v%d", currentVmssVersion)
	newVersionStr := fmt.Sprintf("-v%d", currentVmssVersion+1)

	vmssNameBase := strings.TrimSuffix(outdatedVmssName, versionStr)
	return fmt.Sprintf("%s%s", vmssNameBase, newVersionStr)
}

type VMSSStateVerbose struct {
	VmssCreated     bool        `json:"vmss_created"`
	VmssName        string      `json:"vmss_name"`
	RefreshStatus   string      `json:"refresh_status"`
	RefreshVmssName string      `json:"refresh_vmss_name"`
	CurrentConfig   *VMSSConfig `json:"current_config,omitempty"`
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
