package fivetran

import (
	_ "embed"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

//go:embed open-api-spec.json
var oasJson []byte

const SERVICES_PATH = "components.schemas.NewConnectorRequestV1.discriminator.mapping"
const SCHEMAS_PATH = "components.schemas."
const PROPERTIES_PATH = ".properties.config.properties"
const SCHEMAS_FILE_PATH = "/open-api-spec.json"

const OBJECT_FIELD = "object"
const INT_FIELD = "integer"
const BOOL_FIELD = "boolean"
const ARRAY_FIELD = "array"

func getConnectorSchemaConfig(readonly bool, version int) *schema.Schema {
	fields := getFields()

	if version == 0 {
		// Sensitive config fields, Fivetran returns this fields masked
		fields["oauth_token"] = &schema.Schema{
			Type:        schema.TypeString,
			Optional:    !readonly,
			Sensitive:   true,
			Computed:    readonly,
			Description: "The Twitter App access token.",
		}
		fields["oauth_token_secret"] = &schema.Schema{
			Type:        schema.TypeString,
			Optional:    !readonly,
			Sensitive:   true,
			Computed:    readonly,
			Description: "The Twitter App access token secret.",
		}

		// Computed
		fields["api_type"] = &schema.Schema{
			Type:        schema.TypeString,
			Optional:    !readonly,
			Computed:    true,
			Description: "",
		}
		fields["daily_api_call_limit"] = &schema.Schema{
			Type:        schema.TypeString,
			Optional:    !readonly,
			Computed:    true,
			Description: "",
		}
		fields["test_table_name"] = &schema.Schema{
			Type:        schema.TypeString,
			Optional:    !readonly,
			Computed:    readonly,
			Description: "",
		}
		fields["unique_id"] = &schema.Schema{
			Type:        schema.TypeString,
			Optional:    !readonly,
			Computed:    readonly,
			Description: "",
		}
		fields["organization"] = &schema.Schema{
			Type:        schema.TypeString,
			Optional:    !readonly,
			Computed:    readonly,
			Description: "",
		}
		fields["environment"] = &schema.Schema{
			Type:        schema.TypeString,
			Optional:    !readonly,
			Computed:    readonly,
			Description: "",
		}

		// String collections
		fields["report_suites"] = &schema.Schema{
			Type:        schema.TypeSet,
			Optional:    !readonly,
			Computed:    readonly,
			Description: "Specific report suites to sync. Must be populated if `sync_mode` is set to `SpecificReportSuites`.",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		}
		fields["elements"] = &schema.Schema{
			Type:        schema.TypeSet,
			Optional:    !readonly,
			Computed:    readonly,
			Description: "The elements that you want to sync.",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		}
	}

	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Computed: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: fields,
		},
	}
}

func getFields() map[string]*schema.Schema {
	schemaJson := getSchemaJson()

	services := getAvailableServiceIds(schemaJson)

	fields := make(map[string]*schema.Schema)
	for _, service := range services {
		path := SCHEMAS_PATH + service + PROPERTIES_PATH

		serviceSchema := schemaJson.Path(path).ChildrenMap()
		serviceFields := createFields(serviceSchema)
		for property, value := range serviceFields {
			if existingValue, ok := fields[property]; ok {
				if existingValue.Type != value.Type {
					property = service + property
				} else if existingValue.Type == schema.TypeSet {
					if _, ok := value.Elem.([]string); ok {
						if _, ok := existingValue.Elem.([]string); ok {

						} else {
							property = service + "." + property
						}
					}
					value, ok = updateExistingValue(existingValue, value)
					if !ok {
						property = strings.ToLower(service + "_" + property)
					}
				}
			}
			fields[property] = value
		}
	}
	return fields
}

func getAvailableServiceIds(oasJson *gabs.Container) []string {
	services := []string{}

	file := oasJson.Path(SERVICES_PATH).ChildrenMap()

	for serviceKey := range file {
		services = append(services, serviceKey+"_config_V1")
	}

	return services
}

func getSchemaJson() *gabs.Container {
	shemaJson, err := gabs.ParseJSON(oasJson)
	if err != nil {
		panic(err)
	}

	return shemaJson
}

func createFields(nodesMap map[string]*gabs.Container) map[string]*schema.Schema {
	fields := make(map[string]*schema.Schema)

	for key, node := range nodesMap {
		nodeSchema := &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		}

		nodeDescription := node.Search("description").Data()

		if nodeDescription != nil {
			nodeSchema.Description = nodeDescription.(string)
		}

		nodeFormat := node.Search("format").Data()

		if nodeFormat != nil && nodeFormat == "password" {
			nodeSchema.Sensitive = true
			fields[key] = nodeSchema
			continue
		}

		nodeType := node.Search("type").Data()

		switch nodeType {
		case INT_FIELD:
			nodeSchema.Type = schema.TypeInt
		case BOOL_FIELD:
			nodeSchema.Type = schema.TypeBool
		case ARRAY_FIELD:
			nodeSchema = getArrayFieldSchema(node)
		}
		fields[key] = nodeSchema
	}

	return fields
}

func getArrayFieldSchema(node *gabs.Container) *schema.Schema {
	itemType := node.Path("items.type").Data()

	childrenMap := node.Path("items.properties").ChildrenMap()

	arraySchema := &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Computed: true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		}}

	if itemType == OBJECT_FIELD && len(childrenMap) > 0 {
		childrenSchemaMap := createFields(childrenMap)

		arraySchema.Elem = &schema.Resource{
			Schema: childrenSchemaMap,
		}
	}

	return arraySchema
}

func updateExistingValue(existingValue *schema.Schema, newValue *schema.Schema) (*schema.Schema, bool) {
	if existingSchemaResourceValue, ok := existingValue.Elem.(*schema.Resource); ok {
		if newSchemaResourceValue, ok := newValue.Elem.(*schema.Resource); ok {
			for newSchemaResourceKey, newSchemaResourceValue := range newSchemaResourceValue.Schema {
				existingSchemaResourceValue.Schema[newSchemaResourceKey] = newSchemaResourceValue
			}
			existingValue.Elem = existingSchemaResourceValue
			return existingValue, true
		}
	}
	return existingValue, false
}
