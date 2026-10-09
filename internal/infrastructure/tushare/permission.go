package tushare

import "strings"

// APIGroup separates core, optional, and higher-threshold provider APIs.
type APIGroup string

const (
	APIGroupCore       APIGroup = "core"
	APIGroupOptional   APIGroup = "optional"
	APIGroupRestricted APIGroup = "restricted"
)

// PermissionStatus is the account-specific result, which remains unverified until a live test is authorized.
type PermissionStatus string

const PermissionNotTested PermissionStatus = "NOT_TESTED"

// APIRequirement records published minimum points separately from account-specific permission evidence.
type APIRequirement struct {
	APIName            string
	Group              APIGroup
	MinimumPoints      int
	MinimumPointsKnown bool
	PermissionStatus   PermissionStatus
}

var apiRequirements = map[string]APIRequirement{
	"stock_basic": {APIName: "stock_basic", Group: APIGroupCore, MinimumPoints: 2000, MinimumPointsKnown: true, PermissionStatus: PermissionNotTested},
	"trade_cal":   {APIName: "trade_cal", Group: APIGroupCore, MinimumPoints: 2000, MinimumPointsKnown: true, PermissionStatus: PermissionNotTested},
	"daily":       {APIName: "daily", Group: APIGroupCore, MinimumPoints: 120, MinimumPointsKnown: true, PermissionStatus: PermissionNotTested},
	"adj_factor":  {APIName: "adj_factor", Group: APIGroupCore, MinimumPoints: 2000, MinimumPointsKnown: true, PermissionStatus: PermissionNotTested},
	"suspend_d":   {APIName: "suspend_d", Group: APIGroupOptional, MinimumPoints: 2000, MinimumPointsKnown: true, PermissionStatus: PermissionNotTested},
	"stk_limit":   {APIName: "stk_limit", Group: APIGroupOptional, MinimumPoints: 2000, MinimumPointsKnown: true, PermissionStatus: PermissionNotTested},
	"namechange":  {APIName: "namechange", Group: APIGroupOptional, PermissionStatus: PermissionNotTested},
	"stock_st":    {APIName: "stock_st", Group: APIGroupRestricted, MinimumPoints: 3000, MinimumPointsKnown: true, PermissionStatus: PermissionNotTested},
}

// LookupAPIRequirement returns published API metadata without claiming the current token has access.
func LookupAPIRequirement(apiName string) (APIRequirement, bool) {
	requirement, ok := apiRequirements[strings.ToLower(strings.TrimSpace(apiName))]
	return requirement, ok
}
