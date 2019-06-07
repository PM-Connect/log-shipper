package nomad

import (
	"fmt"
	consulAPI "github.com/hashicorp/consul/api"
	nomadAPI "github.com/hashicorp/nomad/api"
	"github.com/pm-connect/log-shipper/message"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"time"
)

type Client struct {
	Config map[string]string

	NomadClient  *nomadAPI.Client
	ConsulClient *consulAPI.Client

	id string

	sessionID  string
	sessionTTL string

	allocationWorkers *AllocationWorkers
}

type AllocationWorkers struct {
	sync.Mutex
	workers map[string]*AllocationWorker
}

var (
	httpClient = &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout: 1 * time.Second,
		},
	}
)

func NewAllocationWorkers() *AllocationWorkers {
	return &AllocationWorkers{
		workers: map[string]*AllocationWorker{},
	}
}

func (a *AllocationWorkers) Add(alloc nomadAPI.Allocation, worker *AllocationWorker) {
	a.Lock()
	a.workers[alloc.ID] = worker
	a.Unlock()
}

func (a *AllocationWorkers) Remove(alloc nomadAPI.Allocation) {
	a.Lock()
	if _, ok := a.workers[alloc.ID]; ok {
		delete(a.workers, alloc.ID)
	}
	a.Unlock()
}

func (a *AllocationWorkers) Close(alloc nomadAPI.Allocation) {
	a.Lock()
	if worker, ok := a.workers[alloc.ID]; ok {
		close(worker.Stop)
	}
	a.Unlock()
}

func (a *AllocationWorkers) Wait(alloc nomadAPI.Allocation) {
	a.Lock()
	if worker, ok := a.workers[alloc.ID]; ok {
		worker.Wait()
	}
	a.Unlock()
}

func NewClient(config map[string]string, consulAddr string) (*Client, error) {
	consulAddress := consulAddr

	if configConsul, ok := config["consulAddr"]; ok && len(configConsul) > 0 {
		consulAddress = configConsul
	}

	consulClient, err := NewConsulClient(consulAddress)
	if err != nil {
		return nil, err
	}

	nomadClient, err := NewNomadClient(config["nomadAddr"])
	if err != nil {
		return nil, err
	}

	ttl := "10s"
	if x, ok := config["ttl"]; ok {
		ttl = x
	}

	id := "unknown"
	if x, ok := config["id"]; ok {
		id = x
	}

	return &Client{
		id: id,

		Config: config,

		NomadClient:  nomadClient,
		ConsulClient: consulClient,

		sessionTTL: ttl,

		allocationWorkers: NewAllocationWorkers(),
	}, nil
}

func NewConsulClient(consulAddr string) (*consulAPI.Client, error) {
	config := consulAPI.DefaultConfig()

	config.HttpClient = httpClient

	if len(consulAddr) != 0 {
		config.Address = consulAddr
	}

	return consulAPI.NewClient(config)
}

func NewNomadClient(nomadAddr string) (*nomadAPI.Client, error) {
	config := nomadAPI.DefaultConfig()
	if len(nomadAddr) != 0 {
		config.Address = nomadAddr
	}

	return nomadAPI.NewClient(config)
}

func (c *Client) ReceiveLogs(receiver chan<- *message.SourceMessage) {
	err := c.createSession()
	if err != nil {
		panic(err)
	}

	stop := make(chan struct{})
	defer close(stop)

	go c.ConsulClient.Session().RenewPeriodic(c.sessionTTL, c.sessionID, nil, stop)

	allocationPool := NewAllocationPool()

	go c.syncAllocations(allocationPool)

	for {
		select {
		case alloc := <-allocationPool.AllocationAdded:
			stop := make(chan struct{})
			worker := NewAllocationWorker(c.NomadClient, c.ConsulClient, *alloc, c.id, stop)

			c.allocationWorkers.Add(*alloc, worker)

			go worker.Start(receiver)
		case alloc := <-allocationPool.AllocationRemoved:
			c.allocationWorkers.Close(*alloc)
			c.allocationWorkers.Wait(*alloc)
			c.allocationWorkers.Remove(*alloc)
		}
	}
}

func (c *Client) syncAllocations(pool *AllocationPool) {
	duration, err := time.ParseDuration(c.sessionTTL)

	if err != nil {
		panic(err)
	}

	pool.Sync(c.getAllocations())

	ticker := time.NewTicker(duration)
	for range ticker.C {
		pool.Sync(c.getAllocations())
	}
}

func (c *Client) getAllocations() []*nomadAPI.Allocation {
	if len(c.Config["node"]) == 0 {
		var allocations []*nomadAPI.Allocation

		nodes, _, err := c.NomadClient.Nodes().List(nil)
		if err != nil {
			panic(err)
		}

		for _, node := range nodes {
			nodeAllocations, _, err := c.NomadClient.Nodes().Allocations(node.ID, nil)
			if err != nil {
				panic(err)
			}

			allocations = append(allocations, nodeAllocations...)
		}

		return allocations
	}

	nodesToSync := []string{c.Config["node"]}

	failover := false
	if c.Config["failover"] == "yes" {
		failover = true
	}

	currentAllocLockKey := fmt.Sprintf("log-shipper/nomad/locks/%s/nodes/%s/leader", c.id, c.Config["node"])

	currentAllocLock := &consulAPI.KVPair{
		Key:     currentAllocLockKey,
		Value:   []byte(fmt.Sprintf("primary node: %s", c.Config["node"])),
		Session: c.sessionID,
	}

	log.Info(fmt.Sprintf("[NOMAD] %s Acquiring lock: %s", c.id, currentAllocLockKey))

	acquired, _, err := c.ConsulClient.KV().Acquire(currentAllocLock, nil)
	if err != nil {
		panic(err)
	}

	for !acquired {
		acquired, _, err = c.ConsulClient.KV().Acquire(currentAllocLock, nil)
		if err != nil {
			panic(err)
		}

		if !acquired {
			time.Sleep(1 * time.Second)
		}
	}

	log.Info(fmt.Sprintf("[NOMAD] %s Acquired lock: %s", c.id, currentAllocLockKey))

	if failover {
		otherNodes, _, err := c.NomadClient.Nodes().List(nil)
		if err != nil {
			panic(err)
		}

		for _, node := range otherNodes {
			if node.ID == c.Config["node"] {
				continue
			}

			otherNodePrimaryLockKey := fmt.Sprintf("log-shipper/nomad/locks/%s/nodes/%s/leader", c.id, node.ID)

			lock := &consulAPI.KVPair{
				Key:     otherNodePrimaryLockKey,
				Value:   []byte(fmt.Sprintf("primary node: %s", c.Config["node"])),
				Session: c.sessionID,
			}

			log.Info(fmt.Sprintf("[NOMAD] %s FAILOVER Acquiring lock: %s", c.id, otherNodePrimaryLockKey))

			acquired, _, _ := c.ConsulClient.KV().Acquire(lock, nil)

			log.Info(fmt.Sprintf("[NOMAD] %s FAILOVER Acquired lock: %s", c.id, otherNodePrimaryLockKey))

			otherNodeFailoverLockKey := fmt.Sprintf("log-shipper/nomad/locks/%s/nodes/%s/failover", c.id, node.ID)

			failoverLock := &consulAPI.KVPair{
				Key:     otherNodeFailoverLockKey,
				Value:   []byte(fmt.Sprintf("primary node: %s", c.Config["node"])),
				Session: c.sessionID,
			}

			log.Info(fmt.Sprintf("[NOMAD] %s FAILOVER Acquiring failover lock: %s", c.id, otherNodeFailoverLockKey))

			if acquired {
				acquiredFailover, _, _ := c.ConsulClient.KV().Acquire(failoverLock, nil)

				if acquiredFailover {
					nodesToSync = append(nodesToSync, node.ID)
				}

				log.Info(fmt.Sprintf("[NOMAD] %s FAILOVER Acquired failover lock: %s", c.id, otherNodeFailoverLockKey))

				_, _, _ = c.ConsulClient.KV().Release(lock, nil)
			} else {
				_, _, _ = c.ConsulClient.KV().Release(failoverLock, nil)
				log.Info(fmt.Sprintf("[NOMAD] %s FAILOVER Acquire failover lock aborted, recovery detected: %s", c.id, otherNodeFailoverLockKey))
			}
		}
	}

	var allocations []*nomadAPI.Allocation

	for _, nodeID := range nodesToSync {
		nodeAllocations, _, err := c.NomadClient.Nodes().Allocations(nodeID, nil)
		if err != nil {
			panic(err)
		}

		allocations = append(allocations, nodeAllocations...)
	}

	return allocations
}

func (c *Client) createSession() error {
	session := &consulAPI.SessionEntry{
		TTL:      c.sessionTTL,
		Behavior: "delete",
	}

	sessionID, _, err := c.ConsulClient.Session().Create(session, nil)
	if err != nil {
		return err
	}

	c.sessionID = sessionID

	return nil
}
