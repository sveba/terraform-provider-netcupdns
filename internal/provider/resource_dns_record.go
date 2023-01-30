package provider

import (
	"context"

	"github.com/fatih/structs"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

func (r *dnsRecordDataSource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Represents a DNS-Record. See [Netcup-API](https://ccp.netcup.net/run/webservice/servers/endpoint.php#Dnsrecord)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    false,
				Optional:    false,
				Computed:    true,
				Description: "Unique ID of the record. Provided from Netcup-API",
			},
			"domainname": schema.StringAttribute{
				Required:    true,
				Description: "Domainname of the record.",
			},
			"hostname": schema.StringAttribute{
				Required:    true,
				Description: "Name of the record. Use '@' for root of domain.",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Type of Record like A or MX.",
			},
			"priority": schema.StringAttribute{
				Required:    false,
				Optional:    true,
				Computed:    true,
				Description: "Required for MX records.",
			},
			"destination": schema.StringAttribute{
				Required:    true,
				Description: "Target of the record.",
			},
		},
	}
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
		Hostname:    plan.Hostname.ValueString(),
		Type:        plan.Type.ValueString(),
		Destination: plan.Destination.ValueString(),
	}

	if !plan.Priority.IsUnknown() && !plan.Priority.IsNull() {
		newDnsRecord.Priority = plan.Priority.ValueString()
	}

	tflog.Trace(ctx, "Create DNS Record", structs.Map(newDnsRecord))

	// Create new order
	dnsRecord, err := r.client.CreateDnsRecord(plan.Domainname.ValueString(), newDnsRecord)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating dns record",
			"Could not create dns record, unexpected error: "+err.Error(),
		)
		return
	}

	var state = DnsRecord{
		ID:          types.StringValue(dnsRecord.Id),
		Domainname:  plan.Domainname,
		Hostname:    types.StringValue(dnsRecord.Hostname),
		Type:        types.StringValue(dnsRecord.Type),
		Priority:    types.StringValue(dnsRecord.Priority),
		Destination: types.StringValue(dnsRecord.Destination),
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
	dnsRecord, err := r.client.GetDnsRecordById(state.Domainname.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading record",
			"Could not read recordID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "Got DNS Record", structs.Map(dnsRecord))

	state.Hostname = types.StringValue(dnsRecord.Hostname)
	state.Type = types.StringValue(dnsRecord.Type)
	state.Priority = types.StringValue(dnsRecord.Priority)
	state.Destination = types.StringValue(dnsRecord.Destination)

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
		Id:          state.ID.ValueString(),
		Hostname:    plan.Hostname.ValueString(),
		Type:        plan.Type.ValueString(),
		Destination: plan.Destination.ValueString(),
	}

	if !plan.Priority.IsUnknown() && !plan.Priority.IsNull() {
		newDnsRecord.Priority = plan.Priority.ValueString()
	}

	tflog.Trace(ctx, "Updating DNS Record", structs.Map(newDnsRecord))

	// Update order by calling API
	dnsRecord, err := r.client.UpdateDnsRecord(plan.Domainname.ValueString(), newDnsRecord)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error update dnsRecord",
			"Could not update dnsRecordID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to resource schema attribute
	// Generate resource state struct
	var result = DnsRecord{
		ID:          types.StringValue(state.ID.ValueString()),
		Domainname:  types.StringValue(plan.Domainname.ValueString()),
		Hostname:    types.StringValue(dnsRecord.Hostname),
		Type:        types.StringValue(dnsRecord.Type),
		Priority:    types.StringValue(dnsRecord.Priority),
		Destination: types.StringValue(dnsRecord.Destination),
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
		Id:          state.ID.ValueString(),
		Hostname:    state.Hostname.ValueString(),
		Type:        state.Type.ValueString(),
		Priority:    state.Priority.ValueString(),
		Destination: state.Destination.ValueString(),
	}

	tflog.Trace(ctx, "Deleting DNS Record", structs.Map(dnsRecord))

	// Delete order by calling API
	err := r.client.DeleteDnsRecord(state.Domainname.ValueString(), dnsRecord)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting record",
			"Could not delete recordID "+state.ID.ValueString()+": "+err.Error(),
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
