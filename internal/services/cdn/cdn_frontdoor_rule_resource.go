package cdn

import (
	"fmt"
	"time"

	cdn "github.com/Azure/azure-sdk-for-go/services/cdn/mgmt/2021-06-01/cdn"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	cdnfrontdoorruleactions "github.com/hashicorp/terraform-provider-azurerm/internal/services/cdn/frontdoorruleactions"
	cdnfrontdoorruleconditions "github.com/hashicorp/terraform-provider-azurerm/internal/services/cdn/frontdoorruleconditions"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/cdn/parse"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/cdn/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

// TODO: all of the validation functions should be in `./validation`
// WS: Fixed
// TODO: the discriminators (e.g. expandFrontdoorDeliveryRuleActions) should be expanded using the standard way in the Azure SDK (e.g. `&SomeDescriminatedValue{}`)
// WS: I believe there is an issue with the swagger, but I have worked around the issue.
// TODO: several of the schema items expect a single item but are missing MaxItems:1
// WS: many of the condition and actions you can have more than one of in the same collection, but
// I have limited the ones that I know for a fact can only have a single item
// TODO: this needs a split/delta update method
// WS: Fixed
// TODO: all fields within nested block must be set into the state (e.g. we can't do `if model.Foo != nil { out["foo"] = foo }`
// WS: Fixed

func resourceCdnFrontdoorRule() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Create: resourceCdnFrontdoorRuleCreate,
		Read:   resourceCdnFrontdoorRuleRead,
		Update: resourceCdnFrontdoorRuleUpdate,
		Delete: resourceCdnFrontdoorRuleDelete,

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(30 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(30 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(30 * time.Minute),
		},

		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := parse.FrontdoorRuleID(id)
			return err
		}),

		Schema: map[string]*pluginsdk.Schema{

			"name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.CdnFrontdoorRuleName,
			},

			"cdn_frontdoor_rule_set_id": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.FrontdoorRuleSetID,
			},

			"behavior_on_match": {
				Type:     pluginsdk.TypeString,
				Optional: true,
				Default:  string(cdn.MatchProcessingBehaviorContinue),
				ValidateFunc: validation.StringInSlice([]string{
					string(cdn.MatchProcessingBehaviorContinue),
					string(cdn.MatchProcessingBehaviorStop),
				}, false),
			},

			"order": {
				Type:         pluginsdk.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntAtLeast(0),
			},

			"actions": {
				Type:     pluginsdk.TypeList,
				Required: true,
				MaxItems: 1,

				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{

						// Name: UrlRedirect
						// DeliveryRuleUrlRedirectActionParameters
						"url_redirect_action": {
							Type:     pluginsdk.TypeList,
							Optional: true,
							MaxItems: 1,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{

									"redirect_type": {
										Type:     pluginsdk.TypeString,
										Required: true,
										ValidateFunc: validation.StringInSlice([]string{
											string(cdn.RedirectTypeMoved),
											string(cdn.RedirectTypeFound),
											string(cdn.RedirectTypeTemporaryRedirect),
											string(cdn.RedirectTypePermanentRedirect),
										}, false),
									},

									"redirect_protocol": {
										Type:     pluginsdk.TypeString,
										Optional: true,
										Default:  string(cdn.DestinationProtocolMatchRequest),
										ValidateFunc: validation.StringInSlice([]string{
											string(cdn.DestinationProtocolMatchRequest),
											string(cdn.DestinationProtocolHTTP),
											string(cdn.DestinationProtocolHTTPS),
										}, false),
									},

									"destination_path": {
										Type:         pluginsdk.TypeString,
										Optional:     true,
										ValidateFunc: validate.CdnFrontdoorUrlRedirectActionDestinationPath,
									},

									"destination_hostname": {
										Type:         pluginsdk.TypeString,
										Required:     true,
										ValidateFunc: validation.StringIsNotEmpty,
									},

									"query_string": {
										Type:         pluginsdk.TypeString,
										Optional:     true,
										Default:      "",
										ValidateFunc: validate.CdnFrontdoorUrlRedirectActionQueryString,
									},

									"destination_fragment": {
										Type:         pluginsdk.TypeString,
										Optional:     true,
										ValidateFunc: validation.StringIsNotEmpty,
									},
								},
							},
						},

						// Name: URLRewrite
						// URLRewriteAction
						"url_rewrite_action": {
							Type:     pluginsdk.TypeList,
							Optional: true,
							MaxItems: 1,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{

									"source_pattern": {
										Type:         pluginsdk.TypeString,
										Required:     true,
										ValidateFunc: validation.StringIsNotEmpty,
									},

									"destination": {
										Type:         pluginsdk.TypeString,
										Required:     true,
										ValidateFunc: validation.StringIsNotEmpty,
									},

									"preserve_unmatched_path": {
										Type:     pluginsdk.TypeBool,
										Optional: true,
										Default:  false,
									},
								},
							},
						},

						// Name: ModifyRequestHeader
						// DeliveryRuleRequestHeaderAction
						"request_header_action": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{

									"header_action": {
										Type:     pluginsdk.TypeString,
										Required: true,
										ValidateFunc: validation.StringInSlice([]string{
											string(cdn.HeaderActionAppend),
											string(cdn.HeaderActionOverwrite),
											string(cdn.HeaderActionDelete),
										}, false),
									},

									"header_name": {
										Type:         pluginsdk.TypeString,
										Required:     true,
										ValidateFunc: validation.StringIsNotEmpty,
									},

									"value": {
										Type:         pluginsdk.TypeString,
										Optional:     true,
										ValidateFunc: validation.StringIsNotEmpty,
									},
								},
							},
						},

						// Name: ModifyResponseHeader (NameBasicDeliveryRuleActionNameModifyResponseHeader)
						// DeliveryRuleResponseHeaderAction
						"response_header_action": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{

									"header_action": {
										Type:     pluginsdk.TypeString,
										Required: true,
										ValidateFunc: validation.StringInSlice([]string{
											string(cdn.HeaderActionAppend),
											string(cdn.HeaderActionOverwrite),
											string(cdn.HeaderActionDelete),
										}, false),
									},

									"header_name": {
										Type:         pluginsdk.TypeString,
										Required:     true,
										ValidateFunc: validation.StringIsNotEmpty,
									},

									"value": {
										Type:         pluginsdk.TypeString,
										Optional:     true,
										ValidateFunc: validation.StringIsNotEmpty,
									},
								},
							},
						},

						// 'Cache_Expiration’, ‘Cache_Key_Query_String’ and ‘Origin_Group_Override' Actions
						// are only supported in the 2020-09-01 API version. All calls for these Actions
						// will fail 90 days after GA of the AFDx service.

						// Name: RouteConfigurationOverride (NameBasicDeliveryRuleActionNameRouteConfigurationOverride)
						// DeliveryRuleRouteConfigurationOverrideAction
						"route_configuration_override_action": {
							Type:     pluginsdk.TypeList,
							Optional: true,
							MaxItems: 1,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{

									"cdn_frontdoor_origin_group_id": {
										Type:         pluginsdk.TypeString,
										Required:     true,
										ValidateFunc: validate.FrontdoorOriginGroupID,
									},

									"forwarding_protocol": {
										Type:     pluginsdk.TypeString,
										Optional: true,
										Default:  string(cdn.ForwardingProtocolMatchRequest),
										ValidateFunc: validation.StringInSlice([]string{
											string(cdn.ForwardingProtocolHTTPOnly),
											string(cdn.ForwardingProtocolHTTPSOnly),
											string(cdn.ForwardingProtocolMatchRequest),
										}, false),
									},

									"query_string_caching_behavior": {
										Type:     pluginsdk.TypeString,
										Optional: true,
										Default:  string(cdn.RuleQueryStringCachingBehaviorIgnoreQueryString),
										ValidateFunc: validation.StringInSlice([]string{
											string(cdn.RuleQueryStringCachingBehaviorIgnoreQueryString),
											string(cdn.RuleQueryStringCachingBehaviorUseQueryString),
											string(cdn.RuleQueryStringCachingBehaviorIgnoreSpecifiedQueryStrings),
											string(cdn.RuleQueryStringCachingBehaviorIncludeSpecifiedQueryStrings),
										}, false),
									},

									// CSV implemented as a list, code alread written for the expaned and flatten to CSV
									// not valid when IncludeAll or ExcludeAll behavior is defined
									"query_string_parameters": {
										Type:     pluginsdk.TypeList,
										Optional: true,
										MaxItems: 100,

										Elem: &pluginsdk.Schema{
											Type: pluginsdk.TypeString,
										},
									},

									// Content won't be compressed on AzureFrontDoor when requested content is smaller than 1 byte or larger than 1 MB.
									"compression_enabled": {
										Type:     pluginsdk.TypeBool,
										Optional: true,
										Default:  false,
									},

									"cache_behavior": {
										Type:     pluginsdk.TypeString,
										Optional: true,
										Default:  string(cdn.RuleCacheBehaviorHonorOrigin),
										ValidateFunc: validation.StringInSlice([]string{
											string(cdn.RuleCacheBehaviorHonorOrigin),
											string(cdn.RuleCacheBehaviorOverrideAlways),
											string(cdn.RuleCacheBehaviorOverrideIfOriginMissing),
										}, false),
									},

									// Allowed format is d.HH:MM:SS or HH:MM:SS if duration is less than a day
									// maximum duration is 366 days(e.g. 365.23:59:59)
									"cache_duration": {
										Type:         pluginsdk.TypeString,
										Required:     true,
										ValidateFunc: validate.CdnFrontdoorCacheDuration,
									},
								},
							},
						},
					},
				},
			},

			"conditions": {
				Type:     pluginsdk.TypeList,
				Optional: true,
				MaxItems: 1,

				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{

						"remote_address_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"operator":         schemaCdnFrontdoorOperatorRemoteAddress(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorMatchValues(),
								},
							},
						},

						"request_method_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"operator":         schemaCdnFrontdoorOperatorEqualOnly(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorRequestMethodMatchValues(),
								},
							},
						},

						"query_string_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"operator":         schemaCdnFrontdoorOperator(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorMatchValues(),
									"transforms":       schemaCdnFrontdoorRuleTransforms(),
								},
							},
						},

						"post_args_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{

									// In the API this is called selector
									"post_args_name": {
										Type:         pluginsdk.TypeString,
										Required:     true,
										ValidateFunc: validation.StringIsNotEmpty,
									},
									"operator":         schemaCdnFrontdoorOperator(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorMatchValues(),
									"transforms":       schemaCdnFrontdoorRuleTransforms(),
								},
							},
						},

						"request_uri_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"operator":         schemaCdnFrontdoorOperator(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorMatchValues(),
									"transforms":       schemaCdnFrontdoorRuleTransforms(),
								},
							},
						},

						"request_header_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{

									// In the API this is called selector
									// match_values are invalid if operator is 'Any'
									"header_name": {
										Type:         pluginsdk.TypeString,
										Required:     true,
										ValidateFunc: validation.StringIsNotEmpty,
									},
									"operator":         schemaCdnFrontdoorOperator(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorMatchValues(),
									"transforms":       schemaCdnFrontdoorRuleTransforms(),
								},
							},
						},

						"request_body_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"operator":         schemaCdnFrontdoorOperator(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorMatchValuesRequired(),
									"transforms":       schemaCdnFrontdoorRuleTransforms(),
								},
							},
						},

						"request_scheme_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"operator":         schemaCdnFrontdoorOperatorEqualOnly(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorProtocolMatchValues(),
								},
							},
						},

						"url_path_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"operator":         schemaCdnFrontdoorOperator(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorUrlPathConditionMatchValues(),
									"transforms":       schemaCdnFrontdoorRuleTransforms(),
								},
							},
						},

						"url_file_extension_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"operator":         schemaCdnFrontdoorOperator(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorMatchValuesRequired(),
									"transforms":       schemaCdnFrontdoorRuleTransforms(),
								},
							},
						},

						"url_filename_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"operator":         schemaCdnFrontdoorOperator(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorMatchValuesRequired(),
									"transforms":       schemaCdnFrontdoorRuleTransforms(),
								},
							},
						},

						"http_version_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"operator":         schemaCdnFrontdoorOperatorEqualOnly(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorHttpVersionMatchValues(),
								},
							},
						},

						"cookies_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{

									// In the API this is called selector
									"cookie_name": {
										Type:         pluginsdk.TypeString,
										Required:     true,
										ValidateFunc: validation.StringIsNotEmpty,
									},

									"operator":         schemaCdnFrontdoorOperator(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorMatchValues(),
									"transforms":       schemaCdnFrontdoorRuleTransforms(),
								},
							},
						},

						"is_device_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"operator":         schemaCdnFrontdoorOperatorEqualOnly(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorIsDeviceMatchValues(),
								},
							},
						},

						// DeliveryRuleSocketAddrCondition
						"socket_address_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"operator":         schemaCdnFrontdoorOperatorSocketAddress(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorMatchValues(),
								},
							},
						},

						// DeliveryRuleClientPortCondition
						"client_port_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"operator":         schemaCdnFrontdoorOperator(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorMatchValues(),
								},
							},
						},

						// DeliveryRuleServerPortCondition
						"server_port_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"operator":         schemaCdnFrontdoorOperator(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorServerPortMatchValues(),
								},
							},
						},

						// DeliveryRuleHostNameCondition
						// NOTE: The match values do not have to adhere to RFC 1123 standards
						// for valid hostnames. Service team delegates that responsibility
						// to the author of the rule and does not validate passed match values.
						"host_name_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"operator":         schemaCdnFrontdoorOperator(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorMatchValues(),
									"transforms":       schemaCdnFrontdoorRuleTransforms(),
								},
							},
						},

						// DeliveryRuleSslProtocolCondition
						"ssl_protocol_condition": {
							Type:     pluginsdk.TypeList,
							Optional: true,

							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"operator":         schemaCdnFrontdoorOperatorEqualOnly(),
									"negate_condition": schemaCdnFrontdoorNegateCondition(),
									"match_values":     schemaCdnFrontdoorSslProtocolMatchValues(),
								},
							},
						},
					},
				},
			},

			"cdn_frontdoor_rule_set_name": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceCdnFrontdoorRuleCreate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cdn.FrontDoorRulesClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	ruleSetId, err := parse.FrontdoorRuleSetID(d.Get("cdn_frontdoor_rule_set_id").(string))
	if err != nil {
		return err
	}

	id := parse.NewFrontdoorRuleID(ruleSetId.SubscriptionId, ruleSetId.ResourceGroup, ruleSetId.ProfileName, ruleSetId.RuleSetName, d.Get("name").(string))

	if d.IsNewResource() {
		existing, err := client.Get(ctx, id.ResourceGroup, id.ProfileName, id.RuleSetName, id.RuleName)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("checking for existing %s: %+v", id, err)
			}
		}

		if !utils.ResponseWasNotFound(existing.Response) {
			return tf.ImportAsExistsError("azurerm_cdn_frontdoor_rule", id.ID())
		}
	}

	matchProcessingBehaviorValue := cdn.MatchProcessingBehavior(d.Get("behavior_on_match").(string))
	order := d.Get("order").(int)

	actions, err := expandFrontdoorDeliveryRuleActions(d.Get("actions").([]interface{}))
	if err != nil {
		return fmt.Errorf("expanding %q: %+v", "actions", err)
	}

	conditions, err := expandFrontdoorDeliveryRuleConditions(d.Get("conditions").([]interface{}))
	if err != nil {
		return fmt.Errorf("expanding %q: %+v", "conditions", err)
	}

	props := cdn.Rule{
		RuleProperties: &cdn.RuleProperties{
			Actions:                 &actions,
			Conditions:              &conditions,
			MatchProcessingBehavior: matchProcessingBehaviorValue,
			RuleSetName:             &ruleSetId.RuleSetName,
			Order:                   utils.Int32(int32(order)),
		},
	}

	future, err := client.Create(ctx, id.ResourceGroup, id.ProfileName, id.RuleSetName, id.RuleName, props)
	if err != nil {
		return fmt.Errorf("creating %s: %+v", id, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("waiting for the creation of %s: %+v", id, err)
	}

	d.SetId(id.ID())

	return resourceCdnFrontdoorRuleRead(d, meta)
}

func resourceCdnFrontdoorRuleRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cdn.FrontDoorRulesClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.FrontdoorRuleID(d.Id())
	if err != nil {
		return err
	}

	ruleSetId := parse.NewFrontdoorRuleSetID(id.SubscriptionId, id.ResourceGroup, id.ProfileName, id.RuleSetName)

	resp, err := client.Get(ctx, id.ResourceGroup, id.ProfileName, id.RuleSetName, id.RuleName)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("retrieving %s: %+v", id, err)
	}

	d.Set("name", id.RuleName)
	d.Set("cdn_frontdoor_rule_set_id", ruleSetId.ID())

	if props := resp.RuleProperties; props != nil {
		d.Set("behavior_on_match", props.MatchProcessingBehavior)
		d.Set("order", props.Order)

		// BUG: RuleSetName is not being returned by the API
		d.Set("cdn_frontdoor_rule_set_name", ruleSetId.RuleSetName)

		actions, err := flattenFrontdoorDeliveryRuleActions(props.Actions)
		if err != nil {
			return fmt.Errorf("setting %q: %+v", "actions", err)
		}
		d.Set("actions", actions)

		conditions, err := flattenFrontdoorDeliveryRuleConditions(props.Conditions)
		if err != nil {
			return fmt.Errorf("setting %q: %+v", "conditions", err)
		}

		d.Set("conditions", conditions)
	}

	return nil
}

func resourceCdnFrontdoorRuleUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cdn.FrontDoorRulesClient
	ctx, cancel := timeouts.ForUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.FrontdoorRuleID(d.Id())
	if err != nil {
		return err
	}

	props := cdn.RuleUpdateParameters{
		RuleUpdatePropertiesParameters: &cdn.RuleUpdatePropertiesParameters{},
	}

	if d.HasChange("behavior_on_match") {
		matchProcessingBehaviorValue := cdn.MatchProcessingBehavior(d.Get("behavior_on_match").(string))
		props.RuleUpdatePropertiesParameters.MatchProcessingBehavior = matchProcessingBehaviorValue
	}

	if d.HasChange("behavior_on_match") {
		order := d.Get("order").(int)
		props.RuleUpdatePropertiesParameters.Order = utils.Int32(int32(order))
	}

	if d.HasChange("actions") {
		actions, err := expandFrontdoorDeliveryRuleActions(d.Get("actions").([]interface{}))
		if err != nil {
			return fmt.Errorf("expanding %q: %+v", "actions", err)
		}

		props.RuleUpdatePropertiesParameters.Actions = &actions
	}

	if d.HasChange("conditions") {
		conditions, err := expandFrontdoorDeliveryRuleConditions(d.Get("conditions").([]interface{}))
		if err != nil {
			return fmt.Errorf("expanding %q: %+v", "conditions", err)
		}

		if len(conditions) > 10 {
			return fmt.Errorf("expanding %q: configuration file exceeds the maximum of 10 match conditions, got %d", "conditions", len(conditions))
		}

		props.RuleUpdatePropertiesParameters.Conditions = &conditions
	}

	future, err := client.Update(ctx, id.ResourceGroup, id.ProfileName, id.RuleSetName, id.RuleName, props)
	if err != nil {
		return fmt.Errorf("updating %s: %+v", *id, err)
	}
	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("waiting for the update of %s: %+v", *id, err)
	}

	return resourceCdnFrontdoorRuleRead(d, meta)
}

func resourceCdnFrontdoorRuleDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cdn.FrontDoorRulesClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := parse.FrontdoorRuleID(d.Id())
	if err != nil {
		return err
	}

	future, err := client.Delete(ctx, id.ResourceGroup, id.ProfileName, id.RuleSetName, id.RuleName)
	if err != nil {
		return fmt.Errorf("deleting %s: %+v", *id, err)
	}

	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return fmt.Errorf("waiting for the deletion of %s: %+v", *id, err)
	}

	return nil
}

func expandFrontdoorDeliveryRuleActions(input []interface{}) ([]cdn.BasicDeliveryRuleAction, error) {
	results := make([]cdn.BasicDeliveryRuleAction, 0)
	if len(input) == 0 {
		return results, nil
	}

	type expandfunc func(input []interface{}) (*[]cdn.BasicDeliveryRuleAction, error)

	m := *cdnfrontdoorruleactions.InitializeCdnFrontdoorActionMappings()

	actions := map[string]expandfunc{
		m.RouteConfigurationOverride.ConfigName: cdnfrontdoorruleactions.ExpandCdnFrontdoorRouteConfigurationOverrideAction,
		m.RequestHeader.ConfigName:              cdnfrontdoorruleactions.ExpandCdnFrontdoorRequestHeaderAction,
		m.ResponseHeader.ConfigName:             cdnfrontdoorruleactions.ExpandCdnFrontdoorResponseHeaderAction,
		m.URLRedirect.ConfigName:                cdnfrontdoorruleactions.ExpandCdnFrontdoorUrlRedirectAction,
		m.URLRewrite.ConfigName:                 cdnfrontdoorruleactions.ExpandCdnFrontdoorUrlRewriteAction,
	}

	basicDeliveryRuleAction := input[0].(map[string]interface{})

	for actionName, expand := range actions {
		raw := basicDeliveryRuleAction[actionName].([]interface{})
		expanded, err := expand(raw)
		if err != nil {
			return nil, err
		}

		if expanded != nil {
			if actionName == m.URLRewrite.ConfigName && len(*expanded) > 1 {
				return nil, fmt.Errorf("the %q is only allow once in the %q match block, got %d", m.URLRewrite.ConfigName, "actions", len(*expanded))
			}

			if actionName == m.URLRedirect.ConfigName && len(*expanded) > 1 {
				return nil, fmt.Errorf("the %q is only allow once in the %q match block, got %d", m.URLRedirect.ConfigName, "actions", len(*expanded))
			}

			if actionName == m.RouteConfigurationOverride.ConfigName && len(*expanded) > 1 {
				return nil, fmt.Errorf("the %q is only allow once in the %q match block, got %d", m.RouteConfigurationOverride.ConfigName, "actions", len(*expanded))
			}

			results = append(results, *expanded...)
		}
	}

	if len(results) > 5 {
		return nil, fmt.Errorf("the %q match block may only contain upto 5 match actions, got %d", "actions", len(results))
	}

	// validate action block
	if err := validate.CdnFrontdoorActionsBlock(results); err != nil {
		return nil, err
	}

	return results, nil
}

func expandFrontdoorDeliveryRuleConditions(input []interface{}) ([]cdn.BasicDeliveryRuleCondition, error) {
	results := make([]cdn.BasicDeliveryRuleCondition, 0)
	if len(input) == 0 {
		return results, nil
	}

	type expandfunc func(input []interface{}) (*[]cdn.BasicDeliveryRuleCondition, error)
	m := cdnfrontdoorruleconditions.InitializeCdnFrontdoorConditionMappings()

	// TODO: Add validation for this error once I figure out what that validation should be
	// "BadRequest" Message="Parameters specified for RequestMethod condition are invalid.;
	// Property 'Rule.Conditions[2].Parameters.Selector' is required but it was not set"

	conditions := map[string]expandfunc{
		m.ClientPort.ConfigName:       cdnfrontdoorruleconditions.ExpandCdnFrontdoorClientPortCondition,
		m.Cookies.ConfigName:          cdnfrontdoorruleconditions.ExpandCdnFrontdoorCookiesCondition,
		m.HostName.ConfigName:         cdnfrontdoorruleconditions.ExpandCdnFrontdoorHostNameCondition,
		m.HttpVersion.ConfigName:      cdnfrontdoorruleconditions.ExpandCdnFrontdoorHttpVersionCondition,
		m.IsDevice.ConfigName:         cdnfrontdoorruleconditions.ExpandCdnFrontdoorIsDeviceCondition,
		m.PostArgs.ConfigName:         cdnfrontdoorruleconditions.ExpandCdnFrontdoorPostArgsCondition,
		m.QueryString.ConfigName:      cdnfrontdoorruleconditions.ExpandCdnFrontdoorQueryStringCondition,
		m.RemoteAddress.ConfigName:    cdnfrontdoorruleconditions.ExpandCdnFrontdoorRemoteAddressCondition,
		m.RequestBody.ConfigName:      cdnfrontdoorruleconditions.ExpandCdnFrontdoorRequestBodyCondition,
		m.RequestHeader.ConfigName:    cdnfrontdoorruleconditions.ExpandCdnFrontdoorRequestHeaderCondition,
		m.RequestMethod.ConfigName:    cdnfrontdoorruleconditions.ExpandCdnFrontdoorRequestMethodCondition,
		m.RequestScheme.ConfigName:    cdnfrontdoorruleconditions.ExpandCdnFrontdoorRequestSchemeCondition,
		m.RequestUri.ConfigName:       cdnfrontdoorruleconditions.ExpandCdnFrontdoorRequestUriCondition,
		m.ServerPort.ConfigName:       cdnfrontdoorruleconditions.ExpandCdnFrontdoorServerPortCondition,
		m.SocketAddress.ConfigName:    cdnfrontdoorruleconditions.ExpandCdnFrontdoorSocketAddressCondition,
		m.SslProtocol.ConfigName:      cdnfrontdoorruleconditions.ExpandCdnFrontdoorSslProtocolCondition,
		m.UrlFileExtension.ConfigName: cdnfrontdoorruleconditions.ExpandCdnFrontdoorUrlFileExtensionCondition,
		m.UrlFilename.ConfigName:      cdnfrontdoorruleconditions.ExpandCdnFrontdoorUrlFileNameCondition,
		m.UrlPath.ConfigName:          cdnfrontdoorruleconditions.ExpandCdnFrontdoorUrlPathCondition,
	}

	basicDeliveryRuleCondition := input[0].(map[string]interface{})

	for conditionName, expand := range conditions {
		raw := basicDeliveryRuleCondition[conditionName].([]interface{})
		if len(raw) > 0 {
			expanded, err := expand(raw)
			if err != nil {
				return nil, err
			}

			if expanded != nil {
				results = append(results, *expanded...)
			}
		}
	}

	if len(results) > 10 {
		return nil, fmt.Errorf("the %q match block may only contain upto 10 match conditions, got %d", "conditions", len(results))
	}

	return results, nil
}

func flattenFrontdoorDeliveryRuleConditions(input *[]cdn.BasicDeliveryRuleCondition) ([]interface{}, error) {
	results := make([]interface{}, 0)
	if input == nil {
		return results, nil
	}

	c := cdnfrontdoorruleconditions.InitializeCdnFrontdoorConditionMappings()

	clientPortCondition := make([]interface{}, 0)
	cookiesCondition := make([]interface{}, 0)
	hostNameCondition := make([]interface{}, 0)
	httpVersionCondition := make([]interface{}, 0)
	isDeviceCondition := make([]interface{}, 0)
	postArgsCondition := make([]interface{}, 0)
	queryStringCondition := make([]interface{}, 0)
	remoteAddressCondition := make([]interface{}, 0)
	requestBodyCondition := make([]interface{}, 0)
	requestHeaderCondition := make([]interface{}, 0)
	requestMethodCondition := make([]interface{}, 0)
	requestSchemeCondition := make([]interface{}, 0)
	requestURICondition := make([]interface{}, 0)
	serverPortCondition := make([]interface{}, 0)
	socketAddressCondition := make([]interface{}, 0)
	sslProtocolCondition := make([]interface{}, 0)
	urlFileExtensionCondition := make([]interface{}, 0)
	urlFilenameCondition := make([]interface{}, 0)
	urlPathCondition := make([]interface{}, 0)

	for _, BasicDeliveryRuleCondition := range *input {
		// Client Port
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRuleClientPortCondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorClientPortCondition(condition)
			if err != nil {
				return nil, err
			}

			clientPortCondition = append(clientPortCondition, flattened)
			continue
		}

		// Cookies
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRuleCookiesCondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorCookiesCondition(condition)
			if err != nil {
				return nil, err
			}

			cookiesCondition = append(cookiesCondition, flattened)
			continue
		}

		// Host Name
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRuleHostNameCondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorHostNameCondition(condition)
			if err != nil {
				return nil, err
			}

			hostNameCondition = append(hostNameCondition, flattened)
			continue
		}

		// HTTP Version
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRuleHTTPVersionCondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorHttpVersionCondition(condition)
			if err != nil {
				return nil, err
			}
			httpVersionCondition = append(httpVersionCondition, flattened)
			continue
		}

		// Is Device
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRuleIsDeviceCondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorIsDeviceCondition(condition)
			if err != nil {
				return nil, err
			}

			isDeviceCondition = append(isDeviceCondition, flattened)
			continue
		}

		// Post Args
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRulePostArgsCondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorPostArgsCondition(condition)
			if err != nil {
				return nil, err
			}

			postArgsCondition = append(postArgsCondition, flattened)
			continue
		}

		// Query String
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRuleQueryStringCondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorQueryStringCondition(condition)
			if err != nil {
				return nil, err
			}

			queryStringCondition = append(queryStringCondition, flattened)
			continue
		}

		// Remote Address
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRuleRemoteAddressCondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorRemoteAddressCondition(condition)
			if err != nil {
				return nil, err
			}

			remoteAddressCondition = append(remoteAddressCondition, flattened)
			continue
		}

		// Request Body
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRuleRequestBodyCondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorRequestBodyCondition(condition)
			if err != nil {
				return nil, err
			}

			requestBodyCondition = append(requestBodyCondition, flattened)
			continue
		}

		// Request Header
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRuleRequestHeaderCondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorRequestHeaderCondition(condition)
			if err != nil {
				return nil, err
			}

			requestHeaderCondition = append(requestHeaderCondition, flattened)
			continue
		}

		// Request Method
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRuleRequestMethodCondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorRequestMethodCondition(condition)
			if err != nil {
				return nil, err
			}

			requestMethodCondition = append(requestMethodCondition, flattened)
			continue
		}

		// Request Scheme
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRuleRequestSchemeCondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorRequestSchemeCondition(condition)
			if err != nil {
				return nil, err
			}

			requestSchemeCondition = append(requestSchemeCondition, flattened)
			continue
		}

		// Request URI
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRuleRequestURICondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorRequestUriCondition(condition)
			if err != nil {
				return nil, err
			}

			requestURICondition = append(requestURICondition, flattened)
			continue
		}

		// Server Port
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRuleServerPortCondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorServerPortCondition(condition)
			if err != nil {
				return nil, err
			}

			serverPortCondition = append(serverPortCondition, flattened)
			continue
		}

		// Socket Address
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRuleSocketAddrCondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorSocketAddressCondition(condition)
			if err != nil {
				return nil, err
			}

			socketAddressCondition = append(socketAddressCondition, flattened)
			continue
		}

		// Ssl Protocol
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRuleSslProtocolCondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorSslProtocolCondition(condition)
			if err != nil {
				return nil, err
			}

			sslProtocolCondition = append(sslProtocolCondition, flattened)
			continue
		}

		// URL File Extension
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRuleURLFileExtensionCondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorUrlFileExtensionCondition(condition)
			if err != nil {
				return nil, err
			}

			urlFileExtensionCondition = append(urlFileExtensionCondition, flattened)
			continue
		}

		// URL Filename
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRuleURLFileNameCondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorUrlFileNameCondition(condition)
			if err != nil {
				return nil, err
			}

			urlFilenameCondition = append(urlFilenameCondition, flattened)
			continue
		}

		// URL Path
		if condition, ok := BasicDeliveryRuleCondition.AsDeliveryRuleURLPathCondition(); ok {
			flattened, err := cdnfrontdoorruleconditions.FlattenFrontdoorUrlPathCondition(condition)
			if err != nil {
				return nil, err
			}

			urlPathCondition = append(urlPathCondition, flattened)
			continue
		}

		return nil, fmt.Errorf("unknown BasicDeliveryRuleCondition encountered")
	}

	return []interface{}{
		map[string]interface{}{
			c.ClientPort.ConfigName:       clientPortCondition,
			c.Cookies.ConfigName:          cookiesCondition,
			c.HostName.ConfigName:         hostNameCondition,
			c.HttpVersion.ConfigName:      httpVersionCondition,
			c.IsDevice.ConfigName:         isDeviceCondition,
			c.PostArgs.ConfigName:         postArgsCondition,
			c.QueryString.ConfigName:      queryStringCondition,
			c.RemoteAddress.ConfigName:    remoteAddressCondition,
			c.RequestBody.ConfigName:      requestBodyCondition,
			c.RequestHeader.ConfigName:    requestHeaderCondition,
			c.RequestMethod.ConfigName:    requestMethodCondition,
			c.RequestScheme.ConfigName:    requestSchemeCondition,
			c.RequestUri.ConfigName:       requestURICondition,
			c.ServerPort.ConfigName:       serverPortCondition,
			c.SocketAddress.ConfigName:    socketAddressCondition,
			c.SslProtocol.ConfigName:      sslProtocolCondition,
			c.UrlFileExtension.ConfigName: urlFileExtensionCondition,
			c.UrlFilename.ConfigName:      urlFilenameCondition,
			c.UrlPath.ConfigName:          urlPathCondition,
		},
	}, nil
}

func flattenFrontdoorDeliveryRuleActions(input *[]cdn.BasicDeliveryRuleAction) ([]interface{}, error) {
	results := make([]interface{}, 0)
	if input == nil {
		return results, nil
	}

	a := cdnfrontdoorruleactions.InitializeCdnFrontdoorActionMappings()

	requestHeaderActions := make([]interface{}, 0)
	responseHeaderActions := make([]interface{}, 0)
	routeConfigOverrideActions := make([]interface{}, 0)
	urlRedirectActions := make([]interface{}, 0)
	urlRewriteActions := make([]interface{}, 0)

	for _, item := range *input {
		// Route Configuration
		if action, ok := item.AsDeliveryRuleRouteConfigurationOverrideAction(); ok {
			flattened := cdnfrontdoorruleactions.FlattenCdnFrontdoorRouteConfigurationOverrideAction(*action)
			routeConfigOverrideActions = append(routeConfigOverrideActions, flattened)
			continue
		}

		// Request Header
		if action, ok := item.AsDeliveryRuleRequestHeaderAction(); ok {
			if action.Parameters == nil {
				return nil, fmt.Errorf("`parameters` was nil for Delivery Rule Request Header")
			}
			flattened := cdnfrontdoorruleactions.FlattenHeaderActionParameters(action.Parameters)
			requestHeaderActions = append(requestHeaderActions, flattened)
			continue
		}

		// Response Header
		if action, ok := item.AsDeliveryRuleResponseHeaderAction(); ok {
			if action.Parameters == nil {
				return nil, fmt.Errorf("`parameters` was nil for Delivery Rule Response Header")
			}
			flattened := cdnfrontdoorruleactions.FlattenHeaderActionParameters(action.Parameters)
			responseHeaderActions = append(responseHeaderActions, flattened)
			continue
		}

		// URL Redirect
		if action, ok := item.AsURLRedirectAction(); ok {
			flattened := cdnfrontdoorruleactions.FlattenCdnFrontdoorUrlRedirectAction(*action)
			urlRedirectActions = append(urlRedirectActions, flattened)
			continue
		}

		// URL Rewrite
		if action, ok := item.AsURLRewriteAction(); ok {
			flattened := cdnfrontdoorruleactions.FlattenCdnFrontdoorUrlRewriteAction(*action)
			urlRewriteActions = append(urlRewriteActions, flattened)
			continue
		}

		// this would be an unknown DeliveryRuleAction type, but since we're only flattening the above, we can ignore it
	}

	if len(requestHeaderActions) == 0 && len(responseHeaderActions) == 0 && len(routeConfigOverrideActions) == 0 && len(urlRedirectActions) == 0 && len(urlRewriteActions) == 0 {
		return []interface{}{}, nil
	}

	return []interface{}{
		map[string]interface{}{
			a.RequestHeader.ConfigName:              requestHeaderActions,
			a.ResponseHeader.ConfigName:             responseHeaderActions,
			a.RouteConfigurationOverride.ConfigName: routeConfigOverrideActions,
			a.URLRedirect.ConfigName:                urlRedirectActions,
			a.URLRewrite.ConfigName:                 urlRewriteActions,
		},
	}, nil
}