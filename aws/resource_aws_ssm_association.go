package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSsmAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSsmAssociationCreate,
		Read:   resourceAwsSsmAssociationRead,
		Update: resourceAwsSsmAssociationUpdate,
		Delete: resourceAwsSsmAssociationDelete,

		MigrateState:  resourceAwsSsmAssociationMigrateState,
		SchemaVersion: 1,

		Schema: map[string]*schema.Schema{
			"association_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"association_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"instance_id": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
			"document_version": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"parameters": {
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
			},
			"schedule_expression": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"output_location": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"s3_bucket_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"s3_key_prefix": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"targets": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 5,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"values": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func resourceAwsSsmAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[DEBUG] SSM association create: %s", d.Id())

	associationInput := &ssm.CreateAssociationInput{
		Name: aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("association_name"); ok {
		associationInput.AssociationName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("instance_id"); ok {
		associationInput.InstanceId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("document_version"); ok {
		associationInput.DocumentVersion = aws.String(v.(string))
	}

	if v, ok := d.GetOk("schedule_expression"); ok {
		associationInput.ScheduleExpression = aws.String(v.(string))
	}

	if v, ok := d.GetOk("parameters"); ok {
		associationInput.Parameters = expandSSMDocumentParameters(v.(map[string]interface{}))
	}

	if _, ok := d.GetOk("targets"); ok {
		associationInput.Targets = expandAwsSsmTargets(d.Get("targets").([]interface{}))
	}

	if v, ok := d.GetOk("output_location"); ok {
		associationInput.OutputLocation = expandSSMAssociationOutputLocation(v.([]interface{}))
	}

	resp, err := ssmconn.CreateAssociation(associationInput)
	if err != nil {
		return fmt.Errorf("[ERROR] Error creating SSM association: %s", err)
	}

	if resp.AssociationDescription == nil {
		return fmt.Errorf("[ERROR] AssociationDescription was nil")
	}

	d.SetId(*resp.AssociationDescription.AssociationId)
	d.Set("association_id", resp.AssociationDescription.AssociationId)

	return resourceAwsSsmAssociationRead(d, meta)
}

func resourceAwsSsmAssociationRead(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[DEBUG] Reading SSM Association: %s", d.Id())

	params := &ssm.DescribeAssociationInput{
		AssociationId: aws.String(d.Id()),
	}

	resp, err := ssmconn.DescribeAssociation(params)

	if err != nil {
		if isAWSErr(err, ssm.ErrCodeAssociationDoesNotExist, "") {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("[ERROR] Error reading SSM association: %s", err)
	}
	if resp.AssociationDescription == nil {
		return fmt.Errorf("[ERROR] AssociationDescription was nil")
	}

	association := resp.AssociationDescription
	d.Set("association_name", association.AssociationName)
	d.Set("instance_id", association.InstanceId)
	d.Set("name", association.Name)
	d.Set("parameters", association.Parameters)
	d.Set("association_id", association.AssociationId)
	d.Set("schedule_expression", association.ScheduleExpression)
	d.Set("document_version", association.DocumentVersion)

	if err := d.Set("targets", flattenAwsSsmTargets(association.Targets)); err != nil {
		return fmt.Errorf("[DEBUG] Error setting targets error: %#v", err)
	}

	if err := d.Set("output_location", flattenAwsSsmAssociationOutoutLocation(association.OutputLocation)); err != nil {
		return fmt.Errorf("[DEBUG] Error setting output_location error: %#v", err)
	}

	return nil
}

func resourceAwsSsmAssociationUpdate(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[DEBUG] SSM Association update: %s", d.Id())

	associationInput := &ssm.UpdateAssociationInput{
		AssociationId: aws.String(d.Get("association_id").(string)),
	}

	// AWS creates a new version every time the association is updated, so everything should be passed in the update.

	hasChanges := false

	if d.HasChange("association_name") {
		hasChanges = true
	}

	if d.HasChange("document_version") {
		hasChanges = true
	}

	if d.HasChange("schedule_expression") {
		hasChanges = true
	}

	if d.HasChange("parameters") {
		hasChanges = true
	}

	if d.HasChange("output_location") {
		hasChanges = true
	}

	if d.HasChange("targets") {
		hasChanges = true
	}

	if hasChanges {
		if v, ok := d.GetOk("association_name"); ok {
			associationInput.AssociationName = aws.String(v.(string))
		}

		if v, ok := d.GetOk("document_version"); ok {
			associationInput.DocumentVersion = aws.String(v.(string))
		}

		if v, ok := d.GetOk("schedule_expression"); ok {
			associationInput.ScheduleExpression = aws.String(v.(string))
		}

		if v, ok := d.GetOk("parameters"); ok {
			associationInput.Parameters = expandSSMDocumentParameters(v.(map[string]interface{}))
		}

		if _, ok := d.GetOk("targets"); ok {
			associationInput.Targets = expandAwsSsmTargets(d.Get("targets").([]interface{}))
		}

		if v, ok := d.GetOk("output_location"); ok {
			associationInput.OutputLocation = expandSSMAssociationOutputLocation(v.([]interface{}))
		}
	}

	_, err := ssmconn.UpdateAssociation(associationInput)
	if err != nil {
		return fmt.Errorf("[ERROR] Error updating SSM association: %s", err)
	}

	return resourceAwsSsmAssociationRead(d, meta)
}

func resourceAwsSsmAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[DEBUG] Deleting SSM Association: %s", d.Id())

	params := &ssm.DeleteAssociationInput{
		AssociationId: aws.String(d.Get("association_id").(string)),
	}

	_, err := ssmconn.DeleteAssociation(params)

	if err != nil {
		return fmt.Errorf("[ERROR] Error deleting SSM association: %s", err)
	}

	return nil
}

func expandSSMDocumentParameters(params map[string]interface{}) map[string][]*string {
	var docParams = make(map[string][]*string)
	for k, v := range params {
		values := make([]*string, 1)
		values[0] = aws.String(v.(string))
		docParams[k] = values
	}

	return docParams
}

func expandSSMAssociationOutputLocation(config []interface{}) *ssm.InstanceAssociationOutputLocation {
	if config == nil {
		return nil
	}

	//We only allow 1 Item so we can grab the first in the list only
	locationConfig := config[0].(map[string]interface{})

	S3OutputLocation := &ssm.S3OutputLocation{
		OutputS3BucketName: aws.String(locationConfig["s3_bucket_name"].(string)),
	}

	if v, ok := locationConfig["s3_key_prefix"]; ok {
		S3OutputLocation.OutputS3KeyPrefix = aws.String(v.(string))
	}

	return &ssm.InstanceAssociationOutputLocation{
		S3Location: S3OutputLocation,
	}
}

func flattenAwsSsmAssociationOutoutLocation(location *ssm.InstanceAssociationOutputLocation) []map[string]interface{} {
	if location == nil {
		return nil
	}

	result := make([]map[string]interface{}, 0)
	item := make(map[string]interface{})

	item["s3_bucket_name"] = *location.S3Location.OutputS3BucketName

	if location.S3Location.OutputS3KeyPrefix != nil {
		item["s3_key_prefix"] = *location.S3Location.OutputS3KeyPrefix
	}

	result = append(result, item)

	return result
}
