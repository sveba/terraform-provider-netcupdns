package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

type DnsRecord struct {
	ID          types.String `tfsdk:"id"`
	Domainname  types.String `tfsdk:"domainname"`
	Hostname    types.String `tfsdk:"hostname"`
	Type        types.String `tfsdk:"type"`
	Priority    types.String `tfsdk:"priority"`
	Destination types.String `tfsdk:"destination"`
}
