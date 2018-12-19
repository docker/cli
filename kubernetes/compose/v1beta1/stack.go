package v1beta1

import api "github.com/docker/compose-on-kubernetes/api/compose/v1beta1"

// StackList defines a list of stacks
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta1.StackList instead
type StackList = api.StackList

// Stack defines a stack object to be register in the kubernetes API
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta1.Stack instead
type Stack = api.Stack

// StackSpec defines the desired state of Stack
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta1.StackSpec instead
type StackSpec = api.StackSpec

// StackPhase defines the status phase in which the stack is.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta1.StackPhase instead
type StackPhase = api.StackPhase

// These are valid conditions of a stack.
const (
	// StackAvailable means the stack is available.
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta1.StackAvailable instead
	StackAvailable StackPhase = api.StackAvailable
	// StackProgressing means the deployment is progressing.
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta1.StackProgressing instead
	StackProgressing StackPhase = api.StackProgressing
	// StackFailure is added in a stack when one of its members fails to be created
	// or deleted.
	// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta1.StackFailure instead
	StackFailure StackPhase = api.StackFailure
)

// StackStatus defines the observed state of Stack
// Deprecated: Use github.com/docker/compose-on-kubernetes/api/compose/v1beta1.StackStatus instead
type StackStatus = api.StackStatus
