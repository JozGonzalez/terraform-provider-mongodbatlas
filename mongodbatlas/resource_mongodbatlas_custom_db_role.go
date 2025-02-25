package mongodbatlas

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/mwielbut/pointy"
	"github.com/spf13/cast"
	matlas "go.mongodb.org/atlas/mongodbatlas"
)

func resourceMongoDBAtlasCustomDBRole() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceMongoDBAtlasCustomDBRoleCreate,
		ReadContext:   resourceMongoDBAtlasCustomDBRoleRead,
		UpdateContext: resourceMongoDBAtlasCustomDBRoleUpdate,
		DeleteContext: resourceMongoDBAtlasCustomDBRoleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceMongoDBAtlasCustomDBRoleImportState,
		},
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"role_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.All(
					validation.StringMatch(regexp.MustCompile(`[\w-]+`), "`role_name` can contain only letters, digits, underscores, and dashes"),
					func(v interface{}, k string) (ws []string, es []error) {
						value := v.(string)
						if strings.HasPrefix(value, "x-gen") {
							es = append(es, fmt.Errorf("`role_name` cannot start with 'xgen-'"))
						}
						return
					},
				),
			},
			"actions": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": {
							Type:     schema.TypeString,
							Required: true,
						},
						"resources": {
							Type:     schema.TypeSet,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"collection_name": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"database_name": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"cluster": {
										Type:     schema.TypeBool,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
			"inherited_roles": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"database_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"role_name": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

var (
	customRoleLock sync.Mutex
)

func resourceMongoDBAtlasCustomDBRoleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	customRoleLock.Lock()
	defer customRoleLock.Unlock()
	conn := meta.(*MongoDBClient).Atlas
	projectID := d.Get("project_id").(string)

	customDBRoleReq := &matlas.CustomDBRole{
		RoleName:       d.Get("role_name").(string),
		Actions:        expandActions(d),
		InheritedRoles: expandInheritedRoles(d),
	}

	stateConf := &retry.StateChangeConf{
		Pending: []string{"pending"},
		Target:  []string{"created", "failed"},
		Refresh: func() (interface{}, string, error) {
			customDBRoleRes, _, err := conn.CustomDBRoles.Create(ctx, projectID, customDBRoleReq)
			if err != nil {
				if strings.Contains(err.Error(), "Unexpected error") ||
					strings.Contains(err.Error(), "UNEXPECTED_ERROR") ||
					strings.Contains(err.Error(), "500") ||
					strings.Contains(err.Error(), "404") ||
					strings.Contains(err.Error(), "ATLAS_CUSTOM_ROLE_NOT_FOUND") {
					return nil, "pending", nil
				}
				return nil, "failed", err
			}

			return customDBRoleRes, "created", nil
		},
		Timeout:    10 * time.Minute,
		Delay:      3 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	// Wait, catching any errors
	_, err := stateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating custom db role: %s", err))
	}

	d.SetId(encodeStateID(map[string]string{
		"project_id": projectID,
		"role_name":  customDBRoleReq.RoleName,
	}))

	return resourceMongoDBAtlasCustomDBRoleRead(ctx, d, meta)
}

func resourceMongoDBAtlasCustomDBRoleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*MongoDBClient).Atlas
	ids := decodeStateID(d.Id())
	projectID := ids["project_id"]
	roleName := ids["role_name"]

	customDBRole, resp, err := conn.CustomDBRoles.Get(context.Background(), projectID, roleName)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}

		return diag.FromErr(fmt.Errorf("error getting custom db role information: %s", err))
	}

	if err := d.Set("role_name", customDBRole.RoleName); err != nil {
		return diag.FromErr(fmt.Errorf("error setting `role_name` for custom db role (%s): %s", d.Id(), err))
	}

	if err := d.Set("actions", flattenActions(customDBRole.Actions)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting `actions` for custom db role (%s): %s", d.Id(), err))
	}

	if err := d.Set("inherited_roles", flattenInheritedRoles(customDBRole.InheritedRoles)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting `inherited_roles` for custom db role (%s): %s", d.Id(), err))
	}

	return nil
}

func resourceMongoDBAtlasCustomDBRoleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	customRoleLock.Lock()
	defer customRoleLock.Unlock()
	conn := meta.(*MongoDBClient).Atlas
	ids := decodeStateID(d.Id())
	projectID := ids["project_id"]
	roleName := ids["role_name"]

	customDBRole, _, err := conn.CustomDBRoles.Get(ctx, projectID, roleName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error getting custom db role information: %s", err))
	}

	// Clean the roleName because it can be sent into the update request to avoid an unexpected error 500
	customDBRole.RoleName = ""

	if d.HasChange("actions") {
		customDBRole.Actions = expandActions(d)
	}

	if d.HasChange("inherited_roles") {
		customDBRole.InheritedRoles = expandInheritedRoles(d)
	}

	_, _, err = conn.CustomDBRoles.Update(ctx, projectID, roleName, customDBRole)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error updating custom db role (%s): %s", roleName, err))
	}

	return resourceMongoDBAtlasCustomDBRoleRead(ctx, d, meta)
}

func resourceMongoDBAtlasCustomDBRoleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*MongoDBClient).Atlas
	ids := decodeStateID(d.Id())
	projectID := ids["project_id"]
	roleName := ids["role_name"]

	stateConf := &retry.StateChangeConf{
		Pending: []string{"deleting"},
		Target:  []string{"deleted", "failed"},
		Refresh: func() (interface{}, string, error) {
			_, _, err := conn.CustomDBRoles.Get(ctx, projectID, roleName)
			if err != nil {
				if strings.Contains(err.Error(), "404") ||
					strings.Contains(err.Error(), "ATLAS_CUSTOM_ROLE_NOT_FOUND") {
					return "", "deleted", nil
				}
				return nil, "failed", err
			}

			_, err = conn.CustomDBRoles.Delete(ctx, projectID, roleName)
			if err != nil {
				return nil, "failed", fmt.Errorf("error deleting custom db role (%s): %s", roleName, err)
			}

			return nil, "deleting", nil
		},
		Timeout:    10 * time.Minute,
		Delay:      3 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	// Wait, catching any errors
	_, err := stateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceMongoDBAtlasCustomDBRoleImportState(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	conn := meta.(*MongoDBClient).Atlas

	parts := strings.SplitN(d.Id(), "-", 2)
	if len(parts) != 2 {
		return nil, errors.New("import format error: to import a custom db role use the format {project_id}-{role_name}")
	}

	projectID := parts[0]
	roleName := parts[1]

	r, _, err := conn.CustomDBRoles.Get(ctx, projectID, roleName)
	if err != nil {
		return nil, fmt.Errorf("couldn't import custom db role %s in project %s, error: %s", roleName, projectID, err)
	}

	d.SetId(encodeStateID(map[string]string{
		"project_id": projectID,
		"role_name":  r.RoleName,
	}))

	if err := d.Set("project_id", projectID); err != nil {
		log.Printf("[WARN] Error setting project_id for (%s): %s", d.Id(), err)
	}

	return []*schema.ResourceData{d}, nil
}

func expandActions(d *schema.ResourceData) []matlas.Action {
	actions := make([]matlas.Action, len(d.Get("actions").([]interface{})))

	for k, v := range d.Get("actions").([]interface{}) {
		a := v.(map[string]interface{})
		actions[k] = matlas.Action{
			Action:    a["action"].(string),
			Resources: expandActionResources(a["resources"].(*schema.Set)),
		}
	}

	return actions
}

func expandActionResources(resources *schema.Set) []matlas.Resource {
	actionResources := make([]matlas.Resource, resources.Len())

	for k, v := range resources.List() {
		resourceMap := v.(map[string]interface{})
		actionResources[k] = matlas.Resource{
			DB:         pointy.String(resourceMap["database_name"].(string)),
			Collection: pointy.String(resourceMap["collection_name"].(string)),
			Cluster:    pointy.Bool(cast.ToBool(resourceMap["cluster"])),
		}
	}

	return actionResources
}

func flattenActions(actions []matlas.Action) []map[string]interface{} {
	actionList := make([]map[string]interface{}, 0)
	for _, v := range actions {
		actionList = append(actionList, map[string]interface{}{
			"action":    v.Action,
			"resources": flattenActionResources(v.Resources),
		})
	}

	return actionList
}

func flattenActionResources(resources []matlas.Resource) []map[string]interface{} {
	actionResourceList := make([]map[string]interface{}, 0)

	for _, v := range resources {
		if cluster := v.Cluster; cluster != nil {
			actionResourceList = append(actionResourceList, map[string]interface{}{
				"cluster": v.Cluster,
			})
		} else {
			actionResourceList = append(actionResourceList, map[string]interface{}{
				"database_name":   cast.ToString(v.DB),
				"collection_name": cast.ToString(v.Collection),
			})
		}
	}

	return actionResourceList
}

func expandInheritedRoles(d *schema.ResourceData) []matlas.InheritedRole {
	vIR := d.Get("inherited_roles").(*schema.Set).List()
	ir := make([]matlas.InheritedRole, len(vIR))

	if len(vIR) != 0 {
		for i := range vIR {
			r := vIR[i].(map[string]interface{})

			ir[i] = matlas.InheritedRole{
				Db:   r["database_name"].(string),
				Role: r["role_name"].(string),
			}
		}
	}

	return ir
}

func flattenInheritedRoles(roles []matlas.InheritedRole) []map[string]interface{} {
	inheritedRoleList := make([]map[string]interface{}, 0)
	for _, v := range roles {
		inheritedRoleList = append(inheritedRoleList, map[string]interface{}{
			"database_name": v.Db,
			"role_name":     v.Role,
		})
	}

	return inheritedRoleList
}
