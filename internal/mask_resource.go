package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/raito-io/sdk"
	raitoType "github.com/raito-io/sdk/types"
	"github.com/raito-io/sdk/types/models"

	"github.com/raito-io/terraform-provider-raito/internal/types/abac_expression"
	"github.com/raito-io/terraform-provider-raito/internal/utils"
)

var _ resource.Resource = (*MaskResource)(nil)

type MaskResourceModel struct {
	// AccessProviderResourceModel properties. This has to be duplicated because of https://github.com/hashicorp/terraform-plugin-framework/issues/242
	Id          types.String         `tfsdk:"id"`
	Name        types.String         `tfsdk:"name"`
	Description types.String         `tfsdk:"description"`
	State       types.String         `tfsdk:"state"`
	Who         types.Set            `tfsdk:"who"`
	Owners      types.Set            `tfsdk:"owners"`
	WhoAbacRule jsontypes.Normalized `tfsdk:"who_abac_rule"`

	// MaskResourceModel properties.
	Type         types.String `tfsdk:"type"`
	DataSource   types.String `tfsdk:"data_source"`
	Columns      types.Set    `tfsdk:"columns"`
	WhatAbacRule types.Object `tfsdk:"what_abac_rule"`
}

func (m *MaskResourceModel) GetAccessProviderResourceModel() *AccessProviderResourceModel {
	return &AccessProviderResourceModel{
		Id:          m.Id,
		Name:        m.Name,
		Description: m.Description,
		State:       m.State,
		Who:         m.Who,
		Owners:      m.Owners,
		WhoAbacRule: m.WhoAbacRule,
	}
}

func (m *MaskResourceModel) SetAccessProviderResourceModel(ap *AccessProviderResourceModel) {
	m.Id = ap.Id
	m.Name = ap.Name
	m.Description = ap.Description
	m.State = ap.State
	m.Who = ap.Who
	m.Owners = ap.Owners
	m.WhoAbacRule = ap.WhoAbacRule
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
	} else if !m.WhatAbacRule.IsNull() {
		diagnostics.Append(m.abacWhatToAccessProviderInput(ctx, client, result)...)

		if diagnostics.HasError() {
			return diagnostics
		}
	}

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
		maskType, err := client.DataSource().GetMaskingMetadata(ctx, input.DataSources[0].Id)
		if err != nil {
			diagnostics.AddError("Failed to get default mask type", err.Error())

			return diagnostics
		}

		m.Type = types.StringPointerValue(maskType.DefaultMaskExternalName)
	} else {
		m.Type = types.StringPointerValue(input.Type)
	}

	if input.WhatType == raitoType.WhoAndWhatTypeDynamic && input.WhatAbacRule != nil {
		object, objectDiagnostics := m.abacWhatFromAccessProvider(ctx, client, input)
		diagnostics.Append(objectDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}

		m.WhatAbacRule = object
	}

	return diagnostics
}

func (m *MaskResourceModel) UpdateOwners(owners types.Set) {
	m.Owners = owners
}

func (m *MaskResourceModel) abacWhatToAccessProviderInput(ctx context.Context, client *sdk.RaitoClient, result *raitoType.AccessProviderInput) (diagnostics diag.Diagnostics) {
	attributes := m.WhatAbacRule.Attributes()

	scopeAttr := attributes["scope"]

	scope := make([]string, 0)

	if !scopeAttr.IsNull() && !scopeAttr.IsUnknown() {
		scopeFullnameItems, scopeDiagnostics := utils.StringSetToSlice(ctx, attributes["scope"].(types.Set))
		diagnostics.Append(scopeDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}

		for _, scopeFullnameItem := range scopeFullnameItems {
			id, err := client.DataObject().GetDataObjectIdByName(ctx, scopeFullnameItem, *result.DataSource)
			if err != nil {
				diagnostics.AddError("Failed to get data object id", err.Error())

				return diagnostics
			}

			scope = append(scope, id)
		}
	}

	jsonRule := attributes["rule"].(jsontypes.Normalized)

	var abacRule abac_expression.BinaryExpression
	diagnostics.Append(jsonRule.Unmarshal(&abacRule)...)

	if diagnostics.HasError() {
		return diagnostics
	}

	abacInput, err := abacRule.ToGqlInput()
	if err != nil {
		diagnostics.AddError("Failed to convert abac rule to gql input", err.Error())

		return diagnostics
	}

	result.WhatType = utils.Ptr(raitoType.WhoAndWhatTypeDynamic)
	result.WhatAbacRule = &raitoType.WhatAbacRuleInput{
		DoTypes: []string{"column"},
		Scope:   scope,
		Rule:    *abacInput,
	}

	return diagnostics
}

func (m *MaskResourceModel) abacWhatFromAccessProvider(ctx context.Context, client *sdk.RaitoClient, ap *raitoType.AccessProvider) (_ types.Object, diagnostics diag.Diagnostics) {
	objectTypes := map[string]attr.Type{
		"scope": types.SetType{ElemType: types.StringType},
		"rule":  jsontypes.NormalizedType{},
	}

	abacRule := jsontypes.NewNormalizedPointerValue(ap.WhatAbacRule.RuleJson)

	var scopeItems []attr.Value

	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	for scopeItem := range client.AccessProvider().GetAccessProviderAbacWhatScope(cancelCtx, ap.Id) {
		if scopeItem.HasError() {
			diagnostics.AddError("Failed to load access provider abac scope", scopeItem.GetError().Error())

			return types.ObjectNull(objectTypes), diagnostics
		}

		scopeItems = append(scopeItems, types.StringValue(scopeItem.MustGetItem().FullName))
	}

	scope, scopeDiagnostics := types.SetValue(types.StringType, scopeItems)
	diagnostics.Append(scopeDiagnostics...)

	if diagnostics.HasError() {
		return types.ObjectNull(objectTypes), diagnostics
	}

	object, whatAbacDiagnostics := types.ObjectValue(objectTypes, map[string]attr.Value{
		"rule":  abacRule,
		"scope": scope,
	})

	diagnostics.Append(whatAbacDiagnostics...)

	if diagnostics.HasError() {
		return types.ObjectNull(objectTypes), diagnostics
	}

	return object, diagnostics
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
		Computed:            true,
		Sensitive:           false,
		Description:         "The masking method",
		MarkdownDescription: "The masking method, which defines how the data is masked. Available types are defined by the data source.",
	}
	attributes["data_source"] = schema.StringAttribute{
		Required:            true,
		Optional:            false,
		Computed:            false,
		Sensitive:           false,
		Description:         "The ID of the data source of the mask",
		MarkdownDescription: "The ID of the data source of the mask",
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
		Description:         "The full name of columns that should be included in the mask",
		MarkdownDescription: "The full name of columns that should be included in the mask. Items are managed by Raito Cloud if columns is not set (nil).",
	}

	attributes["what_abac_rule"] = schema.SingleNestedAttribute{
		Attributes: map[string]schema.Attribute{
			"scope": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "Scope of the defined abac rule",
				MarkdownDescription: "Scope of the defined abac rule",
			},
			"rule": schema.StringAttribute{
				CustomType:          jsontypes.NormalizedType{},
				Required:            true,
				Optional:            false,
				Computed:            false,
				Sensitive:           false,
				Description:         "json representation of the abac rule",
				MarkdownDescription: "json representation of the abac rule",
				Default:             nil,
			},
		},
		Required:            false,
		Optional:            true,
		Computed:            false,
		Sensitive:           false,
		Description:         "What data object defined by abac rule. Cannot be set when what_data_objects is set.",
		MarkdownDescription: "What data object defined by abac rule. Cannot be set when what_data_objects is set.",
	}

	response.Schema = schema.Schema{
		Attributes:          attributes,
		Description:         "The mask access control resource",
		MarkdownDescription: "The mask access control resource",
		Version:             1,
	}
}

func readMaskResourceColumns(ctx context.Context, client *sdk.RaitoClient, data *MaskResourceModel) (diagnostics diag.Diagnostics) {
	if !data.Columns.IsNull() {
		cancelCtx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()

		whatItemsChannel := client.AccessProvider().GetAccessProviderWhatDataObjectList(cancelCtx, data.Id.ValueString())

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
