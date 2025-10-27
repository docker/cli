package progress

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/moby/moby/client/pkg/progress"
	"github.com/moby/moby/client/pkg/streamformatter"
)

var (
	numberedStates = map[swarm.TaskState]int64{
		swarm.TaskStateNew:       1,
		swarm.TaskStateAllocated: 2,
		swarm.TaskStatePending:   3,
		swarm.TaskStateAssigned:  4,
		swarm.TaskStateAccepted:  5,
		swarm.TaskStatePreparing: 6,
		swarm.TaskStateReady:     7,
		swarm.TaskStateStarting:  8,
		swarm.TaskStateRunning:   9,

		// The following states are not actually shown in progress
		// output, but are used internally for ordering.
		swarm.TaskStateComplete: 10,
		swarm.TaskStateShutdown: 11,
		swarm.TaskStateFailed:   12,
		swarm.TaskStateRejected: 13,
	}

	longestState int
)

const (
	maxProgress     = 9
	maxProgressBars = 20
	maxJobProgress  = 10
)

type progressUpdater interface {
	update(service swarm.Service, tasks []swarm.Task, activeNodes map[string]struct{}, rollback bool) (bool, error)
}

func init() {
	for state := range numberedStates {
		// for jobs, we use the "complete" state, and so it should be factored
		// in to the computation of the longest state.
		if (!terminalState(state) || state == swarm.TaskStateComplete) && len(state) > longestState {
			longestState = len(state)
		}
	}
}

func terminalState(state swarm.TaskState) bool {
	return numberedStates[state] > numberedStates[swarm.TaskStateRunning]
}

// ServiceProgress outputs progress information for convergence of a service.
//
//nolint:gocyclo
func ServiceProgress(ctx context.Context, apiClient client.APIClient, serviceID string, progressWriter io.WriteCloser) error {
	defer progressWriter.Close()

	progressOut := streamformatter.NewJSONProgressOutput(progressWriter, false)

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	defer signal.Stop(sigint)

	var (
		updater     progressUpdater
		converged   bool
		convergedAt time.Time
		monitor     = 5 * time.Second
		rollback    bool
		message     *progress.Progress
	)

	for {
		res, err := apiClient.ServiceInspect(ctx, serviceID, client.ServiceInspectOptions{})
		if err != nil {
			return err
		}

		if res.Service.Spec.UpdateConfig != nil && res.Service.Spec.UpdateConfig.Monitor != 0 {
			monitor = res.Service.Spec.UpdateConfig.Monitor
		}

		if updater == nil {
			updater, err = initializeUpdater(res.Service, progressOut)
			if err != nil {
				return err
			}
		}

		if res.Service.UpdateStatus != nil {
			switch res.Service.UpdateStatus.State {
			case swarm.UpdateStateUpdating:
				rollback = false
			case swarm.UpdateStateCompleted:
				if !converged {
					return nil
				}
			case swarm.UpdateStatePaused:
				return fmt.Errorf("service update paused: %s", res.Service.UpdateStatus.Message)
			case swarm.UpdateStateRollbackStarted:
				if !rollback && res.Service.UpdateStatus.Message != "" {
					progressOut.WriteProgress(progress.Progress{
						ID:     "rollback",
						Action: res.Service.UpdateStatus.Message,
					})
				}
				rollback = true
			case swarm.UpdateStateRollbackPaused:
				return fmt.Errorf("service rollback paused: %s", res.Service.UpdateStatus.Message)
			case swarm.UpdateStateRollbackCompleted:
				if !converged {
					message = &progress.Progress{ID: "rollback", Message: res.Service.UpdateStatus.Message}
				}
				rollback = true
			}
		}
		if converged && time.Since(convergedAt) >= monitor {
			_ = progressOut.WriteProgress(progress.Progress{
				ID:     "verify",
				Action: fmt.Sprintf("Service %s converged", serviceID),
			})
			if message != nil {
				_ = progressOut.WriteProgress(*message)
			}
			return nil
		}

		tasks, err := apiClient.TaskList(ctx, client.TaskListOptions{
			Filters: make(client.Filters).Add("service", res.Service.ID).Add("_up-to-date", "true"),
		})
		if err != nil {
			return err
		}

		activeNodes, err := getActiveNodes(ctx, apiClient)
		if err != nil {
			return err
		}

		converged, err = updater.update(res.Service, tasks.Items, activeNodes, rollback)
		if err != nil {
			return err
		}
		if converged {
			// if the service is a job, there's no need to verify it. jobs are
			// stay done once they're done. skip the verification and just end
			// the progress monitoring.
			//
			// only job services have a non-nil job status, which means we can
			// use the presence of this field to check if the service is a job
			// here.
			if res.Service.JobStatus != nil {
				progress.Message(progressOut, "", "job complete")
				return nil
			}

			if convergedAt.IsZero() {
				convergedAt = time.Now()
			}
			wait := monitor - time.Since(convergedAt)
			if wait >= 0 {
				_ = progressOut.WriteProgress(progress.Progress{
					// Ideally this would have no ID, but
					// the progress rendering code behaves
					// poorly on an "action" with no ID. It
					// returns the cursor to the beginning
					// of the line, so the first character
					// may be difficult to read. Then the
					// output is overwritten by the shell
					// prompt when the command finishes.
					ID:     "verify",
					Action: fmt.Sprintf("Waiting %d seconds to verify that tasks are stable...", wait/time.Second+1),
				})
			}
		} else {
			if !convergedAt.IsZero() {
				_ = progressOut.WriteProgress(progress.Progress{
					ID:     "verify",
					Action: "Detected task failure",
				})
			}
			convergedAt = time.Time{}
		}

		select {
		case <-time.After(200 * time.Millisecond):
		case <-sigint:
			if !converged {
				progress.Message(progressOut, "", "Operation continuing in background.")
				progress.Messagef(progressOut, "", "Use `docker service ps %s` to check progress.", serviceID)
			}
			return nil
		}
	}
}

// getActiveNodes returns all nodes that are currently not in status [swarm.NodeStateDown].
//
// TODO(thaJeztah): this should really be a filter on [apiClient.NodeList] instead of being filtered on the client side.
func getActiveNodes(ctx context.Context, apiClient client.NodeAPIClient) (map[string]struct{}, error) {
	res, err := apiClient.NodeList(ctx, client.NodeListOptions{})
	if err != nil {
		return nil, err
	}

	activeNodes := make(map[string]struct{})
	for _, n := range res.Items {
		if n.Status.State != swarm.NodeStateDown {
			activeNodes[n.ID] = struct{}{}
		}
	}
	return activeNodes, nil
}

func initializeUpdater(service swarm.Service, progressOut progress.Output) (progressUpdater, error) {
	if service.Spec.Mode.Replicated != nil && service.Spec.Mode.Replicated.Replicas != nil {
		return &replicatedProgressUpdater{
			progressOut: progressOut,
		}, nil
	}
	if service.Spec.Mode.Global != nil {
		return &globalProgressUpdater{
			progressOut: progressOut,
		}, nil
	}
	if service.Spec.Mode.ReplicatedJob != nil {
		return newReplicatedJobProgressUpdater(service, progressOut), nil
	}
	if service.Spec.Mode.GlobalJob != nil {
		return &globalJobProgressUpdater{
			progressOut: progressOut,
		}, nil
	}
	return nil, errors.New("unrecognized service mode")
}

func writeOverallProgress(progressOut progress.Output, numerator, denominator int, rollback bool) {
	if rollback {
		progressOut.WriteProgress(progress.Progress{
			ID:     "overall progress",
			Action: fmt.Sprintf("rolling back update: %d out of %d tasks", numerator, denominator),
		})
		return
	}
	progressOut.WriteProgress(progress.Progress{
		ID:     "overall progress",
		Action: fmt.Sprintf("%d out of %d tasks", numerator, denominator),
	})
}

func truncError(errMsg string) string {
	// Remove newlines from the error, which corrupt the output.
	errMsg = strings.ReplaceAll(errMsg, "\n", " ")

	// Limit the length to 75 characters, so that even on narrow terminals
	// this will not overflow to the next line.
	const maxWidth = 75
	if len(errMsg) > maxWidth {
		errMsg = errMsg[:maxWidth-1] + "…"
	}
	return errMsg
}

type replicatedProgressUpdater struct {
	progressOut progress.Output

	// used for mapping slots to a contiguous space
	// this also causes progress bars to appear in order
	slotMap map[int]int

	initialized bool
	done        bool
}

func (u *replicatedProgressUpdater) update(service swarm.Service, tasks []swarm.Task, activeNodes map[string]struct{}, rollback bool) (bool, error) {
	if service.Spec.Mode.Replicated == nil || service.Spec.Mode.Replicated.Replicas == nil {
		return false, errors.New("no replica count")
	}
	replicas := *service.Spec.Mode.Replicated.Replicas

	if !u.initialized {
		u.slotMap = make(map[int]int)

		// Draw progress bars in order
		writeOverallProgress(u.progressOut, 0, int(replicas), rollback)

		if replicas <= maxProgressBars {
			for i := uint64(1); i <= replicas; i++ {
				progress.Update(u.progressOut, fmt.Sprintf("%d/%d", i, replicas), " ")
			}
		}
		u.initialized = true
	}

	tasksBySlot := u.tasksBySlot(tasks, activeNodes)

	// If we had reached a converged state, check if we are still converged.
	if u.done {
		for _, task := range tasksBySlot {
			if task.Status.State != swarm.TaskStateRunning {
				u.done = false
				break
			}
		}
	}

	running := uint64(0)

	for _, task := range tasksBySlot {
		mappedSlot := u.slotMap[task.Slot]
		if mappedSlot == 0 {
			mappedSlot = len(u.slotMap) + 1
			u.slotMap[task.Slot] = mappedSlot
		}

		if !terminalState(task.DesiredState) && task.Status.State == swarm.TaskStateRunning {
			running++
		}

		u.writeTaskProgress(task, mappedSlot, replicas)
	}

	if !u.done {
		writeOverallProgress(u.progressOut, int(running), int(replicas), rollback)

		if running == replicas {
			u.done = true
		}
	}

	return running == replicas, nil
}

func (*replicatedProgressUpdater) tasksBySlot(tasks []swarm.Task, activeNodes map[string]struct{}) map[int]swarm.Task {
	// If there are multiple tasks with the same slot number, favor the one
	// with the *lowest* desired state. This can happen in restart
	// scenarios.
	tasksBySlot := make(map[int]swarm.Task)
	for _, task := range tasks {
		if numberedStates[task.DesiredState] == 0 || numberedStates[task.Status.State] == 0 {
			continue
		}
		if existingTask, ok := tasksBySlot[task.Slot]; ok {
			if numberedStates[existingTask.DesiredState] < numberedStates[task.DesiredState] {
				continue
			}
			// If the desired states match, observed state breaks
			// ties. This can happen with the "start first" service
			// update mode.
			if numberedStates[existingTask.DesiredState] == numberedStates[task.DesiredState] &&
				numberedStates[existingTask.Status.State] <= numberedStates[task.Status.State] {
				continue
			}
		}
		if task.NodeID != "" {
			if _, nodeActive := activeNodes[task.NodeID]; !nodeActive {
				continue
			}
		}
		tasksBySlot[task.Slot] = task
	}

	return tasksBySlot
}

func (u *replicatedProgressUpdater) writeTaskProgress(task swarm.Task, mappedSlot int, replicas uint64) {
	if u.done || replicas > maxProgressBars || uint64(mappedSlot) > replicas {
		return
	}

	if task.Status.Err != "" {
		u.progressOut.WriteProgress(progress.Progress{
			ID:     fmt.Sprintf("%d/%d", mappedSlot, replicas),
			Action: truncError(task.Status.Err),
		})
		return
	}

	if !terminalState(task.DesiredState) && !terminalState(task.Status.State) {
		u.progressOut.WriteProgress(progress.Progress{
			ID:         fmt.Sprintf("%d/%d", mappedSlot, replicas),
			Action:     fmt.Sprintf("%-[1]*s", longestState, task.Status.State),
			Current:    numberedStates[task.Status.State],
			Total:      maxProgress,
			HideCounts: true,
		})
	}
}

type globalProgressUpdater struct {
	progressOut progress.Output

	initialized bool
	done        bool
}

func (u *globalProgressUpdater) update(_ swarm.Service, tasks []swarm.Task, activeNodes map[string]struct{}, rollback bool) (bool, error) {
	tasksByNode := u.tasksByNode(tasks)

	// We don't have perfect knowledge of how many nodes meet the
	// constraints for this service. But the orchestrator creates tasks
	// for all eligible nodes at the same time, so we should see all those
	// nodes represented among the up-to-date tasks.
	nodeCount := len(tasksByNode)

	if !u.initialized {
		if nodeCount == 0 {
			// Two possibilities: either the orchestrator hasn't created
			// the tasks yet, or the service doesn't meet constraints for
			// any node. Either way, we wait.
			u.progressOut.WriteProgress(progress.Progress{
				ID:     "overall progress",
				Action: "waiting for new tasks",
			})
			return false, nil
		}

		writeOverallProgress(u.progressOut, 0, nodeCount, rollback)
		u.initialized = true
	}

	// If we had reached a converged state, check if we are still converged.
	if u.done {
		for _, task := range tasksByNode {
			if task.Status.State != swarm.TaskStateRunning {
				u.done = false
				break
			}
		}
	}

	running := 0

	for _, task := range tasksByNode {
		if _, nodeActive := activeNodes[task.NodeID]; nodeActive {
			if !terminalState(task.DesiredState) && task.Status.State == swarm.TaskStateRunning {
				running++
			}

			u.writeTaskProgress(task, nodeCount)
		}
	}

	if !u.done {
		writeOverallProgress(u.progressOut, running, nodeCount, rollback)

		if running == nodeCount {
			u.done = true
		}
	}

	return running == nodeCount, nil
}

func (*globalProgressUpdater) tasksByNode(tasks []swarm.Task) map[string]swarm.Task {
	// If there are multiple tasks with the same node ID, favor the one
	// with the *lowest* desired state. This can happen in restart
	// scenarios.
	tasksByNode := make(map[string]swarm.Task)
	for _, task := range tasks {
		if numberedStates[task.DesiredState] == 0 || numberedStates[task.Status.State] == 0 {
			continue
		}
		if existingTask, ok := tasksByNode[task.NodeID]; ok {
			if numberedStates[existingTask.DesiredState] < numberedStates[task.DesiredState] {
				continue
			}

			// If the desired states match, observed state breaks
			// ties. This can happen with the "start first" service
			// update mode.
			if numberedStates[existingTask.DesiredState] == numberedStates[task.DesiredState] &&
				numberedStates[existingTask.Status.State] <= numberedStates[task.Status.State] {
				continue
			}
		}
		tasksByNode[task.NodeID] = task
	}

	return tasksByNode
}

func (u *globalProgressUpdater) writeTaskProgress(task swarm.Task, nodeCount int) {
	if u.done || nodeCount > maxProgressBars {
		return
	}

	if task.Status.Err != "" {
		u.progressOut.WriteProgress(progress.Progress{
			ID:     formatter.TruncateID(task.NodeID),
			Action: truncError(task.Status.Err),
		})
		return
	}

	if !terminalState(task.DesiredState) && !terminalState(task.Status.State) {
		u.progressOut.WriteProgress(progress.Progress{
			ID:         formatter.TruncateID(task.NodeID),
			Action:     fmt.Sprintf("%-[1]*s", longestState, task.Status.State),
			Current:    numberedStates[task.Status.State],
			Total:      maxProgress,
			HideCounts: true,
		})
	}
}

// replicatedJobProgressUpdater outputs the progress of a replicated job. This
// progress consists of a few main elements.
//
// The first is the progress bar for the job as a whole. This shows the number
// of completed out of total tasks for the job. Tasks that are currently
// running are not counted.
//
// The second is the status of the "active" tasks for the job. We count a task
// as "active" if it has any non-terminal state, not just running. This is
// shown as a fraction of the maximum concurrent tasks that can be running,
// which is the less of MaxConcurrent or TotalCompletions - completed tasks.
type replicatedJobProgressUpdater struct {
	progressOut progress.Output

	// jobIteration is the service's job iteration, used to exclude tasks
	// belonging to earlier iterations.
	jobIteration uint64

	// concurrent is the value of MaxConcurrent as an int. That is, the maximum
	// number of tasks allowed to be run simultaneously.
	concurrent int

	// total is the value of TotalCompletions, the number of complete tasks
	// desired.
	total int

	// initialized is set to true after the first time update is called. the
	// first time update is called, the components of the progress UI are all
	// written out in an initial pass. this ensure that they will subsequently
	// be in order, no matter how they are updated.
	initialized bool

	// progressDigits is the number digits in total, so that we know how much
	// to pad the job progress field with.
	//
	// when we're writing the number of completed over total tasks, we need to
	// pad the numerator with spaces, so that the bar doesn't jump around.
	// we'll compute that once on init, and then reuse it over and over.
	//
	// we compute this in the least clever way possible: convert to string
	// with strconv.Itoa, then take the len.
	progressDigits int

	// activeDigits is the same, but for active tasks, and it applies to both
	// the numerator and denominator.
	activeDigits int
}

func newReplicatedJobProgressUpdater(service swarm.Service, progressOut progress.Output) *replicatedJobProgressUpdater {
	concurrent := int(*service.Spec.Mode.ReplicatedJob.MaxConcurrent)
	total := int(*service.Spec.Mode.ReplicatedJob.TotalCompletions)

	return &replicatedJobProgressUpdater{
		progressOut:    progressOut,
		concurrent:     concurrent,
		total:          total,
		jobIteration:   service.JobStatus.JobIteration.Index,
		progressDigits: len(strconv.Itoa(total)),
		activeDigits:   len(strconv.Itoa(concurrent)),
	}
}

// update writes out the progress of the replicated job.
func (u *replicatedJobProgressUpdater) update(_ swarm.Service, tasks []swarm.Task, _ map[string]struct{}, _ bool) (bool, error) {
	if !u.initialized {
		u.writeOverallProgress(0, 0)

		// only write out progress bars if there will be less than the maximum
		if u.total <= maxProgressBars {
			for i := 1; i <= u.total; i++ {
				u.progressOut.WriteProgress(progress.Progress{
					ID:     fmt.Sprintf("%d/%d", i, u.total),
					Action: " ",
				})
			}
		}
		u.initialized = true
	}

	// tasksBySlot is a mapping of slot number to the task valid for that slot.
	// it deduplicated tasks occupying the same numerical slot but in different
	// states.
	tasksBySlot := make(map[int]swarm.Task)
	for _, task := range tasks {
		// first, check if the task belongs to this service iteration. skip
		// tasks belonging to other iterations.
		if task.JobIteration == nil || task.JobIteration.Index != u.jobIteration {
			continue
		}

		// then, if the task is in an unknown state, ignore it.
		if numberedStates[task.DesiredState] == 0 ||
			numberedStates[task.Status.State] == 0 {
			continue
		}

		// finally, check if the task already exists in the map
		if existing, ok := tasksBySlot[task.Slot]; ok {
			// if so, use the task with the lower actual state
			if numberedStates[existing.Status.State] > numberedStates[task.Status.State] {
				tasksBySlot[task.Slot] = task
			}
		} else {
			// otherwise, just add it to the map.
			tasksBySlot[task.Slot] = task
		}
	}

	activeTasks := 0
	completeTasks := 0

	for i := 0; i < len(tasksBySlot); i++ {
		task := tasksBySlot[i]
		u.writeTaskProgress(task)

		if numberedStates[task.Status.State] < numberedStates[swarm.TaskStateComplete] {
			activeTasks++
		}

		if task.Status.State == swarm.TaskStateComplete {
			completeTasks++
		}
	}

	u.writeOverallProgress(activeTasks, completeTasks)

	return completeTasks == u.total, nil
}

func (u *replicatedJobProgressUpdater) writeOverallProgress(active, completed int) {
	_ = u.progressOut.WriteProgress(progress.Progress{
		ID: "job progress",
		Action: fmt.Sprintf(
			// * means "use the next positional arg to compute padding"
			"%*d out of %d complete", u.progressDigits, completed, u.total,
		),
		Current:    int64(completed),
		Total:      int64(u.total),
		HideCounts: true,
	})

	// actualDesired is the lesser of MaxConcurrent, or the remaining tasks
	actualDesired := u.total - completed
	if actualDesired > u.concurrent {
		actualDesired = u.concurrent
	}

	_ = u.progressOut.WriteProgress(progress.Progress{
		ID: "active tasks",
		Action: fmt.Sprintf(
			// [n] notation lets us select a specific argument, 1-indexed
			// putting the [1] before the star means "make the string this
			// length". putting the [2] or the [3] means "use this argument
			// here"
			//
			// we pad both the numerator and the denominator because, as the
			// job reaches its conclusion, the number of possible concurrent
			// tasks will go down, as fewer than MaxConcurrent tasks are needed
			// to complete the job.
			"%[1]*[2]d out of %[1]*[3]d tasks", u.activeDigits, active, actualDesired,
		),
	})
}

func (u *replicatedJobProgressUpdater) writeTaskProgress(task swarm.Task) {
	if u.total > maxProgressBars {
		return
	}

	if task.Status.Err != "" {
		_ = u.progressOut.WriteProgress(progress.Progress{
			ID:     fmt.Sprintf("%d/%d", task.Slot+1, u.total),
			Action: truncError(task.Status.Err),
		})
		return
	}

	_ = u.progressOut.WriteProgress(progress.Progress{
		ID:         fmt.Sprintf("%d/%d", task.Slot+1, u.total),
		Action:     fmt.Sprintf("%-*s", longestState, task.Status.State),
		Current:    numberedStates[task.Status.State],
		Total:      maxJobProgress,
		HideCounts: true,
	})
}

// globalJobProgressUpdater is the progressUpdater for GlobalJob-mode services.
// Because GlobalJob services are so much simpler than ReplicatedJob services,
// this updater is in turn simpler as well.
type globalJobProgressUpdater struct {
	progressOut progress.Output

	// initialized is used to detect the first pass of update, and to perform
	// first time initialization logic at that time.
	initialized bool

	// total is the total number of tasks expected for this job
	total int

	// progressDigits is the number of spaces to pad the numerator of the job
	// progress field
	progressDigits int

	taskNodes map[string]struct{}
}

func (u *globalJobProgressUpdater) update(service swarm.Service, tasks []swarm.Task, activeNodes map[string]struct{}, _ bool) (bool, error) {
	if !u.initialized {
		// if there are not yet tasks, then return early.
		if len(tasks) == 0 && len(activeNodes) != 0 {
			_ = u.progressOut.WriteProgress(progress.Progress{
				ID:     "job progress",
				Action: "waiting for tasks",
			})
			return false, nil
		}

		// when a global job starts, all of its tasks are created at once, so
		// we can use len(tasks) to know how many we're expecting.
		u.taskNodes = map[string]struct{}{}

		for _, task := range tasks {
			// skip any tasks not belonging to this job iteration.
			if task.JobIteration == nil || task.JobIteration.Index != service.JobStatus.JobIteration.Index {
				continue
			}

			// collect the list of all node IDs for this service.
			//
			// basically, global jobs will execute on any new nodes that join
			// the cluster in the future. to avoid making things complicated,
			// we will only check the progress of the initial set of nodes. if
			// any new nodes come online during the operation, we will ignore
			// them.
			u.taskNodes[task.NodeID] = struct{}{}
		}

		u.total = len(u.taskNodes)
		u.progressDigits = len(strconv.Itoa(u.total))

		u.writeOverallProgress(0)
		u.initialized = true
	}

	// tasksByNodeID maps a NodeID to the latest task for that Node ID. this
	// lets us pick only the latest task for any given node.
	tasksByNodeID := map[string]swarm.Task{}

	for _, task := range tasks {
		// skip any tasks not belonging to this job iteration
		if task.JobIteration == nil || task.JobIteration.Index != service.JobStatus.JobIteration.Index {
			continue
		}

		// if the task is not on one of the initial set of nodes, ignore it.
		if _, ok := u.taskNodes[task.NodeID]; !ok {
			continue
		}

		// if there is already a task recorded for this node, choose the one
		// with the lower state
		if oldtask, ok := tasksByNodeID[task.NodeID]; ok {
			if numberedStates[oldtask.Status.State] > numberedStates[task.Status.State] {
				tasksByNodeID[task.NodeID] = task
			}
		} else {
			tasksByNodeID[task.NodeID] = task
		}
	}

	complete := 0
	for _, task := range tasksByNodeID {
		u.writeTaskProgress(task)
		if task.Status.State == swarm.TaskStateComplete {
			complete++
		}
	}

	u.writeOverallProgress(complete)
	return complete == u.total, nil
}

func (u *globalJobProgressUpdater) writeTaskProgress(task swarm.Task) {
	if u.total > maxProgressBars {
		return
	}

	if task.Status.Err != "" {
		_ = u.progressOut.WriteProgress(progress.Progress{
			ID:     task.NodeID,
			Action: truncError(task.Status.Err),
		})
		return
	}

	_ = u.progressOut.WriteProgress(progress.Progress{
		ID:         task.NodeID,
		Action:     fmt.Sprintf("%-*s", longestState, task.Status.State),
		Current:    numberedStates[task.Status.State],
		Total:      maxJobProgress,
		HideCounts: true,
	})
}

func (u *globalJobProgressUpdater) writeOverallProgress(complete int) {
	// all tasks for a global job are active at once, so we only write out the
	// total progress.
	_ = u.progressOut.WriteProgress(progress.Progress{
		// see (*replicatedJobProgressUpdater).writeOverallProgress for an
		// explanation of the advanced fmt use in this function.
		ID: "job progress",
		Action: fmt.Sprintf(
			"%*d out of %d complete", u.progressDigits, complete, u.total,
		),
		Current:    int64(complete),
		Total:      int64(u.total),
		HideCounts: true,
	})
}
