package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/raito-io/sdk"
	raitoType "github.com/raito-io/sdk/types"
	"github.com/raito-io/sdk/types/models"

	types2 "github.com/raito-io/terraform-provider-raito/internal/types"
	"github.com/raito-io/terraform-provider-raito/internal/types/abac_expression"
	"github.com/raito-io/terraform-provider-raito/internal/utils"
)

var _ resource.Resource = (*GrantResource)(nil)

type GrantResourceModel struct {
	// AccessProviderResourceModel properties. This has to be duplicated because of https://github.com/hashicorp/terraform-plugin-framework/issues/242
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	State       types.String `tfsdk:"state"`
	Who         types.Set    `tfsdk:"who"`
	Owners      types.Set    `tfsdk:"owners"`

	// GrantResourceModel properties.
	Type            types.String `tfsdk:"type"`
	DataSource      types.String `tfsdk:"data_source"`
	WhatDataObjects types.Set    `tfsdk:"what_data_objects"`
	WhatAbacRule    types.Object `tfsdk:"what_abac_rule"`
}

func (m *GrantResourceModel) GetAccessProviderResourceModel() *AccessProviderResourceModel {
	return &AccessProviderResourceModel{
		Id:          m.Id,
		Name:        m.Name,
		Description: m.Description,
		State:       m.State,
		Who:         m.Who,
		Owners:      m.Owners,
	}
}

func (m *GrantResourceModel) SetAccessProviderResourceModel(ap *AccessProviderResourceModel) {
	m.Id = ap.Id
	m.Name = ap.Name
	m.Description = ap.Description
	m.State = ap.State
	m.Who = ap.Who
	m.Owners = ap.Owners
}

func (m *GrantResourceModel) ToAccessProviderInput(ctx context.Context, client *sdk.RaitoClient, result *raitoType.AccessProviderInput) diag.Diagnostics {
	diagnostics := m.GetAccessProviderResourceModel().ToAccessProviderInput(ctx, client, result)

	if diagnostics.HasError() {
		return diagnostics
	}

	if !m.Type.IsUnknown() {
		result.Type = m.Type.ValueStringPointer()
	}

	result.DataSource = m.DataSource.ValueStringPointer()
	result.Action = utils.Ptr(models.AccessProviderActionGrant)
	result.WhatType = utils.Ptr(raitoType.WhoAndWhatTypeStatic)

	if !m.WhatDataObjects.IsNull() && !m.WhatDataObjects.IsUnknown() {
		diagnostics.Append(m.whatDoToApInput(ctx, client, result)...)

		if diagnostics.HasError() {
			return diagnostics
		}
	} else if !m.WhatAbacRule.IsNull() {
		diagnostics.Append(m.whatAbacToApInput(ctx, client, result)...)

		if diagnostics.HasError() {
			return diagnostics
		}
	}

	return diagnostics
}

func (m *GrantResourceModel) whatDoToApInput(ctx context.Context, client *sdk.RaitoClient, result *raitoType.AccessProviderInput) (diagnostics diag.Diagnostics) {
	elements := m.WhatDataObjects.Elements()

	result.WhatDataObjects = make([]raitoType.AccessProviderWhatInputDO, 0, len(elements))

	for _, whatDataObject := range elements {
		whatDataObjectObject := whatDataObject.(types.Object)
		whatDataObjectAttributes := whatDataObjectObject.Attributes()

		fullname := whatDataObjectAttributes["fullname"].(types.String).ValueString()

		doId, err := client.DataObject().GetDataObjectIdByName(ctx, fullname, *result.DataSource)
		if err != nil {
			diagnostics.AddError("Failed to get data object id", err.Error())

			return diagnostics
		}

		permissionSet := whatDataObjectAttributes["permissions"].(types.Set)
		permissions := make([]*string, 0, len(permissionSet.Elements()))

		for _, p := range permissionSet.Elements() {
			permission := p.(types.String)
			permissions = append(permissions, permission.ValueStringPointer())
		}

		globalPermissionSet := whatDataObjectAttributes["global_permissions"].(types.Set)
		globalPermissions := make([]*string, 0, len(globalPermissionSet.Elements()))

		for _, p := range globalPermissionSet.Elements() {
			permission := p.(types.String)
			globalPermissions = append(globalPermissions, permission.ValueStringPointer())
		}

		result.WhatDataObjects = append(result.WhatDataObjects, raitoType.AccessProviderWhatInputDO{
			DataObjects: []*string{
				&doId,
			},
			Permissions:       permissions,
			GlobalPermissions: globalPermissions,
		})
	}

	return nil
}

func (m *GrantResourceModel) whatAbacToApInput(ctx context.Context, client *sdk.RaitoClient, result *raitoType.AccessProviderInput) (diagnostics diag.Diagnostics) {
	attributes := m.WhatAbacRule.Attributes()

	doTypes, doDiagnostics := utils.StringSetToSlice(ctx, attributes["do_types"].(types.Set))
	diagnostics.Append(doDiagnostics...)

	if diagnostics.HasError() {
		return diagnostics
	}

	permissions, permissionDiagnostics := utils.StringSetToSlice(ctx, attributes["permissions"].(types.Set))
	diagnostics.Append(permissionDiagnostics...)

	if diagnostics.HasError() {
		return diagnostics
	}

	globalPermissions, globalPermissionDiagnostics := utils.StringSetToSlice(ctx, attributes["global_permissions"].(types.Set))
	diagnostics.Append(globalPermissionDiagnostics...)

	if diagnostics.HasError() {
		return diagnostics
	}

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
		DoTypes:           doTypes,
		Permissions:       permissions,
		GlobalPermissions: globalPermissions,
		Scope:             scope,
		Rule:              *abacInput,
	}

	return nil
}

func (m *GrantResourceModel) FromAccessProvider(ctx context.Context, client *sdk.RaitoClient, ap *raitoType.AccessProvider) diag.Diagnostics {
	apResourceModel := m.GetAccessProviderResourceModel()
	diagnostics := apResourceModel.FromAccessProvider(ap)

	if diagnostics.HasError() {
		return diagnostics
	}

	m.SetAccessProviderResourceModel(apResourceModel)
	m.Type = types.StringPointerValue(ap.Type)

	if len(ap.DataSources) != 1 {
		diagnostics.AddError("Failed to get data source", fmt.Sprintf("Expected exactly one data source, got: %d.", len(ap.DataSources)))

		return diagnostics
	}

	m.DataSource = types.StringValue(ap.DataSources[0].Id)

	if ap.WhatType == raitoType.WhoAndWhatTypeDynamic && ap.WhatAbacRule != nil {
		permissions, pDiagnostics := utils.SliceToStringSet(ctx, ap.WhatAbacRule.Permissions)
		diagnostics.Append(pDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}

		globalPermissions, gpDiagnostics := utils.SliceToStringSet(ctx, ap.WhatAbacRule.GlobalPermissions)
		diagnostics.Append(gpDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}

		doTypes, dtDiagnostics := utils.SliceToStringSet(ctx, ap.WhatAbacRule.DoTypes)
		diagnostics.Append(dtDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}

		abacRule := jsontypes.NewNormalizedPointerValue(ap.WhatAbacRule.RuleJson)

		var scopeItems []attr.Value

		cancelCtx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()

		for scopeItem := range client.AccessProvider().GetAccessProviderAbacWhatScope(cancelCtx, ap.Id) {
			if scopeItem.HasError() {
				diagnostics.AddError("Failed to load access provider abac scope", scopeItem.GetError().Error())

				return diagnostics
			}

			scopeItems = append(scopeItems, types.StringValue(scopeItem.MustGetItem().FullName))
		}

		scope, scopeDiagnostics := types.SetValue(types.StringType, scopeItems)
		diagnostics.Append(scopeDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}

		object, whatAbacDiagnostics := types.ObjectValue(map[string]attr.Type{
			"do_types":           types.SetType{ElemType: types.StringType},
			"permissions":        types.SetType{ElemType: types.StringType},
			"global_permissions": types.SetType{ElemType: types.StringType},
			"scope":              types.SetType{ElemType: types.StringType},
			"rule":               jsontypes.NormalizedType{},
		}, map[string]attr.Value{
			"do_types":           doTypes,
			"permissions":        permissions,
			"global_permissions": globalPermissions,
			"scope":              scope,
			"rule":               abacRule,
		})

		diagnostics.Append(whatAbacDiagnostics...)

		if diagnostics.HasError() {
			return diagnostics
		}

		m.WhatAbacRule = object
	}

	return diagnostics
}

func (m *GrantResourceModel) UpdateOwners(owners types.Set) {
	m.Owners = owners
}

type GrantResource struct {
	AccessProviderResource[GrantResourceModel, *GrantResourceModel]
}

func NewGrantResource() resource.Resource {
	return &GrantResource{
		AccessProviderResource[GrantResourceModel, *GrantResourceModel]{
			readHooks:      []ReadHook[GrantResourceModel, *GrantResourceModel]{readGrantWhatItems},
			validationHoos: []ValidationHook[GrantResourceModel, *GrantResourceModel]{validateGrantWhatItems},
		},
	}
}

func (g *GrantResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_grant"
}

func (g *GrantResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	attributes := g.schema("grant")
	attributes["type"] = schema.StringAttribute{
		Required:            false,
		Optional:            true,
		Computed:            false,
		Sensitive:           false,
		Description:         "The type of the grant",
		MarkdownDescription: "The type of the grant",
	}
	attributes["data_source"] = schema.StringAttribute{
		Required:            true,
		Optional:            false,
		Computed:            false,
		Sensitive:           false,
		Description:         "The ID of the data source of the grant",
		MarkdownDescription: "The ID of the data source of the grant",
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(3),
		},
	}
	attributes["what_data_objects"] = schema.SetNestedAttribute{
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"fullname": schema.StringAttribute{
					Required:            true,
					Optional:            false,
					Computed:            false,
					Sensitive:           false,
					Description:         "The full name of the data object in the data source",
					MarkdownDescription: "The full name of the data object in the data source",
				},
				"permissions": schema.SetAttribute{
					ElementType:         types.StringType,
					Required:            false,
					Optional:            true,
					Computed:            true,
					Sensitive:           false,
					Description:         "The set of permissions granted to the data object",
					MarkdownDescription: "The set of permissions granted to the data object",
					Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				},
				"global_permissions": schema.SetAttribute{
					ElementType:         types.StringType,
					Required:            false,
					Optional:            true,
					Computed:            true,
					Sensitive:           false,
					Description:         "The set of global permissions granted to the data object",
					MarkdownDescription: fmt.Sprintf("The set of global permissions granted to the data object. Allowed values are %v", types2.AllGlobalPermissions),
					Validators: []validator.Set{
						setvalidator.ValueStringsAre(
							stringvalidator.OneOf(types2.AllGlobalPermissions...),
						),
					},
					Default: setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{
						types.StringValue(types2.GlobalPermissionRead),
					})),
				},
			},
		},
		Required:            false,
		Optional:            true,
		Computed:            false,
		Sensitive:           false,
		Description:         "The data object what items associated to the grant.",
		MarkdownDescription: "The data object what items associated to the grant. When this is not set (nil), the what list will not be overridden. This is typically used when this should be managed from Raito Cloud.",
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
			"do_types": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "Set of data object types associated to the abac rule",
				MarkdownDescription: "Set of data object types associated to the abac rule",
				Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
			"permissions": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "Set of permissions that should be granted on the matching data object",
				MarkdownDescription: "Set of permissions that should be granted on the matching data object",
				Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
			"global_permissions": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            false,
				Optional:            true,
				Computed:            true,
				Sensitive:           false,
				Description:         "Set of global permissions that should be granted on the matching data object",
				MarkdownDescription: fmt.Sprintf("Set of global permissions that should be granted on the matching data object. Allowed values are %v", types2.AllGlobalPermissions),
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.OneOf(types2.AllGlobalPermissions...),
					),
				},
				Default: setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{
					types.StringValue(types2.GlobalPermissionRead),
				})),
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
		Description:         "Grant access control resource",
		MarkdownDescription: "Grant access control resource",
		Version:             1,
	}
}

func readGrantWhatItems(ctx context.Context, client *sdk.RaitoClient, data *GrantResourceModel) (diagnostics diag.Diagnostics) {
	if !data.WhatDataObjects.IsNull() {
		cancelCtx, cancelFunc := context.WithCancel(ctx)
		defer cancelFunc()

		whatItemsChannel := client.AccessProvider().GetAccessProviderWhatDataObjectList(cancelCtx, data.Id.ValueString())

		stateWhatItems := make([]attr.Value, 0)

		for whatItem := range whatItemsChannel {
			if whatItem.HasError() {
				diagnostics.AddError("Failed to get what data objects", whatItem.GetError().Error())

				return diagnostics
			}

			what := whatItem.GetItem()

			var id *string

			if what.DataObject != nil {
				id = &what.DataObject.FullName
			} else {
				diagnostics.AddError("Invalid what data object", "Received data object is nil")

				continue
			}

			permissions := make([]attr.Value, 0, len(what.Permissions))
			for _, p := range what.Permissions {
				permissions = append(permissions, types.StringPointerValue(p))
			}

			globalPermissions := make([]attr.Value, 0, len(what.GlobalPermissions))
			for _, p := range what.GlobalPermissions {
				globalPermissions = append(globalPermissions, types.StringValue(strings.ToUpper(*p)))
			}

			stateWhatItems = append(stateWhatItems, types.ObjectValueMust(map[string]attr.Type{
				"fullname": types.StringType,
				"permissions": types.SetType{
					ElemType: types.StringType,
				},
				"global_permissions": types.SetType{
					ElemType: types.StringType,
				},
			}, map[string]attr.Value{
				"fullname":           types.StringPointerValue(id),
				"permissions":        types.SetValueMust(types.StringType, permissions),
				"global_permissions": types.SetValueMust(types.StringType, globalPermissions),
			}))
		}

		whatDataObject, whatDiag := types.SetValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"fullname": types.StringType,
				"permissions": types.SetType{
					ElemType: types.StringType,
				},
				"global_permissions": types.SetType{
					ElemType: types.StringType,
				},
			},
		}, stateWhatItems)

		diagnostics.Append(whatDiag...)

		if diagnostics.HasError() {
			return diagnostics
		}

		data.WhatDataObjects = whatDataObject
	}

	return diagnostics
}

func validateGrantWhatItems(_ context.Context, data *GrantResourceModel) (diagnostics diag.Diagnostics) {
	if !data.WhatDataObjects.IsNull() && !data.WhatAbacRule.IsNull() {
		diagnostics.AddError("Cannot set both what_data_objects and what_abac_rule", "Grant Resource cannot have both what_data_objects and what_abac_rule")

		return diagnostics
	}

	return diagnostics
}
