package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/raito-io/sdk"
	raitoType "github.com/raito-io/sdk/types"
	"github.com/raito-io/sdk/types/models"

	"github.com/raito-io/terraform-provider-raito/internal/utils"
)

var _ resource.Resource = (*FilterResource)(nil)

type FilterResourceModel struct {
	// AccessProviderResourceModel properties. This has to be duplicated because of https://github.com/hashicorp/terraform-plugin-framework/issues/242
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	State       types.String `tfsdk:"state"`
	Who         types.Set    `tfsdk:"who"`

	// FilterResourceModel properties
	DataSource   types.String `tfsdk:"data_source"`
	Table        types.String `tfsdk:"table"`
	FilterPolicy types.String `tfsdk:"filter_policy"`
}

func (f *FilterResourceModel) GetAccessProviderResourceModel() *AccessProviderResourceModel {
	return &AccessProviderResourceModel{
		Id:          f.Id,
		Name:        f.Name,
		Description: f.Description,
		State:       f.State,
		Who:         f.Who,
	}
}

func (f *FilterResourceModel) SetAccessProviderResourceModel(ap *AccessProviderResourceModel) {
	f.Id = ap.Id
	f.Name = ap.Name
	f.Description = ap.Description
	f.State = ap.State
	f.Who = ap.Who
}

func (f *FilterResourceModel) ToAccessProviderInput(ctx context.Context, client *sdk.RaitoClient, result *raitoType.AccessProviderInput) diag.Diagnostics {
	diagnostics := f.GetAccessProviderResourceModel().ToAccessProviderInput(ctx, client, result)

	if diagnostics.HasError() {
		return diagnostics
	}

	result.Action = utils.Ptr(models.AccessProviderActionFiltered)
	result.DataSource = f.DataSource.ValueStringPointer()
	result.PolicyRule = f.FilterPolicy.ValueStringPointer()

	if !f.Table.IsNull() && !f.Table.IsUnknown() {
		result.WhatDataObjects = []raitoType.AccessProviderWhatInputDO{
			{
				DataObjectByName: []raitoType.AccessProviderWhatDoByNameInput{
					{
						Fullname:   f.Table.ValueString(),
						Datasource: f.DataSource.ValueString(),
					},
				},
			},
		}
	}

	return diagnostics
}

func (f *FilterResourceModel) FromAccessProvider(ctx context.Context, client *sdk.RaitoClient, input *raitoType.AccessProvider) diag.Diagnostics {
	apResourceModel := f.GetAccessProviderResourceModel()
	diagnostics := apResourceModel.FromAccessProvider(input)

	if diagnostics.HasError() {
		return diagnostics
	}

	f.SetAccessProviderResourceModel(apResourceModel)

	if len(input.DataSources) != 1 {
		diagnostics.AddError("Failed to get data source", fmt.Sprintf("Expected exactly one data source, got: %d.", len(input.DataSources)))

		return diagnostics
	}

	f.DataSource = types.StringValue(input.DataSources[0].Id)
	f.FilterPolicy = types.StringPointerValue(input.PolicyRule)

	return diagnostics
}

type FilterResource struct {
	AccessProviderResource[FilterResourceModel, *FilterResourceModel]
}

func NewFilterResource() resource.Resource {
	return &FilterResource{
		AccessProviderResource: AccessProviderResource[FilterResourceModel, *FilterResourceModel]{
			readHooks: []ReadHook[FilterResourceModel, *FilterResourceModel]{
				readFilterResourceTable,
			},
		},
	}
}

func (f *FilterResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_filter"
}

func (f *FilterResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	attributes := f.schema("filter")
	attributes["data_source"] = schema.StringAttribute{
		Required:            true,
		Optional:            false,
		Computed:            false,
		Sensitive:           false,
		Description:         "The ID of the data source of the filter",
		MarkdownDescription: "The ID of the data source of the filter",
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(3),
		},
	}
	attributes["table"] = schema.StringAttribute{
		Required:            false,
		Optional:            true,
		Computed:            false,
		Sensitive:           false,
		Description:         "The full name of the table that should be filtered",
		MarkdownDescription: "The full name of the table that should be filtered",
	}
	attributes["filter_policy"] = schema.StringAttribute{
		Required:            true,
		Optional:            false,
		Computed:            false,
		Sensitive:           false,
		Description:         "The filter policy that defines how the data is filtered. The policy syntax is defined by the data source.",
		MarkdownDescription: "The filter policy that defines how the data is filtered. The policy syntax is defined by the data source.",
	}

	response.Schema = schema.Schema{
		Attributes:          attributes,
		Description:         "The filter access control resource",
		MarkdownDescription: "The filter access control resource",
		Version:             1,
	}
}

func readFilterResourceTable(ctx context.Context, client *sdk.RaitoClient, data *FilterResourceModel) (diagnostics diag.Diagnostics) {
	if !data.Table.IsNull() {
		cancelCtx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()

		whatItemChannel := client.AccessProvider().GetAccessProviderWhatDataObjectList(cancelCtx, data.Id.ValueString())

		first := true

		for whatItem := range whatItemChannel {
			if !first {
				diagnostics.AddError("Received mutliple tables. Expect exactly one", "Filter resource only supports one table")

				return diagnostics
			}

			first = false

			if whatItem.HasError() {
				diagnostics.AddError("Failed to get filter what data objects", whatItem.GetError().Error())

				return diagnostics
			}

			what := whatItem.GetItem()
			data.Table = types.StringValue(what.DataObject.FullName)
		}
	}

	return diagnostics
}