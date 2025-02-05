package vpcsubnet

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/linode/linodego"
	"github.com/linode/terraform-provider-linode/v2/linode/helper"
)

func NewResource() resource.Resource {
	return &Resource{
		BaseResource: helper.NewBaseResource(
			helper.BaseResourceConfig{
				Name:   "linode_vpc_subnet",
				IDType: types.Int64Type,
				Schema: &frameworkResourceSchema,
			},
		),
	}
}

type Resource struct {
	helper.BaseResource
}

func (r *Resource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	tflog.Debug(ctx, "Import "+r.Config.Name)
	helper.ImportStateWithMultipleIDs(ctx, req, resp, "vpc_id", "id")
}

func (r *Resource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data VPCSubnetModel
	client := r.Meta.Client

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createOpts := linodego.VPCSubnetCreateOptions{
		Label: data.Label.ValueString(),
		IPv4:  data.IPv4.ValueString(),
	}

	vpcId := helper.FrameworkSafeInt64ToInt(data.VPCId.ValueInt64(), &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	subnet, err := client.CreateVPCSubnet(ctx, createOpts, vpcId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create VPC subnet.",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(data.FlattenSubnet(ctx, subnet, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	client := r.Meta.Client
	var data VPCSubnetModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpcId := helper.FrameworkSafeInt64ToInt(data.VPCId.ValueInt64(), &resp.Diagnostics)
	id := helper.FrameworkSafeInt64ToInt(data.ID.ValueInt64(), &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	subnet, err := client.GetVPCSubnet(ctx, vpcId, id)
	if err != nil {
		if lerr, ok := err.(*linodego.Error); ok && lerr.Code == 404 {
			resp.Diagnostics.AddWarning(
				"VPC subnet does not exist.",
				fmt.Sprintf(
					"Removing VPC subnet with ID %v from state because it no longer exists",
					id,
				),
			)
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to refresh the VPC subnet.",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(data.FlattenSubnet(ctx, subnet, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state VPCSubnetModel
	client := r.Meta.Client

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var updateOpts linodego.VPCSubnetUpdateOptions
	shouldUpdate := false

	if !state.Label.Equal(plan.Label) {
		updateOpts.Label = plan.Label.ValueString()
		shouldUpdate = true
	}

	if shouldUpdate {
		vpcId := helper.FrameworkSafeInt64ToInt(
			plan.VPCId.ValueInt64(),
			&resp.Diagnostics,
		)
		id := helper.FrameworkSafeInt64ToInt(plan.ID.ValueInt64(), &resp.Diagnostics)

		if resp.Diagnostics.HasError() {
			return
		}

		subnet, err := client.UpdateVPCSubnet(ctx, vpcId, id, updateOpts)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to update VPC subnet (%d).", id),
				err.Error(),
			)
			return
		}
		resp.Diagnostics.Append(plan.FlattenSubnet(ctx, subnet, true)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		req.State.GetAttribute(ctx, path.Root("updated"), &plan.Updated)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Resource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data VPCSubnetModel
	client := r.Meta.Client

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpcId := helper.FrameworkSafeInt64ToInt(data.VPCId.ValueInt64(), &resp.Diagnostics)
	id := helper.FrameworkSafeInt64ToInt(data.ID.ValueInt64(), &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	err := client.DeleteVPCSubnet(ctx, vpcId, id)
	if err != nil {
		if lerr, ok := err.(*linodego.Error); (ok && lerr.Code != 404) || !ok {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to delete the VPC subnet (%d)", data.ID.ValueInt64()),
				err.Error(),
			)
		}
		return
	}
}
