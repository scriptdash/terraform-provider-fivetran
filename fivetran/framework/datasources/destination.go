package datasources

import (
	"context"
	"fmt"

	"github.com/fivetran/terraform-provider-fivetran/fivetran/framework/core"
	"github.com/fivetran/terraform-provider-fivetran/fivetran/framework/core/model"
	fivetranSchema "github.com/fivetran/terraform-provider-fivetran/fivetran/framework/core/schema"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func Destination() datasource.DataSource {
	return &destination{}
}

// Ensure the implementation satisfies the desired interfaces.
var _ datasource.DataSourceWithConfigure = &destination{}

type destination struct {
	core.ProviderDatasource
}

func (d *destination) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destination"
}

func (d *destination) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: fivetranSchema.DestinationAttributesSchema().GetDatasourceSchema(),
		Blocks:     fivetranSchema.DestinationDatasourceBlocks(),
	}
}

func (d *destination) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.GetClient() == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Fivetran Client",
			"Please report this issue to the provider developers.",
		)

		return
	}

	var data model.DestinationDatasourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	response, err := d.GetClient().NewDestinationDetails().DestinationID(data.Id.ValueString()).DoCustom(ctx)

	if err != nil {
		resp.Diagnostics.AddError(
			"Read error.",
			fmt.Sprintf("%v; code: %v; message: %v", err, response.Code, response.Message),
		)
		return
	}

	data.ReadFromResponse(response)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
