package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsVpcPeeringConnectionAccept() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVPCPeeringAcceptCreate,
		Read:   resourceAwsVPCPeeringAcceptRead,
		Update: resourceAwsVPCPeeringAcceptUpdate,
		Delete: resourceAwsVPCPeeringAcceptDelete,

		Schema: map[string]*schema.Schema{
			"peering_connection_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"accept_status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsVPCPeeringAcceptCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	d.SetId(d.Get("peering_connection_id").(string))

	if cur, ok := d.Get("accept_status").(string); ok && cur == ec2.VpcPeeringConnectionStateReasonCodeActive {
		// already accepted
		return nil
	}

	status, err := resourceVPCPeeringConnectionAccept(conn, d.Id())
	if err != nil {
		return err
	}
	d.Set("accept_status", status)

	if err := setTags(conn, d); err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending-acceptance"},
		Target:  []string{"active"},
		Refresh: resourceAwsVPCPeeringConnectionStateRefreshFunc(conn, d.Id()),
		Timeout: 1 * time.Minute,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return errwrap.Wrapf(fmt.Sprintf(
			"Error waiting for VPC Peering Connection Accept (%s) to become available: {{err}}",
			d.Id()), err)
	}

	return nil
}

func resourceAwsVPCPeeringAcceptRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	pcRaw, status, err := resourceAwsVPCPeeringConnectionStateRefreshFunc(conn, d.Get("peering_connection_id").(string))()
	if err != nil {
		return err
	}
	d.Set("accept_status", status)
	d.SetId(d.Get("peering_connection_id").(string))

	pc := pcRaw.(*ec2.VpcPeeringConnection)
	d.Set("tags", tagsToMap(pc.Tags))

	return nil
}

func resourceAwsVPCPeeringAcceptDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsVPCPeeringAcceptUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	if err := setTags(conn, d); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	return resourceAwsVPCPeeringAcceptRead(d, meta)
}
