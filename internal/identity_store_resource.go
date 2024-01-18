package internal

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/raito-io/sdk"
	raitoType "github.com/raito-io/sdk/types"
)

var _ resource.Resource = (*IdentityStoreResource)(nil)

type IdentityStoreResourceModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	//Locked      types.Bool   `tfsdk:"locked"` // TODO
	Master types.Bool `tfsdk:"master"`
}

func (m *IdentityStoreResourceModel) ToIdentityStoreInput() raitoType.IdentityStoreInput {
	return raitoType.IdentityStoreInput{
		Name:        m.Name.ValueStringPointer(),
		Description: m.Description.ValueStringPointer(),
		Master:      m.Master.ValueBoolPointer(),
	}
}

type IdentityStoreResource struct {
	client *sdk.RaitoClient
}

func NewIdentityStoreResource() resource.Resource {
	return &IdentityStoreResource{}
}

func (i *IdentityStoreResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_identitystore"
}

func (i *IdentityStoreResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "ID of the identity store",
				MarkdownDescription: "ID of the identity store",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Computed:            false,
				Sensitive:           false,
				Description:         "Name of the identity store",
				MarkdownDescription: "Name of the identity store",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(3)},
			},
			"description": schema.StringAttribute{
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "Description of the identity store",
				MarkdownDescription: "Description of the identity store",
				Default:             stringdefault.StaticString(""),
			},
			//"locked": schema.BoolAttribute{
			//	Required:            false,
			//	Optional:            true,
			//	Computed:            true,
			//	Sensitive:           false,
			//	Description:         "Lock the identity store",
			//	MarkdownDescription: "Lock the identity store",
			//	Default:             booldefault.StaticBool(false),
			//},
			"master": schema.BoolAttribute{
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "Is this the master identity store",
				MarkdownDescription: "Is this the master identity store",
				Default:             booldefault.StaticBool(false),
			},
		},
		Description:         "Identity store resource",
		MarkdownDescription: "Identity store resource",
		Version:             1,
	}
}

func (i *IdentityStoreResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data IdentityStoreResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Create identity store
	isResult, err := i.client.IdentityStore().CreateIdentityStore(ctx, data.ToIdentityStoreInput())
	if err != nil {
		response.Diagnostics.AddError("Failed to create identity store", err.Error())

		return
	}

	data.Id = types.StringValue(isResult.Id)
	response.State.Set(ctx, data)
}

func (i *IdentityStoreResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var stateData IdentityStoreResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &stateData)...)

	if response.Diagnostics.HasError() {
		return
	}

	is, err := i.client.IdentityStore().GetIdentityStore(ctx, stateData.Id.ValueString())
	if err != nil {
		notFoundErr := &raitoType.ErrNotFound{}
		if errors.As(err, &notFoundErr) {
			response.State.RemoveResource(ctx)
		} else {
			response.Diagnostics.AddError("Failed to read identity store", err.Error())
		}

		return
	}

	actualData := IdentityStoreResourceModel{
		Id:          types.StringValue(is.Id),
		Name:        types.StringValue(is.Name),
		Description: types.StringValue(is.Description),
		Master:      types.BoolValue(is.Master),
	}

	response.Diagnostics.Append(response.State.Set(ctx, actualData)...)
}

func (i *IdentityStoreResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data IdentityStoreResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	_, err := i.client.IdentityStore().UpdateIdentityStore(ctx, data.Id.ValueString(), data.ToIdentityStoreInput())
	if err != nil {
		response.Diagnostics.AddError("Failed to update identity store", err.Error())

		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, data)...)
}

func (i *IdentityStoreResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data IdentityStoreResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	err := i.client.IdentityStore().DeleteIdentityStore(ctx, data.Id.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Failed to delete identity store", err.Error())
	}

	response.State.RemoveResource(ctx)
}

func (i *IdentityStoreResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sdk.RaitoClient)

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

	i.client = client
}

func (i *IdentityStoreResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
