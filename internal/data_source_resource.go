package internal

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/raito-io/golang-set/set"
	raitoType "github.com/raito-io/sdk/types"

	"github.com/raito-io/terraform-provider-raito/internal/utils"
)

var _ resource.Resource = (*DataSourceResource)(nil)

type DataSourceResourceModel struct {
	Id                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	SyncMethod          types.String `tfsdk:"sync_method"`
	Parent              types.String `tfsdk:"parent"`
	NativeIdentityStore types.String `tfsdk:"native_identity_store"`
	IdentityStores      types.Set    `tfsdk:"identity_stores"`
}

func (m *DataSourceResourceModel) ToDataSourceInput() raitoType.DataSourceInput {
	return raitoType.DataSourceInput{
		Name:        m.Name.ValueStringPointer(),
		Description: m.Description.ValueStringPointer(),
		SyncMethod:  utils.Ptr(raitoType.DataSourceSyncMethod(m.SyncMethod.ValueString())),
		Parent:      m.Parent.ValueStringPointer(),
	}
}

type DataSourceResource struct {
	client DataSourceClient
}

func NewDataSourceResource() resource.Resource {
	return &DataSourceResource{}
}

func (d *DataSourceResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_datasource"
}

func (d *DataSourceResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "ID of the data source",
				MarkdownDescription: "ID of the data source",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Computed:            false,
				Sensitive:           false,
				Description:         "Name of the data source",
				MarkdownDescription: "Name of the data source",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(3)},
			},
			"description": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "Description of the data source",
				MarkdownDescription: "Description of the data source",
				Default:             stringdefault.StaticString(""),
			},
			"sync_method": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "Sync method of the data source",
				MarkdownDescription: "Sync method of the data source",
				Default:             stringdefault.StaticString(string(raitoType.DataSourceSyncMethodOnPrem)),
				Validators:          []validator.String{stringvalidator.OneOf(string(raitoType.DataSourceSyncMethodOnPrem), string(raitoType.DataSourceSyncMethodCloudManualTrigger))},
			},
			"parent": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				Computed:            false,
				Sensitive:           false,
				Description:         "Parent data source id if applicable",
				MarkdownDescription: "Parent data source id if applicable",
			},
			"native_identity_store": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "ID of the native identity store",
				MarkdownDescription: "ID of the native identity store",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"identity_stores": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "Linked identity stores",
				MarkdownDescription: "Linked identity stores",
				DeprecationMessage:  "",
				Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
		},
		Description:         "Data source resource",
		MarkdownDescription: "Data Source resource",
		Version:             1,
	}
}

func (d *DataSourceResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data DataSourceResourceModel

	// Read Terraform plan data into the model
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Create data source
	dataSourceResult, err := d.client.CreateDataSource(ctx, data.ToDataSourceInput())
	if err != nil {
		response.Diagnostics.AddError("Failed to create data source", err.Error())

		return
	}

	data.Id = types.StringValue(dataSourceResult.Id)
	response.State.Set(ctx, data) //Ensure to store id first

	// Load current identity stores
	identityStores, err := d.client.ListIdentityStores(ctx, dataSourceResult.Id)
	if err != nil {
		response.Diagnostics.AddError("Failed to list identity stores", err.Error())

		return
	}

	planExpectedIss := set.Set[string]{}
	for _, identityStoreId := range data.IdentityStores.Elements() {
		idValue := identityStoreId.(types.String)
		planExpectedIss.Add(idValue.ValueString())
	}

	linkedIss := set.Set[string]{}

	for _, identityStore := range identityStores {
		if identityStore.Native {
			data.NativeIdentityStore = types.StringValue(identityStore.Id)
		} else {
			linkedIss.Add(identityStore.Id)
		}
	}

	// Add missing identity stores
	for is := range planExpectedIss {
		if linkedIss.Contains(is) {
			continue
		}

		err = d.client.AddIdentityStoreToDataSource(ctx, dataSourceResult.Id, is)
		if err != nil {
			response.Diagnostics.AddError("Failed to remove identity store from data source", err.Error())

			return
		}
	}

	// Remove old identity stores
	for is := range linkedIss {
		if planExpectedIss.Contains(is) {
			continue
		}

		err = d.client.RemoveIdentityStoreFromDataSource(ctx, dataSourceResult.Id, is)
		if err != nil {
			response.Diagnostics.AddError("Failed to add identity store to data source", err.Error())

			return
		}
	}

	response.State.Set(ctx, data)
}

func (d *DataSourceResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var stateData DataSourceResourceModel

	// Read Terraform plan stateData into the model
	response.Diagnostics.Append(request.State.Get(ctx, &stateData)...)

	if response.Diagnostics.HasError() {
		return
	}

	ds, err := d.client.GetDataSource(ctx, stateData.Id.ValueString())
	if err != nil {
		var notFoundErr *raitoType.ErrNotFound
		if !errors.As(err, &notFoundErr) {
			response.State.RemoveResource(ctx)
		} else {
			response.Diagnostics.AddError("Failed to get data source", err.Error())
		}

		return
	}

	identityStores, err := d.client.ListIdentityStores(ctx, stateData.Id.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Failed to list identity stores", err.Error())

		return
	}

	var nativeIs *string
	isIds := make([]attr.Value, 0, len(identityStores))

	for _, identityStore := range identityStores {
		if identityStore.Native {
			nativeIs = &identityStore.Id
		} else if !identityStore.Master {
			isIds = append(isIds, types.StringValue(identityStore.Id))
		}
	}

	isAttr, diagnostic := types.SetValue(types.StringType, isIds)
	response.Diagnostics.Append(diagnostic...)

	var parentId *string
	if ds.Parent != nil {
		parentId = &ds.Parent.Id
	}

	actualData := DataSourceResourceModel{
		Id:                  types.StringValue(ds.Id),
		Name:                types.StringValue(ds.Name),
		Description:         types.StringValue(ds.Description),
		SyncMethod:          types.StringValue(string(ds.SyncMethod)),
		Parent:              types.StringPointerValue(parentId),
		NativeIdentityStore: types.StringPointerValue(nativeIs),
		IdentityStores:      isAttr,
	}

	response.State.Set(ctx, actualData)
}

func (d *DataSourceResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data DataSourceResourceModel

	// Read Terraform plan data into the model
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Update data source
	_, err := d.client.UpdateDataSource(ctx, data.Id.ValueString(), data.ToDataSourceInput())
	if err != nil {
		response.Diagnostics.AddError("Failed to update data source", err.Error())

		return
	}

	// Load current identity stores
	identityStores, err := d.client.ListIdentityStores(ctx, data.Id.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Failed to list identity stores", err.Error())

		return
	}

	planExpectedIss := set.Set[string]{}
	for _, identityStoreId := range data.IdentityStores.Elements() {
		idValue := identityStoreId.(types.String)
		planExpectedIss.Add(idValue.ValueString())
	}

	linkedIss := set.Set[string]{}

	for _, identityStore := range identityStores {
		if !identityStore.Native && !identityStore.Master {
			linkedIss.Add(identityStore.Id)
		}
	}

	// Add missing identity stores
	for is := range planExpectedIss {
		if linkedIss.Contains(is) {
			continue
		}

		err = d.client.AddIdentityStoreToDataSource(ctx, data.Id.ValueString(), is)
		if err != nil {
			response.Diagnostics.AddError("Failed to remove identity store from data source", err.Error())

			return
		}
	}

	// Remove old identity stores
	for is := range linkedIss {
		if planExpectedIss.Contains(is) {
			continue
		}

		err = d.client.RemoveIdentityStoreFromDataSource(ctx, data.Id.ValueString(), is)
		if err != nil {
			response.Diagnostics.AddError("Failed to add identity store to data source", err.Error())

			return
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, data)...)
}

func (d *DataSourceResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data DataSourceResourceModel

	// Read Terraform plan data into the model
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	err := d.client.DeleteDataSource(ctx, data.Id.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Failed to delete data source", err.Error())

		return
	}

	response.State.RemoveResource(ctx)
}

func (d *DataSourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(RaitoClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *sdk.RaitoClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	if client == nil {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected *sdk.RaitoClient, not to be nil.",
		)

		return
	}

	d.client = client.DataSource()
}

func (d *DataSourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
