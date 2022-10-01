package provider

import (
	"context"

	"github.com/fatih/structs"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/svetob/terraform-provider-netcupdns/internal/client"
)

var (
	_ resource.Resource                = &dnsRecordDataSource{}
	_ resource.ResourceWithConfigure   = &dnsRecordDataSource{}
	_ resource.ResourceWithImportState = &dnsRecordDataSource{}
)

func NewDnsRecordDataSource() resource.Resource {
	return &dnsRecordDataSource{}
}

type dnsRecordDataSource struct {
	client *client.CCPClient
}

func (r *dnsRecordDataSource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_record"
}

func (r dnsRecordDataSource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Represents a DNS-Record. See [Netcup-API](https://ccp.netcup.net/run/webservice/servers/endpoint.php#Dnsrecord)",
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:        types.StringType,
				Required:    false,
				Optional:    false,
				Computed:    true,
				Description: "Unique ID of the record. Provided from Netcup-API",
			},
			"domainname": {
				Type:        types.StringType,
				Required:    true,
				Description: "Domainname of the record.",
			},
			"hostname": {
				Type:        types.StringType,
				Required:    true,
				Description: "Name of the record. Use '@' for root of domain.",
			},
			"type": {
				Type:        types.StringType,
				Required:    true,
				Description: "Type of Record like A or MX.",
			},
			"priority": {
				Type:        types.StringType,
				Required:    false,
				Optional:    true,
				Computed:    true,
				Description: "Required for MX records.",
			},
			"destination": {
				Type:        types.StringType,
				Required:    true,
				Description: "Target of the record.",
			},
		},
	}, nil
}

func (r *dnsRecordDataSource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*client.CCPClient)
}

// Create a new resource
func (r dnsRecordDataSource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider hasn't been configured before apply, likely because it depends on an unknown value from another resource. This leads to weird stuff happening, so we'd prefer if you didn't do that. Thanks!",
		)
		return
	}

	// Retrieve values from plan
	var plan DnsRecord
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var newDnsRecord = client.NewDnsRecord{
		Hostname:    plan.Hostname.Value,
		Type:        plan.Type.Value,
		Destination: plan.Destination.Value,
	}

	if !plan.Priority.Unknown && !plan.Priority.Null {
		newDnsRecord.Priority = plan.Priority.Value
	}

	tflog.Trace(ctx, "Create DNS Record", structs.Map(newDnsRecord))

	// Create new order
	dnsRecord, err := r.client.CreateDnsRecord(plan.Domainname.Value, newDnsRecord)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating dns record",
			"Could not create dns record, unexpected error: "+err.Error(),
		)
		return
	}

	var state = DnsRecord{
		ID:          types.String{Value: dnsRecord.Id},
		Domainname:  plan.Domainname,
		Hostname:    types.String{Value: dnsRecord.Hostname},
		Type:        types.String{Value: dnsRecord.Type},
		Priority:    types.String{Value: dnsRecord.Priority},
		Destination: types.String{Value: dnsRecord.Destination},
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information
func (r dnsRecordDataSource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state DnsRecord
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current value
	dnsRecord, err := r.client.GetDnsRecordById(state.Domainname.Value, state.ID.Value)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading record",
			"Could not read recordID "+state.ID.Value+": "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "Got DNS Record", structs.Map(dnsRecord))

	state.Hostname = types.String{Value: dnsRecord.Hostname}
	state.Type = types.String{Value: dnsRecord.Type}
	state.Priority = types.String{Value: dnsRecord.Priority}
	state.Destination = types.String{Value: dnsRecord.Destination}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update resource
func (r dnsRecordDataSource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DnsRecord
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state
	var state DnsRecord
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var newDnsRecord = client.DnsRecord{
		Id:          state.ID.Value,
		Hostname:    plan.Hostname.Value,
		Type:        plan.Type.Value,
		Destination: plan.Destination.Value,
	}

	if !plan.Priority.Unknown && !plan.Priority.Null {
		newDnsRecord.Priority = plan.Priority.Value
	}

	tflog.Trace(ctx, "Updating DNS Record", structs.Map(newDnsRecord))

	// Update order by calling API
	dnsRecord, err := r.client.UpdateDnsRecord(plan.Domainname.Value, newDnsRecord)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error update dnsRecord",
			"Could not update dnsRecordID "+state.ID.Value+": "+err.Error(),
		)
		return
	}

	// Map response body to resource schema attribute
	// Generate resource state struct
	var result = DnsRecord{
		ID:          types.String{Value: state.ID.Value},
		Domainname:  types.String{Value: plan.Domainname.Value},
		Hostname:    types.String{Value: dnsRecord.Hostname},
		Type:        types.String{Value: dnsRecord.Type},
		Priority:    types.String{Value: dnsRecord.Priority},
		Destination: types.String{Value: dnsRecord.Destination},
	}

	// Set state
	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete resource
func (r dnsRecordDataSource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Get current state
	var state DnsRecord
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var dnsRecord = client.DnsRecord{
		Id:          state.ID.Value,
		Hostname:    state.Hostname.Value,
		Type:        state.Type.Value,
		Priority:    state.Priority.Value,
		Destination: state.Destination.Value,
	}

	tflog.Trace(ctx, "Deleting DNS Record", structs.Map(dnsRecord))

	// Delete order by calling API
	err := r.client.DeleteDnsRecord(state.Domainname.Value, dnsRecord)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting record",
			"Could not delete recordID "+state.ID.Value+": "+err.Error(),
		)
		return
	}

	// Remove resource from state
	resp.State.RemoveResource(ctx)
}

// Import resource
func (r dnsRecordDataSource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Save the import identifier in the id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
