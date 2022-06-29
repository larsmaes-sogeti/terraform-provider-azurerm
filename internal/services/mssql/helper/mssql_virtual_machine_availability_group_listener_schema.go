package helper

import (
	"github.com/hashicorp/go-azure-sdk/resource-manager/sqlvirtualmachine/2022-02-01/availabilitygrouplisteners"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/validate"
	lbValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/loadbalancer/validate"
	sqlValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/mssql/validate"
	networkValidate "github.com/hashicorp/terraform-provider-azurerm/internal/services/network/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
)

type LoadBalancerConfigurationMsSqlVirtualMachineAvailabilityGroupListener struct {
	PrivateIpAddress           []PrivateIpAddressMsSqlVirtualMachineAvailabilityGroupListener `tfschema:"private_ip_address"`
	PublicIpAddressId          string                                                         `tfschema:"public_ip_address_id"`
	LoadBalancerId             string                                                         `tfschema:"load_balancer_id"`
	ProbePort                  int                                                            `tfschema:"probe_port"`
	SqlVirtualMachineInstances []string                                                       `tfschema:"sql_virtual_machine_instances"`
}

func LoadBalancerConfigurationSchemaMsSqlVirtualMachineAvailabilityGroupListener() *pluginsdk.Schema {
	return &pluginsdk.Schema{
		Type:     pluginsdk.TypeList,
		Required: true,
		ForceNew: true,
		MaxItems: 1,
		Elem: &pluginsdk.Resource{
			Schema: map[string]*pluginsdk.Schema{
				"private_ip_address": PrivateIpAddressSchemaMsSqlVirtualMachineAvailabilityGroupListener(),

				"public_ip_address_id": {
					Type:          pluginsdk.TypeString,
					Optional:      true,
					ForceNew:      true,
					ConflictsWith: []string{"load_balancer_configuration.0.private_ip_address"},
					ValidateFunc:  networkValidate.PublicIpAddressID,
				},

				"load_balancer_id": {
					Type:         pluginsdk.TypeString,
					Required:     true,
					ForceNew:     true,
					ValidateFunc: lbValidate.LoadBalancerID,
				},

				"probe_port": {
					Type:         pluginsdk.TypeInt,
					Optional:     true,
					ForceNew:     true,
					ValidateFunc: validate.PortNumber,
				},

				"sql_virtual_machine_instances": {
					Type:     pluginsdk.TypeList,
					Required: true,
					Elem: &pluginsdk.Schema{
						Type: pluginsdk.TypeString,
					},
				},
			},
		},
	}
}

type PrivateIpAddressMsSqlVirtualMachineAvailabilityGroupListener struct {
	IpAddress string `tfschema:"ip_address"`
	SubnetId  string `tfschema:"subnet_id"`
}

func PrivateIpAddressSchemaMsSqlVirtualMachineAvailabilityGroupListener() *pluginsdk.Schema {
	return &pluginsdk.Schema{
		Type:          pluginsdk.TypeList,
		Optional:      true,
		MinItems:      1,
		ConflictsWith: []string{"load_balancer_configuration.0.public_ip_address_id"},
		Elem: &pluginsdk.Resource{
			Schema: map[string]*pluginsdk.Schema{
				"ip_address": {
					Type:         pluginsdk.TypeString,
					Optional:     true,
					ValidateFunc: validation.IsIPAddress,
				},

				"subnet_id": {
					Type:         pluginsdk.TypeString,
					Optional:     true,
					ValidateFunc: networkValidate.SubnetID,
				},
			},
		},
	}
}

type ReplicaMsSqlVirtualMachineAvailabilityGroupListener struct {
	SqlVirtualMachineInstanceId string `tfschema:"sql_virtual_machine_instance_id"`
	Role                        string `tfschema:"role"`
	Commit                      string `tfschema:"commit"`
	Failover                    string `tfschema:"failover"`
	ReadableSecondary           string `tfschema:"readable_secondary"`
}

func ReplicaSchemaMsSqlVirtualMachineAvailabilityGroupListener() *pluginsdk.Schema {
	return &pluginsdk.Schema{
		Type:     pluginsdk.TypeList,
		Optional: true,
		Elem: &pluginsdk.Resource{
			Schema: map[string]*pluginsdk.Schema{
				"sql_virtual_machine_instance_id": {
					Type:         pluginsdk.TypeString,
					Optional:     true,
					ValidateFunc: sqlValidate.SqlVirtualMachineID,
				},

				"role": {
					Type:         pluginsdk.TypeString,
					Optional:     true,
					ValidateFunc: validation.StringInSlice([]string{string(availabilitygrouplisteners.RolePRIMARY), string(availabilitygrouplisteners.RoleSECONDARY)}, false),
				},

				"commit": {
					Type:         pluginsdk.TypeString,
					Optional:     true,
					ValidateFunc: validation.StringInSlice([]string{string(availabilitygrouplisteners.CommitSYNCHRONOUSCOMMIT), string(availabilitygrouplisteners.CommitASYNCHRONOUSCOMMIT)}, false),
				},

				"failover": {
					Type:         pluginsdk.TypeString,
					Optional:     true,
					ValidateFunc: validation.StringInSlice([]string{string(availabilitygrouplisteners.FailoverMANUAL), string(availabilitygrouplisteners.FailoverAUTOMATIC)}, false),
				},

				"readable_secondary": {
					Type:         pluginsdk.TypeString,
					Optional:     true,
					ValidateFunc: validation.StringInSlice([]string{string(availabilitygrouplisteners.ReadableSecondaryNO), string(availabilitygrouplisteners.ReadableSecondaryREADONLY), string(availabilitygrouplisteners.ReadableSecondaryALL)}, false),
				},
			},
		},
	}
}