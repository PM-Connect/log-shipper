package nomad

import (
	"fmt"
	"github.com/pm-connect/log-shipper/message"
	"sync"
	"time"

	consulAPI "github.com/hashicorp/consul/api"
	nomadAPI "github.com/hashicorp/nomad/api"
)

type AllocationWorker struct {
	Stop chan struct{}

	allocation nomadAPI.Allocation

	nomadClient  *nomadAPI.Client
	consulClient *consulAPI.Client

	waitGroup *sync.WaitGroup
}

func NewAllocationWorker(nomadClient *nomadAPI.Client, consulClient *consulAPI.Client, alloc nomadAPI.Allocation, stop chan struct{}) *AllocationWorker {
	return &AllocationWorker{
		Stop:       stop,
		allocation: alloc,
		nomadClient: nomadClient,
		consulClient: consulClient,
		waitGroup: &sync.WaitGroup{},
	}
}

func (w *AllocationWorker) Start(receiver chan<- *message.SourceMessage) {
	go w.watchAllocationEvents(receiver)
}

func (w *AllocationWorker) Wait() {
	w.waitGroup.Wait()
}

func (w *AllocationWorker) watchAllocationEvents(receiver chan<- *message.SourceMessage) {
	w.waitGroup.Add(1)
	defer w.waitGroup.Done()

	ticker := time.NewTicker(10 * time.Second)

	var syncedMessages []string

	for {
		select {
		case <-ticker.C:
			alloc, _, err := w.nomadClient.Allocations().Info(w.allocation.ID, nil)

			if err != nil {
				panic(err)
			}

			for task, state := range alloc.TaskStates {
				for _, event := range state.Events {
					messageKey := fmt.Sprintf("%d %s %s %s", event.Time, event.Type, event.DriverMessage, event.DriverError)

					if stringInSlice(messageKey, syncedMessages) {
						continue
					}

					log := &message.SourceMessage{
						Id: *alloc.Job.ID,
						Message: []byte(fmt.Sprintf("[%s] %s %s", event.Type, event.DriverMessage, event.DriverError)),
						Meta: map[string]string{
							"task": task,
							"alloc": alloc.ID,
							"job": *alloc.Job.ID,
						},
					}

					receiver <- log

					syncedMessages = append(syncedMessages, messageKey)
				}

			}
		case <-w.Stop:
			return
		}
	}
}

func stringInSlice(str string, slice []string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}

	return false
}
