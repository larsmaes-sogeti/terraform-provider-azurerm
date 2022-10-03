package resource

import (
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-06-01/resources"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/resource/parse"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tags"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

func resourceResourceTags() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Create: resourceResourceTagsCreateUpdate,
		Read:   resourceResourceTagsRead,
		Update: resourceResourceTagsCreateUpdate,
		Delete: resourceResourceTagsDelete,
		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := parse.ResourceGroupID(id)
			return err
		}),

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(90 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(90 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(90 * time.Minute),
		},

		Schema: map[string]*pluginsdk.Schema{
			"resource_id": {
				Type: pluginsdk.TypeString,
			},
			"tags": tags.Schema(),
		},
	}
}

func resourceResourceTagsCreateUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Resource.TagsClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	resource_id := d.Get("resource_id").(string)
	t := d.Get("tags").(map[string]interface{})

	if d.IsNewResource() {
		existing, err := client.GetAtScope(ctx, resource_id)
		if err != nil {
			if !utils.ResponseWasNotFound(existing.Response) {
				return fmt.Errorf("checking for presence of existing tag: %+v", err)
			}
		}

		if existing.ID != nil && *existing.ID != "" {
			return tf.ImportAsExistsError("azurerm_resource_tags", *existing.ID)
		}
	}

	parameters := resources.TagsResource{
		Properties: &resources.Tags{
			Tags: tags.Expand(t),
		},
	}

	if _, err := client.CreateOrUpdateAtScope(ctx, resource_id, parameters); err != nil {
		return fmt.Errorf("creating Tags %q: %+v", resource_id, err)
	}

	resp, err := client.GetAtScope(ctx, resource_id)
	if err != nil {
		return fmt.Errorf("retrieving Tags %q: %+v", resource_id, err)
	}

	// @tombuildsstuff: intentionally leaving this for now, since this'll need
	// details in the upgrade notes given how the Resource Group ID is cased incorrectly
	// but needs to be fixed (resourcegroups -> resourceGroups)
	d.SetId(*resp.ID)

	return resourceResourceTagsRead(d, meta)
}

func resourceResourceTagsRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Resource.TagsClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	// id, err := parse.ResourceGroupID(d.Id())
	// if err != nil {
	// 	return err
	// }

	resp, err := client.GetAtScope(ctx, d.Id())
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			log.Printf("[INFO] Error reading resource tags %q - removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("reading resource tags: %+v", err)
	}

	d.Set("tags", resp.Properties)
	return nil
}

func resourceResourceTagsDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Resource.TagsClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	if _, err := client.DeleteAtScope(ctx, d.Id()); err != nil {

		return fmt.Errorf("deleting %s: %+v", d.Id(), err)
	}

	return nil
}
