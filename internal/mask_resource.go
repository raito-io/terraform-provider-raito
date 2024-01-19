package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/raito-io/sdk"
	raitoType "github.com/raito-io/sdk/types"
	"github.com/raito-io/sdk/types/models"

	"github.com/raito-io/terraform-provider-raito/internal/utils"
)

var _ resource.Resource = (*MaskResource)(nil)

type MaskResourceModel struct {
	// AccessProviderResourceModel properties. This has to be duplicated because of https://github.com/hashicorp/terraform-plugin-framework/issues/242
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	State       types.String `tfsdk:"state"`
	Who         types.Set    `tfsdk:"who"`

	// MaskResourceModel properties.
	Type       types.String `tfsdk:"type"`
	DataSource types.String `tfsdk:"data_source"`
	Columns    types.Set    `tfsdk:"columns"`
}

func (m *MaskResourceModel) GetAccessProviderResourceModel() *AccessProviderResourceModel {
	return &AccessProviderResourceModel{
		Id:          m.Id,
		Name:        m.Name,
		Description: m.Description,
		State:       m.State,
		Who:         m.Who,
	}
}

func (m *MaskResourceModel) SetAccessProviderResourceModel(ap *AccessProviderResourceModel) {
	m.Id = ap.Id
	m.Name = ap.Name
	m.Description = ap.Description
	m.State = ap.State
	m.Who = ap.Who
}

func (m *MaskResourceModel) ToAccessProviderInput(ctx context.Context, client *sdk.RaitoClient, result *raitoType.AccessProviderInput) diag.Diagnostics {
	diagnostics := m.GetAccessProviderResourceModel().ToAccessProviderInput(ctx, client, result)

	if diagnostics.HasError() {
		return diagnostics
	}

	if !m.Type.IsUnknown() {
		result.Type = m.Type.ValueStringPointer()
	}

	result.DataSource = m.DataSource.ValueStringPointer()
	result.Action = utils.Ptr(models.AccessProviderActionMask)

	if !m.Columns.IsNull() && !m.Columns.IsUnknown() {
		elements := m.Columns.Elements()

		result.WhatDataObjects = make([]raitoType.AccessProviderWhatInputDO, 0, len(elements))

		for _, whatDataObject := range elements {
			columnName := whatDataObject.(types.String).ValueString()

			doId, err := client.DataObject().GetDataObjectIdByName(ctx, columnName, *result.DataSource)
			if err != nil {
				diagnostics.AddError("Failed to get data object id", err.Error())

				return diagnostics
			}

			result.WhatDataObjects = append(result.WhatDataObjects, raitoType.AccessProviderWhatInputDO{
				DataObjects: []*string{&doId},
			})
		}
	}

	tflog.Info(ctx, fmt.Sprintf("MaskResourceModel: %+v", result))

	return diagnostics
}

func (m *MaskResourceModel) FromAccessProvider(ctx context.Context, client *sdk.RaitoClient, input *raitoType.AccessProvider) diag.Diagnostics {
	apResourceModel := m.GetAccessProviderResourceModel()
	diagnostics := apResourceModel.FromAccessProvider(input)

	if diagnostics.HasError() {
		return diagnostics
	}

	m.SetAccessProviderResourceModel(apResourceModel)

	if len(input.DataSources) != 1 {
		diagnostics.AddError("Failed to get data source", fmt.Sprintf("Expected exactly one data source, got: %d.", len(input.DataSources)))

		return diagnostics
	}

	m.DataSource = types.StringValue(input.DataSources[0].Id)

	if input.Type == nil {
		maskType, err := client.DataSource().GetDefaultMask(ctx, input.DataSources[0].Id)
		if err != nil {
			diagnostics.AddError("Failed to get default mask type", err.Error())

			return diagnostics
		}

		m.Type = types.StringPointerValue(maskType)
	} else {
		m.Type = types.StringPointerValue(input.Type)
	}

	return diagnostics
}

type MaskResource struct {
	AccessProviderResource[MaskResourceModel, *MaskResourceModel]
}

func NewMaskResource() resource.Resource {
	return &MaskResource{
		AccessProviderResource: AccessProviderResource[MaskResourceModel, *MaskResourceModel]{
			readHooks: []ReadHook[MaskResourceModel, *MaskResourceModel]{
				readMaskResourceColumns,
			},
		},
	}
}

func (m *MaskResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_mask"
}

func (m *MaskResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	attributes := m.schema("mask")
	attributes["type"] = schema.StringAttribute{
		Required:            false,
		Optional:            true,
		Computed:            false,
		Sensitive:           false,
		Description:         "Type of the mask/masking method.",
		MarkdownDescription: "Type of the mask. This defines how the data is masked. Available types are defined by the data source.",
	}
	attributes["data_source"] = schema.StringAttribute{
		Required:            true,
		Optional:            false,
		Computed:            false,
		Sensitive:           false,
		Description:         "Data source ID of the mask",
		MarkdownDescription: "Data source ID of the mask",
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(3),
		},
	}
	attributes["columns"] = schema.SetAttribute{
		ElementType:         types.StringType,
		Required:            false,
		Optional:            true,
		Computed:            false,
		Sensitive:           false,
		Description:         "Full name of columns that should be included in the mask",
		MarkdownDescription: "Full name of columns that should be included in the mask. Items are managed by Raito Cloud if columns is not set (nil).",
	}
	//TODO abac rule

	response.Schema = schema.Schema{
		Attributes:          attributes,
		Description:         "Mask access control resource",
		MarkdownDescription: "Mask access control resource",
		Version:             1,
	}
}

func (m *MaskResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data MaskResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	m.update(ctx, &data, response)
}

func readMaskResourceColumns(ctx context.Context, client *sdk.RaitoClient, data *MaskResourceModel) (diagnostics diag.Diagnostics) {
	if !data.Columns.IsNull() {
		whatItemsChannel := client.AccessProvider().GetAccessProviderWhatDataObjectList(ctx, data.Id.ValueString())

		stateWhatItems := make([]attr.Value, 0)

		for whatItem := range whatItemsChannel {
			if whatItem.HasError() {
				diagnostics.AddError("Fauled to get what data objects", whatItem.GetError().Error())

				return diagnostics
			}

			what := whatItem.GetItem()

			if what.DataObject != nil {
				stateWhatItems = append(stateWhatItems, types.StringValue(what.DataObject.FullName))
			} else {
				diagnostics.AddError("Invalid what data object", "Received data object is nil")
			}
		}

		columnsObject, columnsDiag := types.SetValue(types.StringType, stateWhatItems)

		diagnostics.Append(columnsDiag...)

		if diagnostics.HasError() {
			return diagnostics
		}

		data.Columns = columnsObject
	}

	return diagnostics
}