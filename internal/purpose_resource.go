package internal

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/raito-io/sdk"
	raitoType "github.com/raito-io/sdk/types"
	"github.com/raito-io/sdk/types/models"

	"github.com/raito-io/terraform-provider-raito/internal/utils"
)

var _ resource.Resource = (*PurposeResource)(nil)

type PurposeResourceModel struct {
	// AccessProviderResourceModel properties. This has to be duplicated because of https://github.com/hashicorp/terraform-plugin-framework/issues/242
	Id                types.String         `tfsdk:"id"`
	Name              types.String         `tfsdk:"name"`
	Description       types.String         `tfsdk:"description"`
	State             types.String         `tfsdk:"state"`
	Who               types.Set            `tfsdk:"who"`
	Owners            types.Set            `tfsdk:"owners"`
	WhoAbacRule       jsontypes.Normalized `tfsdk:"who_abac_rule"`
	WhoLocked         types.Bool           `tfsdk:"who_locked"`
	InheritanceLocked types.Bool           `tfsdk:"inheritance_locked"`
}

func (p *PurposeResourceModel) GetAccessProviderResourceModel() *AccessProviderResourceModel {
	return &AccessProviderResourceModel{
		Id:                p.Id,
		Name:              p.Name,
		Description:       p.Description,
		State:             p.State,
		Who:               p.Who,
		Owners:            p.Owners,
		WhoAbacRule:       p.WhoAbacRule,
		WhoLocked:         p.WhoLocked,
		InheritanceLocked: p.InheritanceLocked,
	}
}

func (p *PurposeResourceModel) SetAccessProviderResourceModel(model *AccessProviderResourceModel) {
	p.Id = model.Id
	p.Name = model.Name
	p.Description = model.Description
	p.State = model.State
	p.Who = model.Who
	p.Owners = model.Owners
	p.WhoAbacRule = model.WhoAbacRule
	p.WhoLocked = model.WhoLocked
	p.InheritanceLocked = model.InheritanceLocked
}

func (p *PurposeResourceModel) ToAccessProviderInput(ctx context.Context, client *sdk.RaitoClient, result *raitoType.AccessProviderInput) diag.Diagnostics {
	diagnostics := p.GetAccessProviderResourceModel().ToAccessProviderInput(ctx, client, result)

	if diagnostics.HasError() {
		return diagnostics
	}

	result.Action = utils.Ptr(models.AccessProviderActionPurpose)

	return diagnostics
}

func (p *PurposeResourceModel) FromAccessProvider(_ context.Context, _ *sdk.RaitoClient, input *raitoType.AccessProvider) diag.Diagnostics {
	apResourceModel := p.GetAccessProviderResourceModel()
	diagnostics := apResourceModel.FromAccessProvider(input)

	if diagnostics.HasError() {
		return diagnostics
	}

	p.SetAccessProviderResourceModel(apResourceModel)

	return diagnostics
}

func (p *PurposeResourceModel) UpdateOwners(owners types.Set) {
	p.Owners = owners
}

type PurposeResource struct {
	AccessProviderResource[PurposeResourceModel, *PurposeResourceModel]
}

func NewPurposeResource() resource.Resource {
	return &PurposeResource{
		AccessProviderResource[PurposeResourceModel, *PurposeResourceModel]{},
	}
}

func (p *PurposeResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_purpose"
}

func (p *PurposeResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	attributes := p.schema("purpose")

	response.Schema = schema.Schema{
		Attributes:          attributes,
		Description:         "The purpose access control resource",
		MarkdownDescription: "The resource for representing a Raito purpose access control.",
		Version:             1,
	}
}
