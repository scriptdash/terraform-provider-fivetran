package model

import (
	"fmt"
	"strings"

	gfcommon "github.com/fivetran/go-fivetran/common"
	"github.com/fivetran/go-fivetran/connectors"
	"github.com/fivetran/terraform-provider-fivetran/fivetran/common"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type ConnectorDatasourceModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	ConnectedBy types.String `tfsdk:"connected_by"`
	CreatedAt   types.String `tfsdk:"created_at"`
	GroupId     types.String `tfsdk:"group_id"`
	Service     types.String `tfsdk:"service"`

	DestinationSchema types.Object `tfsdk:"destination_schema"`

	SucceededAt     types.String `tfsdk:"succeeded_at"`
	FailedAt        types.String `tfsdk:"failed_at"`
	ServiceVersion  types.String `tfsdk:"service_version"`
	SyncFrequency   types.Int64  `tfsdk:"sync_frequency"`
	ScheduleType    types.String `tfsdk:"schedule_type"`
	Paused          types.Bool   `tfsdk:"paused"`
	PauseAfterTrial types.Bool   `tfsdk:"pause_after_trial"`
	DailySyncTime   types.String `tfsdk:"daily_sync_time"`

	Status types.Object `tfsdk:"status"`

	Config types.Object `tfsdk:"config"`
}

var (
	codeMessageAttrTypes = map[string]attr.Type{
		"code":    types.StringType,
		"message": types.StringType,
	}
)

func (d *ConnectorDatasourceModel) ReadFromResponse(resp connectors.DetailsWithCustomConfigNoTestsResponse) {
	responseContainer := ConnectorModelContainer{}
	responseContainer.ReadFromResponseData(resp.Data.DetailsResponseDataCommon, resp.Data.Config)
	d.ReadFromContainer(responseContainer)

	d.SucceededAt = types.StringValue(resp.Data.SucceededAt.String())
	d.FailedAt = types.StringValue(resp.Data.FailedAt.String())
	d.ServiceVersion = types.StringValue(fmt.Sprintf("%v", *resp.Data.ServiceVersion))
	d.SyncFrequency = types.Int64Value(int64(*resp.Data.SyncFrequency))
	d.ScheduleType = types.StringValue(resp.Data.ScheduleType)
	d.Paused = types.BoolValue(*resp.Data.Paused)
	d.PauseAfterTrial = types.BoolValue(*resp.Data.PauseAfterTrial)
	if resp.Data.DailySyncTime != "" {
		d.DailySyncTime = types.StringValue(resp.Data.DailySyncTime)
	} else {
		d.DailySyncTime = types.StringNull()
	}

	codeMessageAttrType := types.ObjectType{
		AttrTypes: codeMessageAttrTypes,
	}

	warns := []attr.Value{}
	for _, w := range resp.Data.Status.Warnings {
		warns = append(warns, readCommonResponse(w))
	}
	tasks := []attr.Value{}
	for _, t := range resp.Data.Status.Tasks {
		tasks = append(tasks, readCommonResponse(t))
	}

	wsV, _ := types.SetValue(codeMessageAttrType, warns)
	tsV, _ := types.SetValue(codeMessageAttrType, tasks)

	status, _ := types.ObjectValue(
		map[string]attr.Type{
			"setup_state":        types.StringType,
			"is_historical_sync": types.BoolType,
			"sync_state":         types.StringType,
			"update_state":       types.StringType,
			"tasks":              types.SetType{ElemType: codeMessageAttrType},
			"warnings":           types.SetType{ElemType: codeMessageAttrType},
		},
		map[string]attr.Value{
			"setup_state":        types.StringValue(resp.Data.Status.SetupState),
			"is_historical_sync": types.BoolPointerValue(resp.Data.Status.IsHistoricalSync),
			"sync_state":         types.StringValue(resp.Data.Status.SyncState),
			"update_state":       types.StringValue(resp.Data.Status.UpdateState),
			"warnings":           wsV,
			"tasks":              tsV,
		},
	)
	d.Status = status
}

func readCommonResponse(r gfcommon.CommonResponse) attr.Value {
	result, _ := types.ObjectValue(codeMessageAttrTypes,
		map[string]attr.Value{
			"code":    types.StringValue(r.Code),
			"message": types.StringValue(r.Message),
		})
	return result
}

type ConnectorResourceModel struct {
	Id                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	ConnectedBy       types.String `tfsdk:"connected_by"`
	CreatedAt         types.String `tfsdk:"created_at"`
	GroupId           types.String `tfsdk:"group_id"`
	Service           types.String `tfsdk:"service"`
	DestinationSchema types.Object `tfsdk:"destination_schema"`

	Config   types.Object   `tfsdk:"config"`
	Auth     types.Object   `tfsdk:"auth"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`

	RunSetupTests     types.Bool `tfsdk:"run_setup_tests"`
	TrustCertificates types.Bool `tfsdk:"trust_certificates"`
	TrustFingerprints types.Bool `tfsdk:"trust_fingerprints"`
}

func (d *ConnectorResourceModel) ReadFromResponse(resp connectors.DetailsWithCustomConfigNoTestsResponse) {
	responseContainer := ConnectorModelContainer{}
	responseContainer.ReadFromResponseData(resp.Data.DetailsResponseDataCommon, resp.Data.Config)
	d.ReadFromContainer(responseContainer)
}

func (d *ConnectorResourceModel) ReadFromCreateResponse(resp connectors.DetailsWithCustomConfigResponse) {
	responseContainer := ConnectorModelContainer{}
	responseContainer.ReadFromResponseData(resp.Data.DetailsResponseDataCommon, resp.Data.Config)
	d.ReadFromContainer(responseContainer)
}

func (d *ConnectorResourceModel) GetConfigMap(nullOnNull bool) (map[string]interface{}, error) {
	if d.Config.IsNull() && nullOnNull {
		return nil, nil
	}
	result := getValueFromAttrValue(d.Config, common.GetConfigFieldsMap(), nil, d.Service.ValueString()).(map[string]interface{})
	serviceName := d.Service.ValueString()
	serviceFields := common.GetFieldsForService(serviceName)
	allFields := common.GetConfigFieldsMap()
	err := patchServiceSpecificFields(result, serviceName, serviceFields, allFields)
	return result, err
}

func (d *ConnectorResourceModel) GetAuthMap(nullOnNull bool) (map[string]interface{}, error) {
	if d.Auth.IsNull() && nullOnNull {
		return nil, nil
	}
	serviceName := d.Service.ValueString()
	serviceFields := common.GetAuthFieldsForService(serviceName)
	allFields := common.GetAuthFieldsMap()

	result := getValueFromAttrValue(d.Auth, allFields, nil, serviceName).(map[string]interface{})
	err := patchServiceSpecificFields(result, serviceName, serviceFields, allFields)
	return result, err
}

func (d *ConnectorResourceModel) GetDestinatonSchemaForConfig() (map[string]interface{}, error) {
	return getDestinatonSchemaForConfig(d.Service,
		d.DestinationSchema.Attributes()["name"],
		d.DestinationSchema.Attributes()["table"],
		d.DestinationSchema.Attributes()["prefix"],
	)
}

func (d *ConnectorResourceModel) ReadFromContainer(c ConnectorModelContainer) {
	d.Id = types.StringValue(c.Id)
	d.Name = types.StringValue(c.Schema)
	d.ConnectedBy = types.StringValue(c.ConnectedBy)
	d.CreatedAt = types.StringValue(c.CreatedAt)
	d.GroupId = types.StringValue(c.GroupId)
	d.Service = types.StringValue(c.Service)

	d.DestinationSchema = getDestinationSchemaValue(c.Service, c.Schema)

	// Always read config; if we imported with an earlier version of this code,
	// the config might be null in Terraform but obviously exists in Fivetran.
	d.Config = getValue(
		types.ObjectType{AttrTypes: getAttrTypes(common.GetConfigFieldsMap())},
		c.Config,
		getValueFromAttrValue(d.Config, common.GetConfigFieldsMap(), nil, c.Service).(map[string]interface{}),
		common.GetConfigFieldsMap(), nil, c.Service).(basetypes.ObjectValue)
}

func (d *ConnectorDatasourceModel) ReadFromContainer(c ConnectorModelContainer) {
	d.Id = types.StringValue(c.Id)
	d.Name = types.StringValue(c.Schema)
	d.ConnectedBy = types.StringValue(c.ConnectedBy)
	d.CreatedAt = types.StringValue(c.CreatedAt)
	d.GroupId = types.StringValue(c.GroupId)
	d.Service = types.StringValue(c.Service)

	d.DestinationSchema = getDestinationSchemaValue(c.Service, c.Schema)

	d.Config = getValue(
		types.ObjectType{AttrTypes: getAttrTypes(common.GetConfigFieldsMap())},
		c.Config,
		c.Config,
		common.GetConfigFieldsMap(),
		nil,
		c.Service).(basetypes.ObjectValue)
}

type ConnectorModelContainer struct {
	Id          string
	Name        string
	ConnectedBy string
	CreatedAt   string
	GroupId     string
	Service     string
	Schema      string

	Config map[string]interface{}

	RunSetupTests     bool
	TrustCertificates bool
	TrustFingerprints bool
}

func (c *ConnectorModelContainer) ReadFromResponseData(data connectors.DetailsResponseDataCommon, config map[string]interface{}) {
	c.Id = data.ID
	c.Name = data.Schema
	c.ConnectedBy = data.ConnectedBy
	c.CreatedAt = data.CreatedAt.String()
	c.GroupId = data.GroupID
	c.Service = data.Service
	c.Schema = data.Schema
	c.Config = config
}

func getDestinatonSchemaForConfig(serviceId, nameAttr, tableAttr, prefixAttr attr.Value) (map[string]interface{}, error) {
	service := serviceId.(basetypes.StringValue).ValueString()
	if _, ok := common.GetDestinationSchemaFields()[service]; !ok {
		return nil, fmt.Errorf("unknown connector service: `%v`", service)
	}
	if common.GetDestinationSchemaFields()[service]["schema_prefix"] {
		if prefixAttr.IsNull() || prefixAttr.IsUnknown() {
			return nil, fmt.Errorf("`destination_schema.prefix` field is required to create `%v` connector", service)
		}
		if !nameAttr.IsNull() {
			return nil, fmt.Errorf("`destination_schema.name` field can't be set for `%v` connector", service)
		}
		if !tableAttr.IsNull() {
			return nil, fmt.Errorf("`destination_schema.table` field can't be set for `%v` connector", service)
		}
		prefix := prefixAttr.(types.String).ValueString()
		return map[string]interface{}{
			"schema_prefix": prefix,
		}, nil
	} else {
		if nameAttr.IsNull() {
			return nil, fmt.Errorf("`destination_schema.name` field is required to create `%v` connector", service)
		}
		result := map[string]interface{}{
			"schema": nameAttr.(types.String).ValueString(),
		}
		if common.GetDestinationSchemaFields()[service]["table"] {
			if tableAttr.IsNull() {
				return nil, fmt.Errorf("`destination_schema.table` field is required to create `%v` connector", service)
			}
			result["table"] = tableAttr.(types.String).ValueString()
		}
		return result, nil
	}
}

func getDestinationSchemaValue(service, schema string) types.Object {
	r, _ := types.ObjectValue(
		map[string]attr.Type{
			"name":   types.StringType,
			"table":  types.StringType,
			"prefix": types.StringType,
		},
		getDestinationSchemaValuesMap(service, schema),
	)
	return r
}

func getDestinationSchemaValuesMap(service, schema string) map[string]attr.Value {
	if _, ok := common.GetDestinationSchemaFields()[service]; !ok {
		panic(fmt.Errorf("unknown connector service: `%v`", service))
	}

	if common.GetDestinationSchemaFields()[service]["schema_prefix"] {
		return map[string]attr.Value{
			"name":   types.StringNull(),
			"table":  types.StringNull(),
			"prefix": types.StringValue(schema),
		}
	} else {
		result := map[string]attr.Value{
			"table":  types.StringNull(),
			"prefix": types.StringNull(),
		}
		s := strings.Split(schema, ".")
		result["name"] = types.StringValue(s[0])
		if len(s) > 1 && common.GetDestinationSchemaFields()[service]["table"] {
			result["table"] = types.StringValue(s[1])
		}
		return result
	}
}
