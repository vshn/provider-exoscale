package iamkeycontroller

import (
	exoscalesdk "github.com/exoscale/egoscale/v3"

	"k8s.io/utils/ptr"
)

const (
	IamRolePermissionsBypassGovernanceRetention = "bypass-governance-retention"
)

var (
	policyAllow = exoscalesdk.IAMServicePolicyRuleActionAllow
	policyDeny  = exoscalesdk.IAMServicePolicyRuleActionDeny
)

func createRole(keyName string, buckets []string) *exoscalesdk.CreateIAMRoleRequest {

	policyRules := exoscalesdk.IAMServicePolicyTypeRules

	// We specifically need to deny the listing of buckets, or the customer is able to see all of them
	rules := []exoscalesdk.IAMServicePolicyRule{
		{
			Action:     policyDeny,
			Expression: "operation in ['list-sos-buckets-usage', 'list-buckets']",
		},
	}
	// we must first add buckets to deny list and then add the allow rule, otherwise it will not work
	for _, bucket := range buckets {
		rules = append(rules, exoscalesdk.IAMServicePolicyRule{
			Action:     policyDeny,
			Expression: "resources.bucket != " + "'" + bucket + "'",
		})
	}
	rules = append(rules, exoscalesdk.IAMServicePolicyRule{
		Action:     policyAllow,
		Expression: "true",
	})

	iamRole := exoscalesdk.CreateIAMRoleRequest{
		Name:        keyName,
		Description: "IAM Role for SOS+IAM creation, it was autogenerated by provider-exoscale",
		Permissions: []string{
			IamRolePermissionsBypassGovernanceRetention,
		},
		Editable: ptr.To(true),
		Policy: &exoscalesdk.IAMPolicy{
			DefaultServiceStrategy: exoscalesdk.IAMPolicyDefaultServiceStrategyDeny,
			Services: map[string]exoscalesdk.IAMServicePolicy{
				"sos": {
					Type:  policyRules,
					Rules: rules,
				},
			},
		},
	}

	return &iamRole
}
