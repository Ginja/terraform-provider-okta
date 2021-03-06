package okta

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

var platformIncludeResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"type": &schema.Schema{
			Type:         schema.TypeString,
			Optional:     true,
			Default:      "ANY",
			ValidateFunc: validation.StringInSlice([]string{"ANY", "MOBILE", "DESKTOP"}, false),
		},
		"os_type": &schema.Schema{
			Type:         schema.TypeString,
			Optional:     true,
			Default:      "ANY",
			ValidateFunc: validation.StringInSlice([]string{"ANY", "IOS", "WINDOWS", "ANDROID", "OTHER", "OSX"}, false),
		},
		"os_expression": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Only available with OTHER OS type",
		},
	},
}

var userIdPatternResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"match_type": &schema.Schema{
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringInSlice([]string{"SUFFIX", "EQUALS", "STARTS_WITH", "CONTAINS", "EXPRESSION"}, false),
		},
		"value": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
		},
	},
}

func resourcePolicyRuleIdpDiscovery() *schema.Resource {
	return &schema.Resource{
		Exists:   resourcePolicyRuleIdpDiscoveryExists,
		Create:   resourcePolicyRuleIdpDiscoveryCreate,
		Read:     resourcePolicyRuleIdpDiscoveryRead,
		Update:   resourcePolicyRuleIdpDiscoveryUpdate,
		Delete:   resourcePolicyRuleIdpDiscoveryDelete,
		Importer: createPolicyRuleImporter(),

		Schema: buildBaseRuleSchema(map[string]*schema.Schema{
			"idp_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"idp_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "OKTA",
			},
			"app_include": &schema.Schema{
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Description: "Applications to include in discovery rule",
			},
			"app_exclude": &schema.Schema{
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Description: "Applications to exclude in discovery rule",
			},
			"platform_include": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     platformIncludeResource,
				Optional: true,
			},
			"user_identifier_type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"IDENTIFIER", "ATTRIBUTE", ""}, false),
			},
			"user_identifier_attribute": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"user_identifier_patterns": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     userIdPatternResource,
			},
		}),
	}
}

func buildPlatformInclude(d *schema.ResourceData) *IdpDiscoveryRulePlatform {
	includeList := []*IdpDiscoveryRulePlatformInclude{}

	if v, ok := d.GetOk("platform_include"); ok {
		valueList := v.(*schema.Set).List()

		for _, item := range valueList {
			if value, ok := item.(map[string]interface{}); ok {
				includeList = append(includeList, &IdpDiscoveryRulePlatformInclude{
					Os: &IdpDiscoveryRulePlatformOS{
						Expression: getMapString(value, "os_expression"),
						Type:       getMapString(value, "os_type"),
					},
					Type: getMapString(value, "type"),
				})
			}
		}
	}

	return &IdpDiscoveryRulePlatform{
		Include: includeList,
	}
}

func buildUserIdPatterns(d *schema.ResourceData) []*IdpDiscoveryRulePattern {
	var patternList []*IdpDiscoveryRulePattern

	if raw, ok := d.GetOk("user_identifier_patterns"); ok {
		patterns := raw.(*schema.Set).List()

		for _, pattern := range patterns {
			if value, ok := pattern.(map[string]interface{}); ok {
				patternList = append(patternList, &IdpDiscoveryRulePattern{
					MatchType: getMapString(value, "match_type"),
					Value:     getMapString(value, "value"),
				})
			}
		}
	}

	return patternList
}

func flattenUserIdPatterns(patterns []*IdpDiscoveryRulePattern) *schema.Set {
	var flattened []interface{}

	if patterns != nil {
		for _, v := range patterns {
			flattened = append(flattened, map[string]interface{}{
				"match_type": v.MatchType,
				"value":      v.Value,
			})
		}
	}

	return schema.NewSet(schema.HashResource(userIdPatternResource), flattened)
}

func flattenPlatformInclude(platform *IdpDiscoveryRulePlatform) *schema.Set {
	var flattend []interface{}

	if platform != nil && platform.Include != nil {
		flattened := make([]interface{}, len(platform.Include))
		for i, v := range platform.Include {
			flattened[i] = map[string]interface{}{
				"os_expression": v.Os.Expression,
				"os_type":       v.Os.Type,
				"type":          v.Type,
			}
		}
	}
	return schema.NewSet(schema.HashResource(platformIncludeResource), flattend)
}

func resourcePolicyRuleIdpDiscoveryExists(d *schema.ResourceData, m interface{}) (bool, error) {
	client := getSupplementFromMetadata(m)
	rule, _, err := client.GetIdpDiscoveryRule(d.Get("policyid").(string), d.Id())

	return err == nil && rule.ID != "", err
}

func resourcePolicyRuleIdpDiscoveryCreate(d *schema.ResourceData, m interface{}) error {
	newRule := buildIdpDiscoveryRule(d, m)
	client := getSupplementFromMetadata(m)
	rule, resp, err := client.CreateIdpDiscoveryRule(d.Get("policyid").(string), *newRule, nil)
	if err != nil {
		return responseErr(resp, err)
	}

	d.SetId(rule.ID)
	setRuleStatus(d, m, rule.Status)

	return resourcePolicyRuleIdpDiscoveryRead(d, m)
}

func setRuleStatus(d *schema.ResourceData, m interface{}, status string) error {
	desiredStatus := d.Get("status").(string)

	if status != desiredStatus {
		client := getSupplementFromMetadata(m)
		if desiredStatus == "INACTIVE" {
			return responseErr(client.DeactivateRule(d.Get("policyid").(string), d.Id()))
		} else if desiredStatus == "ACTIVE" {
			return responseErr(client.ActivateRule(d.Get("policyid").(string), d.Id()))
		}
	}

	return nil
}

func resourcePolicyRuleIdpDiscoveryRead(d *schema.ResourceData, m interface{}) error {
	client := getSupplementFromMetadata(m)
	rule, resp, err := client.GetIdpDiscoveryRule(d.Get("policyid").(string), d.Id())
	if err != nil {
		return responseErr(resp, err)
	}

	d.Set("name", rule.Name)
	d.Set("status", rule.Status)
	d.Set("priority", rule.Priority)
	d.Set("user_identifier_attribute", rule.Conditions.UserIdentifier.Attribute)
	d.Set("user_identifier_type", rule.Conditions.UserIdentifier.Type)
	d.Set("network_connection", rule.Conditions.Network.Connection)

	return setNonPrimitives(d, map[string]interface{}{
		"network_includes":         convertStringArrToInterface(rule.Conditions.Network.Include),
		"network_excludes":         convertStringArrToInterface(rule.Conditions.Network.Exclude),
		"platform_include":         flattenPlatformInclude(rule.Conditions.Platform),
		"user_identifier_patterns": flattenUserIdPatterns(rule.Conditions.UserIdentifier.Patterns),
		"app_include":              convertStringSetToInterface(rule.Conditions.App.Include),
		"app_exclude":              convertStringSetToInterface(rule.Conditions.App.Exclude),
	})
}

func resourcePolicyRuleIdpDiscoveryUpdate(d *schema.ResourceData, m interface{}) error {
	newRule := buildIdpDiscoveryRule(d, m)
	client := getSupplementFromMetadata(m)
	rule, resp, err := client.UpdateIdpDiscoveryRule(d.Get("policyid").(string), d.Id(), *newRule, nil)
	if err != nil {
		return responseErr(resp, err)
	}

	setRuleStatus(d, m, rule.Status)

	return resourcePolicyRuleIdpDiscoveryRead(d, m)
}

func resourcePolicyRuleIdpDiscoveryDelete(d *schema.ResourceData, m interface{}) error {
	client := getSupplementFromMetadata(m)

	resp, err := client.DeleteIdpDiscoveryRule(d.Get("policyid").(string), d.Id())
	return suppressErrorOn404(resp, err)
}

// Build Policy Sign On Rule from resource data
func buildIdpDiscoveryRule(d *schema.ResourceData, m interface{}) *IdpDiscoveryRule {
	rule := &IdpDiscoveryRule{
		Actions: &IdpDiscoveryRuleActions{
			IDP: &IdpDiscoveryRuleIdp{
				Providers: []*IdpDiscoveryRuleProvider{
					{
						Type: d.Get("idp_type").(string),
						ID:   d.Get("idp_id").(string),
					},
				},
			},
		},
		Conditions: &IdpDiscoveryRuleConditions{
			App: &IdpDiscoveryRuleApp{
				Include: convertInterfaceToStringSetNullable(d.Get("app_include")),
				Exclude: convertInterfaceToStringSetNullable(d.Get("app_exclude")),
			},
			Network: &IdpDiscoveryRuleNetwork{
				Connection: d.Get("network_connection").(string),
				// plural name here is vestigial due to old policy rule resources
				Include: convertInterfaceToStringArr(d.Get("network_includes")),
				Exclude: convertInterfaceToStringArr(d.Get("network_excludes")),
			},
			Platform: buildPlatformInclude(d),
			UserIdentifier: &IdpDiscoveryRuleUserIdentifier{
				Attribute: d.Get("user_identifier_attribute").(string),
				Type:      d.Get("user_identifier_type").(string),
				Patterns:  buildUserIdPatterns(d),
			},
		},
		Type:   idpDiscovery,
		Name:   d.Get("name").(string),
		Status: d.Get("status").(string),
	}

	if priority, ok := d.GetOk("priority"); ok {
		rule.Priority = priority.(int)
	}

	return rule
}
