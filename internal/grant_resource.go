package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/raito-io/sdk"
	raitoType "github.com/raito-io/sdk/types"
	"github.com/raito-io/sdk/types/models"

	types2 "github.com/raito-io/terraform-provider-raito/internal/types"
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

	// GrantResourceModel properties.
	Type            types.String `tfsdk:"type"`
	DataSource      types.String `tfsdk:"data_source"`
	WhatDataObjects types.Set    `tfsdk:"what_data_objects"`
}

func (m *GrantResourceModel) GetAccessProviderResourceModel() *AccessProviderResourceModel {
	return &AccessProviderResourceModel{
		Id:          m.Id,
		Name:        m.Name,
		Description: m.Description,
		State:       m.State,
		Who:         m.Who,
	}
}

func (m *GrantResourceModel) SetAccessProviderResourceModel(ap *AccessProviderResourceModel) {
	m.Id = ap.Id
	m.Name = ap.Name
	m.Description = ap.Description
	m.State = ap.State
	m.Who = ap.Who
}

func (m *GrantResourceModel) ToAccessProviderInput(ctx context.Context, client *sdk.RaitoClient, result *raitoType.AccessProviderInput) diag.Diagnostics {
	diagnostics := m.GetAccessProviderResourceModel().ToAccessProviderInput(ctx, client, result)

	if diagnostics.HasError() {
		return diagnostics
	}

	result.Type = m.Type.ValueStringPointer()
	result.DataSource = m.DataSource.ValueStringPointer()
	result.Action = utils.Ptr(models.AccessProviderActionGrant)

	if !m.WhatDataObjects.IsNull() && !m.WhatDataObjects.IsUnknown() {
		elements := m.WhatDataObjects.Elements()

		result.WhatDataObjects = make([]raitoType.AccessProviderWhatInputDO, 0, len(elements))

		for _, whatDataObject := range elements {
			whatDataObjectObject := whatDataObject.(types.Object)
			whatDataObjectAttributes := whatDataObjectObject.Attributes()

			fullname := whatDataObjectAttributes["name"].(types.String).ValueString()

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
	}

	return diagnostics
}

func (m *GrantResourceModel) FromAccessProvider(ap *raitoType.AccessProvider) diag.Diagnostics {
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

	return diagnostics
}

type GrantResource struct {
	AccessProviderResource[*GrantResourceModel]
}

func NewGrantResource() resource.Resource {
	return &GrantResource{}
}

func (g GrantResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_grant"
}

func (g GrantResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	attributes := g.schema("grant")
	attributes["type"] = schema.StringAttribute{
		Required:            false,
		Optional:            true,
		Computed:            true,
		Sensitive:           false,
		Description:         "Type of the grant",
		MarkdownDescription: "Type of the grant",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	attributes["data_source"] = schema.StringAttribute{
		Required:            true,
		Optional:            false,
		Computed:            false,
		Sensitive:           false,
		Description:         "Data source ID of the grant",
		MarkdownDescription: "Data source ID of the grant",
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(3),
		},
	}
	attributes["what_data_objects"] = schema.SetNestedAttribute{
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					Required:            true,
					Optional:            false,
					Computed:            false,
					Sensitive:           false,
					Description:         "Full name of the data object in the data source",
					MarkdownDescription: "Full name of the data object in the data source",
				},
				"permissions": schema.SetAttribute{
					ElementType:         types.StringType,
					Required:            false,
					Optional:            true,
					Computed:            true,
					Sensitive:           false,
					Description:         "Set of permissions granted to the data object",
					MarkdownDescription: "Set of permissions granted to the data object",
					Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				},
				"global_permissions": schema.SetAttribute{
					ElementType:         types.StringType,
					Required:            false,
					Optional:            true,
					Computed:            true,
					Sensitive:           false,
					Description:         "Set of global permissions granted to the data object",
					MarkdownDescription: fmt.Sprintf("Set of global permissions granted to the data object. Allowed values are %v", types2.AllGlobalPermissions),
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
		Description:         "Data objects what items associated to the grant.",
		MarkdownDescription: "Data objects what items associated to the grant. May not be set if what_abac_rule is set. Items are managed by Raito Cloud if what_data_objects is not set (nil).",
	}
	// TODO once abac is in production
	//attributes["what_abac_rule"] = schema.SetNestedAttribute{
	//	NestedObject: schema.NestedAttributeObject{
	//		Attributes: map[string]schema.Attribute{
	//			"scope": schema.SetAttribute{
	//				ElementType:         types.StringType,
	//				Required:            false,
	//				Optional:            true,
	//				Computed:            true,
	//				Sensitive:           false,
	//				Description:         "Scope of the defined abac rule",
	//				MarkdownDescription: "Scope of the defined abac rule",
	//				Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
	//			},
	//			"do_types": schema.SetAttribute{
	//				ElementType:         types.StringType,
	//				Required:            false,
	//				Optional:            true,
	//				Computed:            true,
	//				Sensitive:           false,
	//				Description:         "Set of data object types associated to the abac rule",
	//				MarkdownDescription: "Set of data object types associated to the abac rule",
	//				Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
	//			},
	//			"permissions": schema.SetAttribute{
	//				ElementType:         types.StringType,
	//				Required:            false,
	//				Optional:            true,
	//				Computed:            true,
	//				Sensitive:           false,
	//				Description:         "Set of permissions that should be granted on the matching data object",
	//				MarkdownDescription: "Set of permissions that should be granted on the matching data object",
	//				Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
	//			},
	//			"global_permissions": schema.SetAttribute{
	//				ElementType:         types.StringType,
	//				Required:            false,
	//				Optional:            true,
	//				Computed:            true,
	//				Sensitive:           false,
	//				Description:         "Set of global permissions that should be granted on the matching data object",
	//				MarkdownDescription: fmt.Sprintf("Set of global permissions that should be granted on the matching data object. Allowed values are %v", types2.AllGlobalPermissions),
	//				Validators: []validator.Set{
	//					setvalidator.ValueStringsAre(
	//						stringvalidator.OneOf(types2.AllGlobalPermissions...),
	//					),
	//				},
	//				Default: setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{
	//					types.StringValue(types2.GlobalPermissionRead),
	//				})),
	//			},
	//			"rule": schema.StringAttribute{
	//				Required:            true,
	//				Optional:            false,
	//				Computed:            false,
	//				Sensitive:           false,
	//				Description:         "Json representation of the abac rule",
	//				MarkdownDescription: "json representation of the abac rule",
	//				Validators:          []validator.String{
	//					//TODO
	//				},
	//				PlanModifiers:       nil,
	//				Default:             nil,
	//			},
	//		},
	//	},
	//	CustomType:          nil,
	//	Required:            false,
	//	Optional:            false,
	//	Computed:            false,
	//	Sensitive:           false,
	//	Description:         "",
	//	MarkdownDescription: "",
	//	DeprecationMessage:  "",
	//	Validators:          nil,
	//	PlanModifiers:       nil,
	//	Default:             nil,
	//}

	response.Schema = schema.Schema{
		Attributes:          attributes,
		Description:         "Grant access control resource",
		MarkdownDescription: "Grant access control resource",
		Version:             1,
	}
}

func (g GrantResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data GrantResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	g.create(ctx, &data, response)
}

func (g GrantResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data GrantResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	g.read(ctx, &data, response, func(ctx context.Context, client *sdk.RaitoClient, data *GrantResourceModel) (diagnostics diag.Diagnostics) {
		if !data.WhatDataObjects.IsNull() {
			whatItemsChannel := client.AccessProvider().GetAccessProviderWhatDataObjectList(ctx, data.Id.ValueString())

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
					diagnostics.AddError("Invalid what data object name", "Data object full name is not set")

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
					"name": types.StringType,
					"permissions": types.SetType{
						ElemType: types.StringType,
					},
					"global_permissions": types.SetType{
						ElemType: types.StringType,
					},
				}, map[string]attr.Value{
					"name":               types.StringPointerValue(id),
					"permissions":        types.SetValueMust(types.StringType, permissions),
					"global_permissions": types.SetValueMust(types.StringType, globalPermissions),
				}))
			}

			whatDataObject, whatDiag := types.SetValue(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name": types.StringType,
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
	})
}

func (g GrantResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data GrantResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	g.update(ctx, data.Id.ValueString(), &data, response)
}
