package internal

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/raito-io/golang-set/set"
	"github.com/raito-io/sdk"
	raitoType "github.com/raito-io/sdk/types"
	"github.com/raito-io/sdk/types/models"

	"github.com/raito-io/terraform-provider-raito/internal/utils"
)

type AccessProviderResourceModel struct {
	Id          types.String
	Name        types.String
	Description types.String
	State       types.String
	Who         types.Set
}

type AccessProviderModel interface {
	ToAccessProviderInput(ctx context.Context, client *sdk.RaitoClient, result *raitoType.AccessProviderInput) diag.Diagnostics
	FromAccessProvider(input *raitoType.AccessProvider) diag.Diagnostics
	GetAccessProviderResourceModel() *AccessProviderResourceModel
	SetAccessProviderResourceModel(model *AccessProviderResourceModel)
}

type AccessProviderResource[T AccessProviderModel] struct {
	client *sdk.RaitoClient
}

func (a *AccessProviderResource[T]) schema(typeName string) map[string]schema.Attribute {
	defaultSchema := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Required:            false,
			Optional:            false,
			Computed:            true,
			Sensitive:           false,
			Description:         fmt.Sprintf("ID of the %s", typeName),
			MarkdownDescription: fmt.Sprintf("ID of the %s", typeName),
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			Required:            true,
			Optional:            false,
			Computed:            false,
			Sensitive:           false,
			Description:         fmt.Sprintf("Name of the %s", typeName),
			MarkdownDescription: fmt.Sprintf("Name of the %s", typeName),
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(3),
			},
		},
		"description": schema.StringAttribute{
			Required:            false,
			Optional:            true,
			Computed:            true,
			Sensitive:           false,
			Description:         fmt.Sprintf("Description of the %s", typeName),
			MarkdownDescription: fmt.Sprintf("Description of the %s", typeName),
			Default:             stringdefault.StaticString(""),
		},
		"state": schema.StringAttribute{
			Required:            false,
			Optional:            true,
			Computed:            true,
			Sensitive:           false,
			Description:         fmt.Sprintf("State of the %s", typeName),
			MarkdownDescription: fmt.Sprintf("State of the %s. Possible values: [%q, %q]", typeName, models.AccessProviderStateActive.String(), models.AccessProviderStateInactive.String()),
			Validators: []validator.String{
				stringvalidator.OneOf(models.AccessProviderStateActive.String(), models.AccessProviderStateInactive.String()),
			},
			Default: stringdefault.StaticString(models.AccessProviderStateActive.String()),
		},
		"who": schema.SetNestedAttribute{
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"user": schema.StringAttribute{
						Required:            false,
						Optional:            true,
						Computed:            false,
						Sensitive:           false,
						Description:         "Email address of user",
						MarkdownDescription: "Email address of user. Can not be set if group or access control is set.",
						Validators: []validator.String{
							stringvalidator.RegexMatches(regexp.MustCompile(`.+@.+\..+`), "value must be a valid email address"),
						},
					},
					"group": schema.StringAttribute{
						Required:            false,
						Optional:            true,
						Computed:            false,
						Sensitive:           false,
						Description:         "Raito group ID",
						MarkdownDescription: "Raito group ID. Can not be set if user or access control is set.",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(3),
						},
					},
					"access_control": schema.StringAttribute{
						Required:            false,
						Optional:            true,
						Computed:            false,
						Sensitive:           false,
						Description:         "Raito access control ID",
						MarkdownDescription: "Raito access control ID. Can not be set if user or group is set.",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(3),
						},
					},
					"promise_duration": schema.Int64Attribute{
						Required:            false,
						Optional:            true,
						Computed:            false,
						Sensitive:           false,
						Description:         "Set to indicate who item as promise in seconds",
						MarkdownDescription: "Set promise_duration to indicate who item as promise. Defined in seconds.",
						Validators: []validator.Int64{
							int64validator.AtLeast(1),
						},
					},
				},
				CustomType:    nil,
				Validators:    nil,
				PlanModifiers: nil,
			},
			Required:            false,
			Optional:            true,
			Computed:            false,
			Sensitive:           false,
			Description:         fmt.Sprintf("Who items associated to the %s", typeName),
			MarkdownDescription: fmt.Sprintf("Who items associated to the %s. May not be set if who_abac_rule is set. Items are managed by Raito Cloud of who is not set (nil)", typeName),
		},
	}

	return defaultSchema
}

func (a *AccessProviderResource[T]) create(ctx context.Context, data T, response *resource.CreateResponse) {
	input := raitoType.AccessProviderInput{}
	state := data.GetAccessProviderResourceModel().State

	response.Diagnostics.Append(data.ToAccessProviderInput(ctx, a.client, &input)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Create the access provider
	ap, err := a.client.AccessProvider().CreateAccessProvider(ctx, input)
	if err != nil {
		response.Diagnostics.AddError("Failed to create access provider", err.Error())

		return
	}

	response.Diagnostics.Append(data.FromAccessProvider(ap)...)
	response.Diagnostics.Append(response.State.Set(ctx, data)...)

	if response.Diagnostics.HasError() {
		return
	}

	ap, diagnostics := a.updateState(ctx, data, state, ap)

	response.Diagnostics.Append(diagnostics...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(data.FromAccessProvider(ap)...)
	response.Diagnostics.Append(response.State.Set(ctx, data)...)
}

func (a *AccessProviderResource[T]) updateState(ctx context.Context, data T, state types.String, ap *raitoType.AccessProvider) (_ *raitoType.AccessProvider, diagnostics diag.Diagnostics) {
	if !state.Equal(data.GetAccessProviderResourceModel().State) {
		var err error

		if data.GetAccessProviderResourceModel().State.ValueString() == models.AccessProviderStateActive.String() {
			ap, err = a.client.AccessProvider().DeactivateAccessProvider(ctx, ap.Id)
			if err != nil {
				diagnostics.AddError("Failed to activate access provider", err.Error())

				return ap, diagnostics
			}
		} else if data.GetAccessProviderResourceModel().State.ValueString() == models.AccessProviderStateInactive.String() {
			ap, err = a.client.AccessProvider().ActivateAccessProvider(ctx, ap.Id)
			if err != nil {
				diagnostics.AddError("Failed to deactivate access provider", err.Error())

				return ap, diagnostics
			}
		} else {
			diagnostics.AddError("Invalid state", fmt.Sprintf("Invalid state: %s", data.GetAccessProviderResourceModel().State.ValueString()))

			return ap, diagnostics
		}
	}

	return ap, diagnostics
}

func (a *AccessProviderResource[T]) read(ctx context.Context, data T, response *resource.ReadResponse, hooks ...func(ctx context.Context, client *sdk.RaitoClient, data T) diag.Diagnostics) {
	apModel := data.GetAccessProviderResourceModel()

	// Get the access provider
	ap, err := a.client.AccessProvider().GetAccessProvider(ctx, apModel.Id.ValueString())
	if err != nil {
		notFoundErr := &raitoType.ErrNotFound{}
		if errors.As(err, &notFoundErr) {
			response.State.RemoveResource(ctx)

			return
		}

		response.Diagnostics.AddError("Failed to read access provider", err.Error())

		return
	}

	if ap.State == models.AccessProviderStateDeleted {
		response.State.RemoveResource(ctx)

		return
	}

	response.Diagnostics.Append(data.FromAccessProvider(ap)...)

	if response.Diagnostics.HasError() {
		return
	}

	apModel = data.GetAccessProviderResourceModel()

	// If who in initial state is not nil, get all who items
	if !apModel.Who.IsNull() {
		definedPromises := set.Set[string]{}

		// Search al promises defined in the terraform state
		for _, whoItem := range apModel.Who.Elements() {
			whoItemObject := whoItem.(types.Object)
			attributes := whoItemObject.Attributes()

			if !attributes["promise_duration"].IsNull() {
				if !attributes["user"].IsNull() {
					definedPromises.Add(_userPrefix(attributes["user"].(types.String).ValueString()))
				} else if !attributes["group"].IsNull() {
					definedPromises.Add(_groupPrefix(attributes["group"].(types.String).ValueString()))
				} else if !attributes["access_control"].IsNull() {
					definedPromises.Add(_accessControlPrefix(attributes["access_control"].(types.String).ValueString()))
				}
			}
		}

		stateWhoItems := make([]attr.Value, 0)

		// Get all who items. Ignore implemented promises.
		whoItems := a.client.AccessProvider().GetAccessProviderWhoList(ctx, apModel.Id.ValueString())
		for whoItem := range whoItems {
			if whoItem.HasError() {
				response.Diagnostics.AddError("Failed to read who item from access provider", whoItem.GetError().Error())

				return
			}

			var user, group, whoAp *string

			item := whoItem.GetItem()
			switch benificiaryItem := item.Item.(type) {
			case *raitoType.AccessProviderWhoListItemItemUser:
				user = benificiaryItem.Email
			case *raitoType.AccessProviderWhoListItemItemGroup:
				group = &benificiaryItem.Id
			case *raitoType.AccessProviderWhoListItemItemAccessProvider:
				whoAp = &benificiaryItem.Id
			default:
				response.Diagnostics.AddError("Invalid who item", fmt.Sprintf("Invalid who item: %T", benificiaryItem))

				return
			}

			if item.Type == raitoType.AccessWhoItemTypeWhogrant {
				if (user != nil && definedPromises.Contains(_userPrefix(*user))) || (group != nil && definedPromises.Contains(_groupPrefix(*group))) || (whoAp != nil && definedPromises.Contains(_accessControlPrefix(*whoAp))) {
					continue
				}
			} else if item.PromiseDuration == nil {
				response.Diagnostics.AddError("Invalid who item detected.", "Invalid who item. Promise duration not set on promise who item")
			}

			stateWhoItems = append(stateWhoItems, types.ObjectValueMust(
				map[string]attr.Type{
					"user":             types.StringType,
					"group":            types.StringType,
					"access_control":   types.StringType,
					"promise_duration": types.Int64Type,
				}, map[string]attr.Value{
					"user":             types.StringPointerValue(user),
					"group":            types.StringPointerValue(group),
					"access_control":   types.StringPointerValue(whoAp),
					"promise_duration": types.Int64PointerValue(item.PromiseDuration),
				}))
		}

		who, whoDiag := types.SetValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"user":             types.StringType,
				"group":            types.StringType,
				"access_control":   types.StringType,
				"promise_duration": types.Int64Type,
			},
		}, stateWhoItems)

		response.Diagnostics.Append(whoDiag...)

		if response.Diagnostics.HasError() {
			return
		}

		apModel.Who = who
	}

	// Set all global access provider attributes
	data.SetAccessProviderResourceModel(apModel)

	// Execute action specific hooks
	for _, hook := range hooks {
		response.Diagnostics.Append(hook(ctx, a.client, data)...)

		if response.Diagnostics.HasError() {
			return
		}
	}

	// Set new state of the access provider
	response.State.Set(ctx, data)
}

func (a *AccessProviderResource[T]) update(ctx context.Context, id string, data T, response *resource.UpdateResponse) {
	input := raitoType.AccessProviderInput{}
	state := data.GetAccessProviderResourceModel().State

	response.Diagnostics.Append(data.ToAccessProviderInput(ctx, a.client, &input)...)

	if response.Diagnostics.HasError() {
		return
	}

	// Check for implemented promises
	definedPromises := set.Set[string]{}

	for _, whoItem := range input.WhoItems {
		if whoItem.Type != nil && *whoItem.Type == raitoType.AccessWhoItemTypeWhopromise {
			if whoItem.User != nil {
				definedPromises.Add(_userPrefix(*whoItem.User))
			} else if whoItem.Group != nil {
				definedPromises.Add(_groupPrefix(*whoItem.Group))
			} else if whoItem.AccessProvider != nil {
				definedPromises.Add(_accessControlPrefix(*whoItem.AccessProvider))
			}
		}
	}

	whoItemChannel := a.client.AccessProvider().GetAccessProviderWhoList(ctx, id)
	for whoItem := range whoItemChannel {
		if whoItem.HasError() {
			response.Diagnostics.AddError("Failed to read who item from access provider", whoItem.GetError().Error())

			return
		}

		item := whoItem.GetItem()

		if item.Type == raitoType.AccessWhoItemTypeWhogrant {
			var key string
			var user, group, whoAp *string

			switch beneficiaryItem := item.Item.(type) {
			case *raitoType.AccessProviderWhoListItemItemUser:
				if beneficiaryItem.Email == nil {
					continue
				}

				key = _userPrefix(*beneficiaryItem.Email)
				user = &beneficiaryItem.Id
			case *raitoType.AccessProviderWhoListItemItemGroup:
				key = _groupPrefix(beneficiaryItem.Id)
				group = &beneficiaryItem.Id
			case *raitoType.AccessProviderWhoListItemItemAccessProvider:
				key = _accessControlPrefix(beneficiaryItem.Id)
				whoAp = &beneficiaryItem.Id
			default:
				continue
			}

			if definedPromises.Contains(key) {
				input.WhoItems = append(input.WhoItems, raitoType.WhoItemInput{
					Type:           utils.Ptr(raitoType.AccessWhoItemTypeWhogrant),
					User:           user,
					Group:          group,
					AccessProvider: whoAp,
					ExpiresAfter:   item.ExpiresAfter,
					ExpiresAt:      item.ExpiresAt,
				})
			}
		}
	}

	// Update access provider
	ap, err := a.client.AccessProvider().UpdateAccessProvider(ctx, id, input)
	if err != nil {
		response.Diagnostics.AddError("Failed to update access provider", err.Error())

		return
	}

	response.Diagnostics.Append(data.FromAccessProvider(ap)...)
	response.Diagnostics.Append(response.State.Set(ctx, data)...)

	if response.Diagnostics.HasError() {
		return
	}

	ap, diagnostics := a.updateState(ctx, data, state, ap)

	response.Diagnostics.Append(diagnostics...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(data.FromAccessProvider(ap)...)
	response.Diagnostics.Append(response.State.Set(ctx, data)...)
}

func (a *AccessProviderResource[T]) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data T

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	err := a.client.AccessProvider().DeleteAccessProvider(ctx, data.GetAccessProviderResourceModel().Id.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Failed to delete access provider", err.Error())

		return
	}

	response.State.RemoveResource(ctx)
}

func (a *AccessProviderResource[T]) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	a.client = client
}

func (a *AccessProviderResource[T]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (a *AccessProviderResource[T]) ValidateConfig(ctx context.Context, request resource.ValidateConfigRequest, response *resource.ValidateConfigResponse) {
	var data T

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	// For each who item check if exactly one of user, group or access_control is set.
	who := &data.GetAccessProviderResourceModel().Who

	if !who.IsNull() {
		for _, whoItem := range who.Elements() {
			whoItemAttribute := whoItem.(types.Object)

			attributes := whoItemAttribute.Attributes()

			attributesFound := 0

			attrFn := func(key string) {
				if attribute, found := attributes[key]; found && !attribute.IsNull() {
					attributesFound++
				}
			}

			attrFn("user")
			attrFn("group")
			attrFn("access_control")

			if attributesFound != 1 {
				response.Diagnostics.AddError(
					"Invalid who item. Exactly one of user, group or access_control must be set.",
					fmt.Sprintf("Expected exactly one of user, group or access_control, got: %d.", attributesFound),
				)

				break
			}
		}
	}
}

func (a *AccessProviderResourceModel) ToAccessProviderInput(ctx context.Context, client *sdk.RaitoClient, result *raitoType.AccessProviderInput) (diagnostics diag.Diagnostics) {
	result.Name = a.Name.ValueStringPointer()
	result.Description = a.Description.ValueStringPointer()

	if !a.Who.IsNull() && !a.Who.IsUnknown() {
		whoItems := a.Who.Elements()

		result.WhoItems = make([]raitoType.WhoItemInput, 0, len(whoItems))

		for _, whoItem := range whoItems {
			whoObject := whoItem.(types.Object)
			whoAttributes := whoObject.Attributes()

			raitoWhoItem := raitoType.WhoItemInput{
				Type: utils.Ptr(raitoType.AccessWhoItemTypeWhogrant),
			}

			if promiseDurationAttribute, found := whoAttributes["promise_duration"]; found && !promiseDurationAttribute.IsNull() {
				promiseDurationInt := promiseDurationAttribute.(types.Int64)
				raitoWhoItem.PromiseDuration = promiseDurationInt.ValueInt64Pointer()
				raitoWhoItem.Type = utils.Ptr(raitoType.AccessWhoItemTypeWhopromise)
			}

			if userAttribute, found := whoAttributes["user"]; found && !userAttribute.IsNull() {
				userString := userAttribute.(types.String)

				userInformation, err := client.User().GetUserByEmail(ctx, userString.ValueString())
				if err != nil {
					diagnostics.AddError("Failed to get user", err.Error())

					continue
				}

				raitoWhoItem.User = &userInformation.Id
			} else if groupAttribute, found := whoAttributes["group"]; found && !groupAttribute.IsNull() {
				raitoWhoItem.Group = groupAttribute.(types.String).ValueStringPointer()
			} else if accessControlAttribute, found := whoAttributes["access_control"]; found && !accessControlAttribute.IsNull() {
				raitoWhoItem.AccessProvider = accessControlAttribute.(types.String).ValueStringPointer()
			} else {
				diagnostics.AddError("Failed to get who item", "No user, group, or access control set")

				continue
			}

			result.WhoItems = append(result.WhoItems, raitoWhoItem)
		}
	}

	return diagnostics
}

func (a *AccessProviderResourceModel) FromAccessProvider(ap *raitoType.AccessProvider) (diagnostics diag.Diagnostics) {
	a.Id = types.StringValue(ap.Id)
	a.Name = types.StringValue(ap.Name)
	a.Description = types.StringValue(ap.Description)
	a.State = types.StringValue(ap.State.String())

	return diagnostics
}

func _userPrefix(u string) string {
	return "user:" + u
}

func _groupPrefix(g string) string {
	return "group:" + g
}

func _accessControlPrefix(a string) string {
	return "access_control:" + a
}
