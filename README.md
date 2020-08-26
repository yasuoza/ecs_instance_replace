# ecs_instance_replace

Replace EC2(ECS Container) instance which is belongs to Auto Scaling Group.

## Usage

```bash
$ ecs_instance_replace replace i-0d0123456789abcde

2020/08/01 15:38:40 Target Cluster: arn:aws:ecs:us-west-2:123456789012:cluster/ecs-cluster
2020/08/01 15:38:40 Target ASG: ecs-cluster-ecs-instance-asg
2020/08/01 15:38:41 Auto Scaling group ecs-cluster-ecs-instance-asg is updated to desired_capacity: 3, max_size: 3
2020/08/01 15:38:41 Waiting all container instances are ACTIVE
2020/08/01 15:38:57 Waiting all container instances are ACTIVE
2020/08/01 15:39:12 Waiting all container instances are ACTIVE
2020/08/01 15:39:28 Waiting all container instances are ACTIVE
2020/08/01 15:39:44 Waiting all container instances are ACTIVE
2020/08/01 15:39:59 Container instance arn:aws:ecs:us-west-2:123456789012:container-instance/afa9cf13-9dd6-4de2-8c84-454c64746cc7 is marked DRAINING
2020/08/01 15:39:59 Waiting running 6 tasks in arn:aws:ecs:us-west-2:123456789012:container-instance/afa9cf13-9dd6-4de2-8c84-454c64746cc7 (i-0d0123456789abcde) are STOPPED
2020/08/01 15:40:15 Waiting running 0 tasks in arn:aws:ecs:us-west-2:123456789012:container-instance/afa9cf13-9dd6-4de2-8c84-454c64746cc7 (i-0d0123456789abcde) are STOPPED
2020/08/01 15:40:15 Waiting all tasks are RUNNING
2020/08/01 15:40:30 Waiting all tasks are RUNNING
2020/08/01 15:40:34 Desired capacity of ecs-cluster-ecs-instance-asg is reverted
2020/08/01 15:40:34 Auto Scaling group ecs-cluster-ecs-instance-asg is updated to desired_capacity: 2, max_size: 2
2020/08/01 15:40:34 Waiting instance i-0d0123456789abcde to be terminated
2020/08/01 15:40:49 Waiting instance i-0d0123456789abcde to be terminated
2020/08/01 15:41:04 Waiting instance i-0d0123456789abcde to be terminated
2020/08/01 15:41:19 Waiting instance i-0d0123456789abcde to be terminated
2020/08/01 15:41:21 i-0d0123456789abcde is replaced by i-0f0123456789abcde
```

## LICENSE

MIT.
