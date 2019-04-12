package cloudformation

// CreationPolicy prevents a resource status from reaching create complete until AWS CloudFormation receives a specified number of success signals or the timeout period is exceeded. To signal a resource, you can use the cfn-signal helper script or SignalResource API. AWS CloudFormation publishes valid signals to the stack events so that you track the number of signals sent.
type CreationPolicy struct {

	// AutoScalingCreationPolicy specifies how many instances must signal success for the update to succeed.
	AutoScalingCreationPolicy *AutoScalingCreationPolicy `json:"AutoScalingCreationPolicy,omitempty"`

	// ResourceSignal configures the number of required success signals and the length of time that AWS CloudFormation waits for those signals.
	ResourceSignal *ResourceSignal `json:"ResourceSignal,omitempty"`
}

// AutoScalingCreationPolicy specifies how many instances must signal success for the update to succeed.
type AutoScalingCreationPolicy struct {

	// MinSuccessfulInstancesPercent specifies the percentage of instances in an Auto Scaling replacement update that must signal success for the update to succeed. You can specify a value from 0 to 100. AWS CloudFormation rounds to the nearest tenth of a percent. For example, if you update five instances with a minimum successful percentage of 50, three instances must signal success. If an instance doesn't send a signal within the time specified by the Timeout property, AWS CloudFormation assumes that the instance wasn't created.
	MinSuccessfulInstancesPercent float64 `json:"MinSuccessfulInstancesPercent,omitempty"`
}

// ResourceSignal configures the number of required success signals and the length of time that AWS CloudFormation waits for those signals.
type ResourceSignal struct {

	// Count is the number of success signals AWS CloudFormation must receive before it sets the resource status as CREATE_COMPLETE. If the resource receives a failure signal or doesn't receive the specified number of signals before the timeout period expires, the resource creation fails and AWS CloudFormation rolls the stack back.
	Count float64 `json:"Count,omitempty"`

	// Timeout is the length of time that AWS CloudFormation waits for the number of signals that was specified in the Count property. The timeout period starts after AWS CloudFormation starts creating the resource, and the timeout expires no sooner than the time you specify but can occur shortly thereafter. The maximum time that you can specify is 12 hours.
	// The value must be in ISO8601 duration format, in the form: "PT#H#M#S", where each # is the number of hours, minutes, and seconds, respectively. For best results, specify a period of time that gives your instances plenty of time to get up and running. A shorter timeout can cause a rollback.
	Timeout string `json:"Timeout,omitempty"`
}

// DeletionPolicy can preserve or (in some cases) backup a resource when its stack is deleted. You specify a DeletionPolicy attribute for each resource that you want to control. If a resource has no DeletionPolicy attribute, AWS CloudFormation deletes the resource by default.
// Either "Delete", "Retain" or "Snapshot".
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
type DeletionPolicy string

// UpdatePolicy specifies how AWS CloudFormation handles updates to the AWS::AutoScaling::AutoScalingGroup or AWS::Lambda::Alias resource.
// For AWS::AutoScaling::AutoScalingGroup resources, AWS CloudFormation invokes one of three update policies depending on the type of change you make or whether a scheduled action is associated with the Auto Scaling group.
// The AutoScalingReplacingUpdate and AutoScalingRollingUpdate policies apply only when you do one or more of the following:
// - Change the Auto Scaling group's AWS::AutoScaling::LaunchConfiguration.
// - Change the Auto Scaling group's VPCZoneIdentifier property
// - Change the Auto Scaling group's LaunchTemplate property
// - Update an Auto Scaling group that contains instances that don't match the current LaunchConfiguration.
// If both the AutoScalingReplacingUpdate and AutoScalingRollingUpdate policies are specified, setting the WillReplace property to true gives AutoScalingReplacingUpdate precedence.
// The AutoScalingScheduledAction policy applies when you update a stack that includes an Auto Scaling group with an associated scheduled action.
// For AWS::Lambda::Alias resources, AWS CloudFormation performs an AWS CodeDeploy deployment when the version changes on the alias. For more information, see CodeDeployLambdaAliasUpdate Policy.
type UpdatePolicy struct {

	// AutoScalingReplacingUpdate specifies whether AWS CloudFormation replaces an Auto Scaling group with a new one or replaces only the instances in the Auto Scaling group.
	AutoScalingReplacingUpdate *AutoScalingReplacingUpdate `json:"AutoScalingReplacingUpdate,omitempty"`

	// AutoScalingRollingUpdate enable you to specify whether AWS CloudFormation updates instances that are in an Auto Scaling group in batches or all at once.
	AutoScalingRollingUpdate *AutoScalingRollingUpdate `json:"AutoScalingRollingUpdate,omitempty"`

	// AutoScalingScheduledAction specifies how AWS CloudFormation handles updates for the MinSize, MaxSize, and DesiredCapacity properties when the AWS::AutoScaling::AutoScalingGroup resource has an associated scheduled action.
	AutoScalingScheduledAction *AutoScalingScheduledAction `json:"AutoScalingScheduledAction,omitempty"`

	// CodeDeployLambdaAliasUpdate performs an AWS CodeDeploy deployment when the version changes on an AWS::Lambda::Alias resource.
	CodeDeployLambdaAliasUpdate *CodeDeployLambdaAliasUpdate `json:"CodeDeployLambdaAliasUpdate,omitempty"`
}

// AutoScalingScheduledAction specifies how AWS CloudFormation handles updates for the MinSize, MaxSize, and DesiredCapacity properties when the AWS::AutoScaling::AutoScalingGroup resource has an associated scheduled action, use the AutoScalingScheduledAction policy.
// With scheduled actions, the group size properties of an Auto Scaling group can change at any time. When you update a stack with an Auto Scaling group and scheduled action, AWS CloudFormation always sets the group size property values of your Auto Scaling group to the values that are defined in the AWS::AutoScaling::AutoScalingGroup resource of your template, even if a scheduled action is in effect.
// If you do not want AWS CloudFormation to change any of the group size property values when you have a scheduled action in effect, use the AutoScalingScheduledAction update policy to prevent AWS CloudFormation from changing the MinSize, MaxSize, or DesiredCapacity properties unless you have modified these values in your template.
type AutoScalingScheduledAction struct {
	// Specifies whether AWS CloudFormation ignores differences in group size properties between your current Auto Scaling group and the Auto Scaling group described in the AWS::AutoScaling::AutoScalingGroup resource of your template during a stack update. If you modify any of the group size property values in your template, AWS CloudFormation uses the modified values and updates your Auto Scaling group. (default: false)
	IgnoreUnmodifiedGroupSizeProperties bool `json:"IgnoreUnmodifiedGroupSizeProperties,omitempty"`
}

// AutoScalingReplacingUpdate specifies whether AWS CloudFormation replaces an Auto Scaling group with a new one or replaces only the instances in the Auto Scaling group.
type AutoScalingReplacingUpdate struct {
	WillReplace bool `json:"WillReplace,omitempty"`
}

// AutoScalingRollingUpdate enable you to specify whether AWS CloudFormation updates instances that are in an Auto Scaling group in batches or all at once.
type AutoScalingRollingUpdate struct {

	// MaxBatchSize specifies the maximum number of instances that AWS CloudFormation updates.
	MaxBatchSize float64 `json:"MaxBatchSize,omitempty"`

	// MinInstancesInService specifies the minimum number of instances that must be in service within the Auto Scaling group while AWS CloudFormation updates old instances.
	MinInstancesInService float64 `json:"MinInstancesInService,omitempty"`

	// MinSuccessfulInstancesPercent specifies the percentage of instances in an Auto Scaling rolling update that must signal success for an update to succeed. You can specify a value from 0 to 100. AWS CloudFormation rounds to the nearest tenth of a percent. For example, if you update five instances with a minimum successful percentage of 50, three instances must signal success.
	MinSuccessfulInstancesPercent float64 `json:"MinSuccessfulInstancesPercent,omitempty"`

	// PauseTime is the amount of time that AWS CloudFormation pauses after making a change to a batch of instances to give those instances time to start software applications. For example, you might need to specify PauseTime when scaling up the number of instances in an Auto Scaling group.
	PauseTime string `json:"PauseTime,omitempty"`

	// SuspendProcesses specifies the Auto Scaling processes to suspend during a stack update. Suspending processes prevents Auto Scaling from interfering with a stack update. For example, you can suspend alarming so that Amazon EC2 Auto Scaling doesn't execute scaling policies associated with an alarm. For valid values, see the ScalingProcesses.member.N parameter for the SuspendProcesses action in the Amazon EC2 Auto Scaling API Reference.
	SuspendProcesses []string `json:"SuspendProcesses,omitempty"`

	// WaitOnResourceSignals specifies whether the Auto Scaling group waits on signals from new instances during an update. Use this property to ensure that instances have completed installing and configuring applications before the Auto Scaling group update proceeds. AWS CloudFormation suspends the update of an Auto Scaling group after new EC2 instances are launched into the group. AWS CloudFormation must receive a signal from each new instance within the specified PauseTime before continuing the update. To signal the Auto Scaling group, use the cfn-signal helper script or SignalResource API.
	WaitOnResourceSignals bool `json:"WaitOnResourceSignals,omitempty"`
}

// CodeDeployLambdaAliasUpdate performs an AWS CodeDeploy deployment when the version changes on an AWS::Lambda::Alias resource.
type CodeDeployLambdaAliasUpdate struct {

	// AfterAllowTrafficHook is the name of the Lambda function to run after traffic routing completes.
	AfterAllowTrafficHook string `json:"AfterAllowTrafficHook,omitempty"`

	// ApplicationName is the name of the AWS CodeDeploy application.
	ApplicationName string `json:"ApplicationName"`

	// BeforeAllowTrafficHook is the name of the Lambda function to run before traffic routing starts.
	BeforeAllowTrafficHook string `json:"BeforeAllowTrafficHook,omitempty"`

	// DeploymentGroupName is the name of the AWS CodeDeploy deployment group. This is where the traffic-shifting policy is set.
	DeploymentGroupName string `json:"DeploymentGroupName"`
}
