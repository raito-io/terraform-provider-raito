package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/raito-io/sdk"
	"github.com/raito-io/sdk/services"
)

var _ datasource.DataSource = (*DataSourceDataSource)(nil)

type DataSourceDataSourceModel struct {
	Id                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	SyncMethod          types.String `tfsdk:"sync_method"`
	Parent              types.String `tfsdk:"parent"`
	NativeIdentityStore types.String `tfsdk:"native_identity_store"`
	IdentityStores      types.Set    `tfsdk:"identity_stores"`
}

type DataSourceDataSource struct {
	client *sdk.RaitoClient
}

func NewDataSourceDataSource() datasource.DataSource {
	return &DataSourceDataSource{}
}

func (d *DataSourceDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_datasource"
}

func (d *DataSourceDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The ID of the requested data source",
				MarkdownDescription: "The ID of the requested data source",
			},
			"name": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				Computed:            false,
				Sensitive:           false,
				Description:         "The name of the requested data source",
				MarkdownDescription: "The name of the requested data source",
				Validators:          nil,
			},
			"description": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The description of the data source",
				MarkdownDescription: "The description of the data source",
			},
			"sync_method": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The sync method of the data source. Should be set to ON_PREM for now.",
				MarkdownDescription: "The sync method of the data source. Should be set to `ON_PREM` for now.",
			},
			"parent": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The ID of the parent data source, if applicable",
				MarkdownDescription: "The ID of the parent data source, if applicable",
			},
			"native_identity_store": schema.StringAttribute{
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The ID of the native identity store",
				MarkdownDescription: "The ID of the native identity store",
			},
			"identity_stores": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            false,
				Optional:            false,
				Computed:            true,
				Sensitive:           false,
				Description:         "The IDs of the identity stores that also link to the data source",
				MarkdownDescription: "The IDs of the identity stores that also link to the data source",
			},
		},
		Description:         "Find a data source based on the name",
		MarkdownDescription: "Find a data source based on the name",
	}
}

func (d *DataSourceDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data DataSourceDataSourceModel

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	dsChan := d.client.DataSource().ListDataSources(ctx, services.WithDataSourceListSearch(&name))

	for ds := range dsChan {
		if ds.HasError() {
			response.Diagnostics.AddError("Failed to list data sources", ds.GetError().Error())

			return
		}

		dsItem := ds.GetItem()

		if dsItem.Name == name {
			identityStores, err := d.client.DataSource().ListIdentityStores(ctx, dsItem.Id)
			if err != nil {
				response.Diagnostics.AddError("Failed to list identity stores", err.Error())

				return
			}

			var nativeIs *string
			isIds := make([]attr.Value, 0, len(identityStores))

			for i, identityStore := range identityStores {
				if identityStore.Native {
					nativeIs = &identityStores[i].Id
				} else if !identityStore.Master {
					isIds = append(isIds, types.StringValue(identityStore.Id))
				}
			}

			isAttr, diagnostic := types.SetValue(types.StringType, isIds)
			response.Diagnostics.Append(diagnostic...)

			var parentId *string
			if dsItem.Parent != nil {
				parentId = &dsItem.Parent.Id
			}

			data.Id = types.StringValue(dsItem.Id)
			data.Description = types.StringValue(dsItem.Description)
			data.SyncMethod = types.StringValue(string(dsItem.SyncMethod))
			data.Parent = types.StringPointerValue(parentId)
			data.NativeIdentityStore = types.StringPointerValue(nativeIs)
			data.IdentityStores = isAttr

			response.State.Set(ctx, data)
		}
	}
}

func (d *DataSourceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client
}
