package ecs_instance_replace

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
)

// SleepInterval is used to for waiter interval.
var SleepInterval = 15 * time.Second

// App represents ecs_instance_replace application.
type App struct {
	asg        *autoscaling.AutoScaling
	ec2        *ec2.EC2
	ecs        *ecs.ECS
	instanceId string
}

// NewApp returns App data.
func NewApp(instanceId string) *App {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	return &App{
		instanceId: instanceId,
		asg:        autoscaling.New(sess),
		ec2:        ec2.New(sess),
		ecs:        ecs.New(sess),
	}
}

// GetScalingGroup retreives Auto Scaling Group which given instance is beongs to.
func (a *App) GetScalingGroup() (*autoscaling.Group, error) {
	inst, err := a.asg.DescribeAutoScalingInstances(&autoscaling.DescribeAutoScalingInstancesInput{
		InstanceIds: []*string{&a.instanceId},
		MaxRecords:  aws.Int64(1),
	})
	if err != nil {
		return nil, err
	}

	res, err := a.asg.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: []*string{inst.AutoScalingInstances[0].AutoScalingGroupName}})
	if err != nil {
		return nil, err
	}
	if len(res.AutoScalingGroups) != 1 {
		return nil, fmt.Errorf("failed to get scaling group for %s", a.instanceId)
	}
	return res.AutoScalingGroups[0], nil
}

// GetClusterArn retreives ECS cluster which given instance is beongs to.
func (a *App) GetClusterArn() (string, error) {
	var nextToken *string
	for {
		lout, err := a.ecs.ListClusters(&ecs.ListClustersInput{
			MaxResults: aws.Int64(100),
			NextToken:  nextToken,
		})
		if err != nil {
			return "", err
		}
		nextToken = lout.NextToken

		for _, clusterArn := range lout.ClusterArns {
			cout, err := a.ecs.ListContainerInstances(&ecs.ListContainerInstancesInput{
				Cluster:    clusterArn,
				Filter:     aws.String(fmt.Sprintf("ec2InstanceId==%s", a.instanceId)),
				MaxResults: aws.Int64(1),
				Status:     aws.String("Active"),
			})
			if err != nil {
				return "", err
			}
			if len(cout.ContainerInstanceArns) == 1 {
				return aws.StringValue(clusterArn), nil
			}
		}
		if nextToken == nil {
			return "", fmt.Errorf("failed to get cluster for instance %s", a.instanceId)
		}
	}
}

// GetContainerInstanceArn retreives continer instance arn for the given instance id.
func (a *App) GetContainerInstanceArn(clusterArn string) (string, error) {
	cout, err := a.ecs.ListContainerInstances(&ecs.ListContainerInstancesInput{
		Cluster:    aws.String(clusterArn),
		Filter:     aws.String(fmt.Sprintf("ec2InstanceId==%s", a.instanceId)),
		MaxResults: aws.Int64(1),
		Status:     aws.String("Active"),
	})
	if err != nil {
		return "", err
	}
	if len(cout.ContainerInstanceArns) == 0 {
		return "", fmt.Errorf("instnce_id %s is not in cluster %s", a.instanceId, clusterArn)
	}
	return aws.StringValue(cout.ContainerInstanceArns[0]), nil
}

// ListInServiceInstanceIds retreives Auto Scaling Group's instance ids which given instance belongs to.
func (a *App) ListInServiceInstanceIds(scalingGroup *autoscaling.Group) ([]string, error) {
	ret := []string{}

	var nextToken *string
	for {
		dout, err := a.asg.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{scalingGroup.AutoScalingGroupName},
			MaxRecords:            aws.Int64(100),
			NextToken:             nextToken,
		})
		if err != nil {
			return nil, err
		}
		for _, group := range dout.AutoScalingGroups {
			for _, inst := range group.Instances {
				if *inst.LifecycleState == "InService" {
					ret = append(ret, *inst.InstanceId)
				}
			}
		}
		if nextToken == nil {
			return ret, nil
		}
	}
}

// UpdateAutoScalingGroupCapacity updates Auto Scaling Group capacity.
func (a *App) UpdateAutoScalingGroupCapacity(scalingGroup *autoscaling.Group, newDesiredCnt int64, newMaxSize int64) error {
	params := &autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: scalingGroup.AutoScalingGroupName,
		DesiredCapacity:      aws.Int64(newDesiredCnt),
		MaxSize:              aws.Int64(newMaxSize),
	}
	_, err := a.asg.UpdateAutoScalingGroup(params)
	return err
}

// UpdateAutoScalingGroupMaxSize updates Auto Scaling Group's MaxSize.
func (a *App) UpdateAutoScalingGroupMaxSize(scalingGroup *autoscaling.Group, newMaxSize int64) error {
	params := &autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: scalingGroup.AutoScalingGroupName,
		MaxSize:              aws.Int64(newMaxSize),
	}
	_, err := a.asg.UpdateAutoScalingGroup(params)
	return err
}

// WaitContainerInstanceActive waits until container instance is ACTIVE.
func (a *App) WaitContainerInstanceActive(clusterArn string, newDesiredCnt int64, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	errc := make(chan error)

	go func() {
		for {
			params := &ecs.ListContainerInstancesInput{
				Cluster:    aws.String(clusterArn),
				Status:     aws.String("ACTIVE"),
				MaxResults: aws.Int64(100),
			}
			out, err := a.ecs.ListContainerInstances(params)
			if err != nil {
				errc <- err
				return
			}
			if len(out.ContainerInstanceArns) == int(newDesiredCnt) {
				errc <- nil
				return
			}
			log.Println("Waiting all container instances are ACTIVE")
			time.Sleep(SleepInterval)
		}
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// DrainTargetContainerInstance drains container instance.
func (a *App) DrainTargetContainerInstance(clusterArn string, containerInstanceArn string) error {
	_, err := a.ecs.UpdateContainerInstancesState(&ecs.UpdateContainerInstancesStateInput{
		Cluster:            aws.String(clusterArn),
		ContainerInstances: aws.StringSlice([]string{containerInstanceArn}),
		Status:             aws.String("DRAINING"),
	})
	return err
}

// WaitTargetContainerInstanceDrained waits until container instance drained.
func (a *App) WaitTargetContainerInstanceDrained(clusterArn string, containerInstanceArn string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	errc := make(chan error)

	go func() {
		for {
			params := &ecs.DescribeContainerInstancesInput{
				Cluster:            aws.String(clusterArn),
				ContainerInstances: []*string{&containerInstanceArn},
			}
			res, err := a.ecs.DescribeContainerInstances(params)
			if err != nil {
				errc <- err
				return
			}

			// Theoretically, we have only one container instance waiting to be drained.
			// But for safety, we walk through all retreived container instances.
			runingCnt := int64(0)
			for _, inst := range res.ContainerInstances {
				log.Printf(
					"Waiting running %d tasks in %s (%s) are STOPPED\n",
					*inst.RunningTasksCount, *inst.ContainerInstanceArn, *inst.Ec2InstanceId,
				)
				runingCnt += *inst.RunningTasksCount
			}

			if runingCnt == 0 {
				errc <- nil
				return
			} else {
				time.Sleep(SleepInterval)
			}
		}
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// WaitTasksMigration waits until cluster's services are all stable.
func (a *App) WaitTasksMigration(clusterArn string, timeout time.Duration, maxAttempts int) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	go func() {
		log.Println("Waiting all tasks are RUNNING")
		for {
			tick := time.Tick(SleepInterval)
			select {
			case <-ctx.Done():
				return
			case <-tick:
				log.Println("Waiting all tasks are RUNNING")
			}
		}
	}()

	var nextToken *string
	sArns := []*string{}
	for {
		res, err := a.ecs.ListServices(&ecs.ListServicesInput{
			Cluster:    aws.String(clusterArn),
			MaxResults: aws.Int64(100),
			LaunchType: aws.String("EC2"),
			NextToken:  nextToken,
		})
		if err != nil {
			return err
		}
		sArns = append(sArns, res.ServiceArns...)
		if nextToken == nil {
			break
		}
	}

	wparams := &ecs.DescribeServicesInput{
		Cluster:  aws.String(clusterArn),
		Services: sArns,
	}
	err := a.ecs.WaitUntilServicesStableWithContext(
		ctx,
		wparams,
		request.WithWaiterDelay(request.ConstantWaiterDelay(SleepInterval)),
		request.WithWaiterMaxAttempts(maxAttempts),
	)
	return err
}

// DeregisterContainerInstance deregisters target instance from ECS cluster.
func (a *App) DeregisterContainerInstance(clusterArn string, containerInstanceArn string) error {
	_, err := a.ecs.DeregisterContainerInstance(&ecs.DeregisterContainerInstanceInput{
		Cluster:           aws.String(clusterArn),
		ContainerInstance: aws.String(containerInstanceArn),
		Force:             aws.Bool(true),
	})
	return err
}

// TerminateInstance removes target instance from scale-in protection and decrement desired capacity.
func (a *App) TerminateInstance(scalingGroup *autoscaling.Group) error {
	_, err := a.asg.SetInstanceProtection(&autoscaling.SetInstanceProtectionInput{
		AutoScalingGroupName: scalingGroup.AutoScalingGroupName,
		InstanceIds:          aws.StringSlice([]string{a.instanceId}),
		ProtectedFromScaleIn: aws.Bool(false),
	})
	if err != nil {
		return err
	}

	_, err = a.asg.TerminateInstanceInAutoScalingGroup(&autoscaling.TerminateInstanceInAutoScalingGroupInput{
		InstanceId:                     aws.String(a.instanceId),
		ShouldDecrementDesiredCapacity: aws.Bool(true),
	})
	return err
}

// WaitInstanceTermination waits until target instance is terminated.
func (a *App) WaitInstanceTermination(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	go func() {
		log.Printf("Waiting instance %s to be terminated\n", a.instanceId)
		for {
			tick := time.Tick(SleepInterval)
			select {
			case <-ctx.Done():
				return
			case <-tick:
				log.Printf("Waiting instance %s to be terminated\n", a.instanceId)
			}
		}
	}()

	params := &ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{a.instanceId}),
	}
	return a.ec2.WaitUntilInstanceTerminatedWithContext(ctx, params)
}
