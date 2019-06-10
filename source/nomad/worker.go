package nomad

import (
	"fmt"
	"strings"
	"sync"
	"time"

	consulAPI "github.com/hashicorp/consul/api"
	nomadAPI "github.com/hashicorp/nomad/api"
	"github.com/mitchellh/hashstructure"
	"github.com/pm-connect/log-shipper/message"
)

type AllocationWorker struct {
	Stop chan struct{}

	allocation nomadAPI.Allocation

	nomadClient  *nomadAPI.Client
	consulClient *consulAPI.Client

	waitGroup *sync.WaitGroup

	allocationConfig *AllocationConfig

	clusterName string
}

func NewAllocationWorker(nomadClient *nomadAPI.Client, consulClient *consulAPI.Client, clusterName string, alloc nomadAPI.Allocation, stop chan struct{}) *AllocationWorker {
	return &AllocationWorker{
		Stop:         stop,
		allocation:   alloc,
		nomadClient:  nomadClient,
		consulClient: consulClient,
		waitGroup:    &sync.WaitGroup{},
		clusterName: clusterName,
	}
}

func (w *AllocationWorker) Start(receiver chan<- *message.SourceMessage) {
	gotConfig := w.updateAllocationConfig()

	if !w.allocationConfig.JobConfig.Enabled || !gotConfig {
		return
	}

	go w.syncAllocationConfig()
	go w.watchAllocationEvents(receiver)
	go w.watchTasks(receiver)
	go w.watchGroups(receiver)
}

func (w *AllocationWorker) Wait() {
	w.waitGroup.Wait()
}

func (w *AllocationWorker) updateAllocationConfig() bool {
	alloc, _, err := w.nomadClient.Allocations().Info(w.allocation.ID, nil)
	if err != nil {
		return false
	}

	config := NewFromAllocation(alloc)

	w.allocationConfig = config

	return true
}

func (w *AllocationWorker) syncAllocationConfig() {
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ticker.C:
			w.updateAllocationConfig()
		case <-w.Stop:
			return
		}
	}
}

func (w *AllocationWorker) watchTasks(receiver chan<- *message.SourceMessage) {

}

func (w *AllocationWorker) watchGroups(receiver chan<- *message.SourceMessage) {

}

func (w *AllocationWorker) watchAllocationEvents(receiver chan<- *message.SourceMessage) {
	w.waitGroup.Add(1)
	defer w.waitGroup.Done()

	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ticker.C:
			gotConfig := w.updateAllocationConfig()
			if !gotConfig {
				return
			}

			alloc := w.allocationConfig.Alloc

			for task, state := range alloc.TaskStates {
				taskKey := fmt.Sprintf("%s/%s/%s", *alloc.Job.Name, alloc.TaskGroup, task)
				if !w.allocationConfig.TaskConfigs[taskKey].Enabled || !w.allocationConfig.TaskConfigs[taskKey].WatchNomadEvents {
					continue
				}

				for _, event := range state.Events {
					hash, err := hashstructure.Hash(event, nil)

					if err != nil {
						panic(err)
					}

					key := fmt.Sprintf("log-shipper/nomad/synced-events/%s/allocs/%s/%d", w.clusterName, alloc.ID, hash)

					existing, _, _ := w.consulClient.KV().Get(key, nil)

					if existing != nil {
						continue
					}

					t := time.Unix(0, event.Time)

					log := &message.SourceMessage{
						Id:      *alloc.Job.ID,
						Message: []byte(strings.TrimSpace(fmt.Sprintf("[%s] %s %s", event.Type, event.DriverMessage, event.DriverError))),
						Meta: map[string]string{
							"task":           task,
							"alloc":          alloc.ID,
							"job":            *alloc.Job.ID,
							"group":          alloc.TaskGroup,
							"event_type":     event.Type,
							"driver_message": event.DriverMessage,
							"driver_error":   event.DriverError,
						},
						Attributes: &message.Attributes{
							Type:      "nomad_task_events",
							Timestamp: t.Unix(),
						},
					}

					receiver <- log

					_, _ = w.consulClient.KV().Put(&consulAPI.KVPair{
						Key:   key,
						Value: []byte(""),
					}, nil)
				}

			}
		case <-w.Stop:
			return
		}
	}
}
