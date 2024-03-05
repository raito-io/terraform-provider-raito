package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/raito-io/sdk-go"
	"github.com/raito-io/sdk-go/services"
	raitoTypes "github.com/raito-io/sdk-go/types"

	"github.com/raito-io/terraform-provider-raito/internal/utils"
)

var _ datasource.DataSource = (*UserDataSource)(nil)

type UserDataSourceModel struct {
	Id        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Email     types.String `tfsdk:"email"`
	Type      types.String `tfsdk:"type"`
	RaitoUser types.Bool   `tfsdk:"raito_user"`
	Roles     types.Set    `tfsdk:"roles"`
}

type UserDataSource struct {
	client *sdk.RaitoClient
}

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

func (u *UserDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_user"
}

func (u *UserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The ID of the requested user",
				MarkdownDescription: "The ID of the requested user",
			},
			"email": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Computed:            false,
				Sensitive:           false,
				Description:         "The email of the requested user",
				MarkdownDescription: "The email of the requested user",
			},
			"name": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The name of the requested user",
				MarkdownDescription: "The name of the requested user",
			},
			"type": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The type of the requested user (Human or Machine)",
				MarkdownDescription: "The type of the requested user (Human or Machine)",
			},
			"raito_user": schema.BoolAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "Whether the requested user is a Raito user",
				MarkdownDescription: "Whether the requested user is a Raito user",
			},
			"roles": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "User global roles",
				MarkdownDescription: "User global roles",
			},
		},
		Description:         "Find a user by email address",
		MarkdownDescription: "Find a Raito [User](https://docs.raito.io/docs/cloud/admin/user_management) by email address",
	}
}

func (u *UserDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data UserDataSourceModel

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	user, err := u.client.User().GetUserByEmail(ctx, data.Email.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Failed to get user", err.Error())

		return
	}

	data.Id = types.StringValue(user.Id)
	data.Name = types.StringValue(user.Name)
	data.Type = types.StringValue(string(user.Type))
	data.RaitoUser = types.BoolValue(user.IsRaitoUser)

	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	roles := u.client.Role().ListRoleAssignmentsOnUser(cancelCtx, user.Id, services.WithRoleAssignmentListFilter(&raitoTypes.RoleAssignmentFilterInput{
		OnlyGlobal: utils.Ptr(true),
	}))

	actualRoles := make([]types.String, 0)

	for role := range roles {
		if role.HasError() {
			response.Diagnostics.AddError("Failed to get roles", role.GetError().Error())

			return
		}

		roleId := role.GetItem().GetId()
		roleName := roleId[:len(roleId)-len(roleIdSuffix)] // Cut off "Role" suffix
		actualRoles = append(actualRoles, types.StringValue(roleName))
	}

	rolesSet, rolesDiagnostics := types.SetValueFrom(ctx, types.StringType, actualRoles)

	response.Diagnostics.Append(rolesDiagnostics...)

	if response.Diagnostics.HasError() {
		return
	}

	data.Roles = rolesSet

	response.Diagnostics.Append(response.State.Set(ctx, data)...)
}

func (u *UserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	u.client = client
}
