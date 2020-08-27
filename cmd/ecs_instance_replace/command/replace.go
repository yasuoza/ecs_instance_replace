package command

import (
	"flag"
	"log"
	"math"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/mitchellh/cli"
	instance_replace "github.com/yasuoza/ecs_instance_replace"
)

// ReplaceCommand reperesents replace command.
type ReplaceCommand struct {
	UI *cli.BasicUi
}

// Run executes parse args and pass args to RunContext.
func (c *ReplaceCommand) Run(args []string) int {
	if parseStatus := c.parseArgs(args); parseStatus != 0 {
		return parseStatus
	}

	instanceId := args[0]
	if err := c.run(instanceId); err != nil {
		log.Printf("Failed to replace instance_id: %s. Reason: %v\n", instanceId, err)
		return 1
	}

	return 0
}

// Help represents help message for replace command.
func (c *ReplaceCommand) Help() string {
	helpText := `
Usage: ecs_instance_replace replace EC2_INSTANEC_ID
  Will execute replace aginst given EC2_INSTANEC_ID.
  This command only supports EC2 instance for ECS Cluster managed by Auto Scaling Grouped.
  Following operations will be done by this command.
    - Increment Auto Scaling Group's DesiredCapacity and MaxSize if necessary.
    - Drain given EC2 instance.
    - Wait all tasks are migrated.
    - Remove Scale in protection.
    - Revert Auto Scaling Group's configuration.
`

	return strings.TrimSpace(helpText)
}

// Synopsis represents synopsis message for replace command.
func (c *ReplaceCommand) Synopsis() string {
	return "Replace given ec2 instance with new instance"
}

func (c *ReplaceCommand) parseArgs(args []string) int {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.Usage = func() {
		c.UI.Info(c.Help())
	}
	flags.Parse(args)

	args = flags.Args()
	if len(args) != 1 {
		flags.Usage()
		return 127
	}
	return 0
}

func (c *ReplaceCommand) run(instanceId string) error {
	app := instance_replace.NewApp(instanceId)

	scalingGroup, err := app.GetScalingGroup()
	if err != nil {
		return err
	}
	clusterArn, err := app.GetClusterArn()
	if err != nil {
		return err
	}
	containerInstanceArn, err := app.GetContainerInstanceArn(clusterArn)
	if err != nil {
		return err
	}

	beforeIds, err := app.ListInServiceInstanceIds(scalingGroup)
	if err != nil {
		return err
	}

	log.Printf("Target Cluster: %s\n", clusterArn)
	log.Printf("Target ASG: %s\n", *scalingGroup.AutoScalingGroupName)

	if err := replace(app, instanceId, scalingGroup, clusterArn, containerInstanceArn); err != nil {
		log.Fatal(err)
	}

	afterIds, err := app.ListInServiceInstanceIds(scalingGroup)
	if err != nil {
		log.Fatal(err)
	}

	diff := strings.Join(instance_replace.SliceDifference(afterIds, beforeIds), ",")
	log.Printf("%s is replaced by %s", instanceId, diff)

	return nil
}

func replace(app *instance_replace.App, instanceId string, scalingGroup *autoscaling.Group, clusterArn string, containerInstanceArn string) error {
	timeout := 15 * time.Minute
	newDesiredCnt := *scalingGroup.DesiredCapacity + 1
	newMaxSize := int64(math.Max(float64(newDesiredCnt), float64(*scalingGroup.MaxSize)))

	if err := app.UpdateAutoScalingGroupCapacity(scalingGroup, newDesiredCnt, newMaxSize); err != nil {
		return err
	}
	log.Printf(
		"Auto Scaling group %s is updated to DesiredCapacity: %d, MaxSize: %d\n",
		*scalingGroup.AutoScalingGroupName, newDesiredCnt, newMaxSize,
	)

	if err := app.WaitContainerInstanceActive(clusterArn, newDesiredCnt, timeout); err != nil {
		return err
	}

	if err := app.DrainTargetContainerInstance(clusterArn, containerInstanceArn); err != nil {
		return err
	}
	log.Printf("Container instance %s is marked DRAINING\n", containerInstanceArn)

	if err := app.WaitTargetContainerInstanceDrained(clusterArn, containerInstanceArn, timeout); err != nil {
		return err
	}

	if err := app.WaitTasksMigration(clusterArn, timeout, 240); err != nil {
		return err
	}

	if err := app.DeregisterContainerInstance(clusterArn, containerInstanceArn); err != nil {
		return err
	}
	log.Printf("Container instance %s is deregistered\n", containerInstanceArn)

	if err := app.TerminateInstance(scalingGroup); err != nil {
		return err
	}
	log.Printf("Desired capacity of %s is reverted to %d\n", *scalingGroup.AutoScalingGroupName, *scalingGroup.DesiredCapacity)

	if newMaxSize > *scalingGroup.MaxSize {
		if err := app.UpdateAutoScalingGroupMaxSize(scalingGroup, *scalingGroup.MaxSize); err != nil {
			return err
		}
		log.Printf(
			"Max size of Auto Scaling group %s is reverted to %d\n", *scalingGroup.AutoScalingGroupName, *scalingGroup.MaxSize,
		)
	}

	if err := app.WaitInstanceTermination(timeout); err != nil {
		return err
	}

	return nil
}
