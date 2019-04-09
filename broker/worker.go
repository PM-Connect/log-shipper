package broker

import (
	"fmt"
	"github.com/pm-connect/log-shipper/message"
	"github.com/pm-connect/log-shipper/monitoring"
	log "github.com/sirupsen/logrus"
	"sync"
)

type WorkerManager struct {
	WaitGroup *sync.WaitGroup
	StopChan  chan interface{}
	Monitor   *monitoring.Monitor
}

type Worker struct {
	Manager          *WorkerManager
	ConnectorManager *ConnectorManager
	ProcessMonitor   *monitoring.Process
}

func NewWorkerManager(monitor *monitoring.Monitor) *WorkerManager {
	return &WorkerManager{
		WaitGroup: &sync.WaitGroup{},
		StopChan:  make(chan interface{}),
		Monitor:   monitor,
	}
}

func (m *WorkerManager) Start(workers int, receiver chan []byte, connectorManager *ConnectorManager) {
	for i := 0; i < workers; i++ {
		m.WaitGroup.Add(1)

		processMonitor := monitoring.NewProcess(fmt.Sprintf("worker %d", i), "worker")

		processMonitor.SetState("starting")

		if m.Monitor != nil {
			m.Monitor.AddProcess(processMonitor)
		}

		worker := &Worker{Manager: m, ConnectorManager: connectorManager, ProcessMonitor: processMonitor}
		go worker.Start(receiver)
	}
}

func (m *WorkerManager) Wait() {
	m.WaitGroup.Wait()
}

// Stop sends a cancel signal and blocks until the workers have finished.
func (m *WorkerManager) Stop() {
	close(m.StopChan)
	m.Wait()
}

func (w *Worker) Start(receiver chan []byte) {
	defer w.Manager.WaitGroup.Done()

	for {
		w.ProcessMonitor.SetState("waiting")
		select {
		case data := <-receiver:
			w.ProcessMonitor.SetState("processing")
			w.ProcessMonitor.Stats.IncrementBytes(uint64(len(data)))
			w.ProcessMonitor.Stats.IncrementMessagesInbound(uint64(1))
			w.ProcessMonitor.Stats.AddInFlightMessage()

			l, err := message.ProtobufToBroker(data)
			if err != nil {
				w.Manager.Monitor.LogForProcess(w.ProcessMonitor, log.ErrorLevel, err.Error())
				continue
			}

			for _, target := range l.Targets {
				err := w.ConnectorManager.SendToTarget(target, data)

				if err == nil {
					w.ProcessMonitor.Stats.IncrementMessagesOutbound(uint64(1))
				}
			}

			w.ProcessMonitor.Stats.RemoveInFlightMessage()
		case <-w.Manager.StopChan:
			w.ProcessMonitor.SetState("finished")
			return
		}
	}
}
