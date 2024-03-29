package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/svetob/terraform-provider-netcupdns/internal/client"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ provider.Provider = &netcupCcpProvider{}
)

func New() provider.Provider {
	return &netcupCcpProvider{}
}

type netcupCcpProvider struct{}

func (p *netcupCcpProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "netcupdns"
}

// Schema
func (p *netcupCcpProvider) Schema(_ context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Netcup-DNS provider provides possibility to modify Netcup-DNS records",
		Attributes: map[string]schema.Attribute{
			"customer_number": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Netcup customer number. Alternative defined by env `NETCUP_CUSTOMER_NUMBER`",
			},
			"key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Netcup CCP API key. Alternative defined by env `NETCUP_API_KEY`",
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Netcup CCP API password. Alternative defined by env `NETCUP_API_PASSWORD`",
			},
		},
	}
}

// Provider schema struct
type providerData struct {
	CustomerNumber types.String `tfsdk:"customer_number"`
	Key            types.String `tfsdk:"key"`
	Password       types.String `tfsdk:"password"`
}

func (p *netcupCcpProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config providerData
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// User must provide a user to the provider
	var customerNumber string
	if config.CustomerNumber.IsUnknown() {
		// Cannot connect to client with an unknown value
		resp.Diagnostics.AddWarning(
			"Unable to create client",
			"Cannot use unknown value as customer number",
		)
		return
	}

	if config.CustomerNumber.IsNull() {
		customerNumber = os.Getenv("NETCUP_CUSTOMER_NUMBER")
	} else {
		customerNumber = config.CustomerNumber.ValueString()
	}

	if customerNumber == "" {
		resp.Diagnostics.AddError(
			"Unable to find customer number",
			"Customer number cannot be an empty string",
		)
		return
	}

	var ccpApiPassword string
	if config.Password.IsUnknown() {
		resp.Diagnostics.AddError(
			"Unable to create client",
			"Cannot use unknown value as api password",
		)
		return
	}

	if config.Password.IsNull() {
		ccpApiPassword = os.Getenv("NETCUP_API_PASSWORD")
	} else {
		ccpApiPassword = config.Password.ValueString()
	}

	if ccpApiPassword == "" {
		resp.Diagnostics.AddError(
			"Unable to find password",
			"Api Password cannot be an empty string",
		)
		return
	}

	var ccpApiKey string
	if config.Key.IsUnknown() {
		resp.Diagnostics.AddError(
			"Unable to create client",
			"Cannot use unknown value as api key",
		)
		return
	}

	if config.Key.IsNull() {
		ccpApiKey = os.Getenv("NETCUP_API_KEY")
	} else {
		ccpApiKey = config.Key.ValueString()
	}

	if ccpApiKey == "" {
		resp.Diagnostics.AddError(
			"Unable to create client",
			"Api key cannot be an empty string",
		)
		return
	}

	c, err := client.NewCCPClient(customerNumber, ccpApiKey, ccpApiPassword)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create CCP client",
			"Unable to authenticate customer "+customerNumber+" with Netcup CCP API\n\n"+err.Error(),
		)
		return
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *netcupCcpProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDnsRecordDataSource,
	}
}

func (p *netcupCcpProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
