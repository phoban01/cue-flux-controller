package v1alpha1

type PolicyRule string

var (
	PolicyRuleDeny   PolicyRule = "deny"
	PolicyRuleAudit  PolicyRule = "audit"
	PolicyRuleIgnore PolicyRule = "ignore"
)
