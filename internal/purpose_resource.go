package internal

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/raito-io/sdk"
	raitoType "github.com/raito-io/sdk/types"
	"github.com/raito-io/sdk/types/models"

	"github.com/raito-io/terraform-provider-raito/internal/utils"
)

var _ resource.Resource = (*PurposeResource)(nil)

type PurposeResourceModel struct {
	// AccessProviderResourceModel properties. This has to be duplicated because of https://github.com/hashicorp/terraform-plugin-framework/issues/242
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	State       types.String `tfsdk:"state"`
	Who         types.Set    `tfsdk:"who"`

	// PurposeResourceModel properties.
	Type types.String `tfsdk:"type"`
	What types.Set    `tfsdk:"what"`
}

func (p *PurposeResourceModel) GetAccessProviderResourceModel() *AccessProviderResourceModel {
	return &AccessProviderResourceModel{
		Id:          p.Id,
		Name:        p.Name,
		Description: p.Description,
		State:       p.State,
		Who:         p.Who,
	}
}

func (p *PurposeResourceModel) SetAccessProviderResourceModel(model *AccessProviderResourceModel) {
	p.Id = model.Id
	p.Name = model.Name
	p.Description = model.Description
	p.State = model.State
	p.Who = model.Who
}

func (p *PurposeResourceModel) ToAccessProviderInput(ctx context.Context, client *sdk.RaitoClient, result *raitoType.AccessProviderInput) diag.Diagnostics {
	diagnostics := p.GetAccessProviderResourceModel().ToAccessProviderInput(ctx, client, result)

	if diagnostics.HasError() {
		return diagnostics
	}

	result.Type = p.Type.ValueStringPointer()
	result.Action = utils.Ptr(models.AccessProviderActionPurpose)

	if !p.What.IsNull() && !p.What.IsUnknown() {
		elements := p.What.Elements()

		result.WhatAccessProviders = make([]raitoType.AccessProviderWhatInputAP, 0, len(elements))

		for _, whatApObject := range elements {
			whatApId := whatApObject.(types.String)

			result.WhatAccessProviders = append(result.WhatAccessProviders, raitoType.AccessProviderWhatInputAP{
				AccessProvider: whatApId.ValueString(),
			})
		}
	}

	return diagnostics
}

func (p *PurposeResourceModel) FromAccessProvider(_ context.Context, _ *sdk.RaitoClient, input *raitoType.AccessProvider) diag.Diagnostics {
	apResourceModel := p.GetAccessProviderResourceModel()
	diagnostics := apResourceModel.FromAccessProvider(input)

	if diagnostics.HasError() {
		return diagnostics
	}

	p.SetAccessProviderResourceModel(apResourceModel)
	p.Type = types.StringPointerValue(input.Type)

	return diagnostics
}

type PurposeResource struct {
	AccessProviderResource[PurposeResourceModel, *PurposeResourceModel]
}

func NewPurposeResource() resource.Resource {
	return &PurposeResource{
		AccessProviderResource[PurposeResourceModel, *PurposeResourceModel]{
			readHooks: []ReadHook[PurposeResourceModel, *PurposeResourceModel]{
				readPurposeWhatAccessProviders,
			},
		},
	}
}

func (p *PurposeResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_purpose"
}

func (p *PurposeResource) Schema(_ context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	attributes := p.schema("purpose")
	attributes["type"] = schema.StringAttribute{
		Required:            false,
		Optional:            true,
		Computed:            true,
		Sensitive:           false,
		Description:         "The type of the purpose",
		MarkdownDescription: "The type of the purpose",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	attributes["what"] = schema.SetAttribute{
		ElementType:         types.StringType,
		Required:            false,
		Optional:            true,
		Computed:            false,
		Sensitive:           false,
		Description:         "The other access controls that should get linked to this purpose",
		MarkdownDescription: "The other access controls that should get linked to this purpose. If the user doesn't own the requested access controls, an access request will be created for them.",
	}

	response.Schema = schema.Schema{
		Attributes:          attributes,
		Description:         "The purpose access control resource",
		MarkdownDescription: "The purpose access control resource",
		Version:             1,
	}
}

func readPurposeWhatAccessProviders(ctx context.Context, client *sdk.RaitoClient, data *PurposeResourceModel) (diagnostics diag.Diagnostics) {
	if !data.What.IsNull() {
		whatItemsChannel := client.AccessProvider().GetAccessProviderWhatAccessProviderList(ctx, data.Id.ValueString())

		stateWhatItems := make([]attr.Value, 0)

		for whatItem := range whatItemsChannel {
			if whatItem.HasError() {
				diagnostics.AddError("Failed to get what access providers", whatItem.GetError().Error())

				return diagnostics
			}

			what := whatItem.GetItem()

			stateWhatItems = append(stateWhatItems, types.StringValue(what.AccessProvider.Id))
		}

		whatAps, whatDiag := types.SetValue(types.StringType, stateWhatItems)

		diagnostics.Append(whatDiag...)

		if diagnostics.HasError() {
			return diagnostics
		}

		data.What = whatAps
	}

	return diagnostics
}
