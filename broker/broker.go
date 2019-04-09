package broker

import (
	"github.com/pm-connect/log-shipper/config"
	"github.com/pm-connect/log-shipper/connection"
	"github.com/pm-connect/log-shipper/limiter"
	"github.com/pm-connect/log-shipper/monitoring"
)

type Broker struct {
	Sources    map[string]*Source
	Targets    map[string]*Target
	NumWorkers int

	WorkerManager    *WorkerManager
	ConnectorManager *ConnectorManager

	Monitor *monitoring.Monitor
}

type Source struct {
	ConnectionDetails *connection.Details
	Targets           []string
}

type Target struct {
	ConnectionDetails *connection.Details
	Config            config.Target
	RateLimitRules    []RateLimitRule
}

type RateLimitRule struct {
	RateLimiter     *limiter.RateLimiter
	BreachBehaviour config.BreachBehaviour
}

type TargetChannels map[string]chan []byte

func NewBroker(workers int, monitor *monitoring.Monitor) *Broker {
	broker := Broker{}

	broker.Monitor = monitor

	broker.Sources = map[string]*Source{}
	broker.Targets = map[string]*Target{}

	broker.WorkerManager = NewWorkerManager(monitor)
	broker.ConnectorManager = NewConnectorManager(monitor)

	broker.NumWorkers = workers

	return &broker
}

func (b *Broker) AddSource(name string, source *Source) {
	b.Sources[name] = source
}

func (b *Broker) AddTarget(name string, target *Target) {
	b.Targets[name] = target
}

func (b *Broker) Start() error {
	receiver := b.ConnectorManager.Start(b.Sources, b.Targets)

	b.WorkerManager.Start(b.NumWorkers, receiver, b.ConnectorManager)
	b.ConnectorManager.WaitForSources()
	b.WorkerManager.Stop()
	b.ConnectorManager.StopTargets()
	b.ConnectorManager.WaitForTargets()

	return nil
}

func (b *Broker) Stop() {
	b.ConnectorManager.StopSources()
}
