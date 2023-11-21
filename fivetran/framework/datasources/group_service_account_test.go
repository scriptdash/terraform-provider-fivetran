package datasources_test

import (
	"net/http"
	"testing"

	"github.com/fivetran/go-fivetran/tests/mock"
	tfmock "github.com/fivetran/terraform-provider-fivetran/fivetran/tests/mock"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestDataSourceGroupServiceAccountMappingMock(t *testing.T) {
	var getHandler *mock.Handler

	step1 := resource.TestStep{
		Config: `
		data "fivetran_group_service_account" "test_data" {
			provider = fivetran-provider
			id = "group_id"
		}`,

		Check: resource.ComposeAggregateTestCheckFunc(
			func(s *terraform.State) error {
				tfmock.AssertEqual(t, getHandler.Interactions, 2)
				return nil
			},
			resource.TestCheckResourceAttr("data.fivetran_group_service_account.test_data", "service_account", "service_account_value"),
			resource.TestCheckResourceAttr("data.fivetran_group_service_account.test_data", "id", "group_id"),
		),
	}

	resource.Test(
		t,
		resource.TestCase{
			PreCheck: func() {
				tfmock.MockClient().Reset()

				getHandler = tfmock.MockClient().When(http.MethodGet, "/v1/groups/group_id/service-account").ThenCall(
					func(req *http.Request) (*http.Response, error) {
						var responseData = tfmock.CreateMapFromJsonString(t, `
						{
							"service_account": "service_account_value"
						}
						`)
						return tfmock.FivetranSuccessResponse(t, req, http.StatusOK, "Success", responseData), nil
					},
				)
			},
			ProtoV5ProviderFactories: tfmock.ProtoV5ProviderFactory,
			CheckDestroy: func(s *terraform.State) error {
				return nil
			},
			Steps: []resource.TestStep{
				step1,
			},
		},
	)
}
