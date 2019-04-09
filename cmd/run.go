package cmd

import (
	"fmt"
	"github.com/pm-connect/log-shipper/monitoring"
	"github.com/pm-connect/log-shipper/source/dummy"
	"github.com/pm-connect/log-shipper/target/blackhole"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/pm-connect/log-shipper/broker"
	"github.com/pm-connect/log-shipper/config"
	"github.com/pm-connect/log-shipper/connection"
	"github.com/pm-connect/log-shipper/limiter"
	log "github.com/sirupsen/logrus"
)

// RunCommand contains the config and methods for the Run command.
type RunCommand struct {
	Config  string `help:"Specify the path to the config file."`
	Workers int    `help:"Specify the number of works to run."`
	Ui bool `help:"Start with the ui and api enabled."`
}

// NewRunCommand created a new instance of the RunCommand ready to use.
func NewRunCommand() *RunCommand {
	return &RunCommand{
		Config:  "./config.yaml",
		Workers: 1,
		Ui: false,
	}
}

// Run starts the command.
func (c *RunCommand) Run() error {
	conf, err := c.loadConfig()
	if err != nil {
		log.Fatalf("error reading config: %s", err)
	}

	monitor := monitoring.NewMonitor(log.StandardLogger())

	sourceManager := connection.NewManager()
	targetManager := connection.NewManager()

	logBroker := broker.NewBroker(c.Workers, monitor)

	if c.Ui {
		err := c.startUi(monitor)

		if err != nil {
			log.Fatalf("error starting ui: %s", err)
		}
	}

	for name, source := range conf.Sources {
		switch source.Provider {
		case "dummy":
			sourceManager.AddConnection(name, &dummy.Source{})
		}
	}

	for name, target := range conf.Targets {
		switch target.Provider {
		case "blackhole":
			targetManager.AddConnection(name, &blackhole.Target{})
		}
	}

	err = c.startProcesses(conf, sourceManager, targetManager, logBroker)
	if err != nil {
		log.Fatalf("error starting processes: %s", err)
	}

	return nil
}

func (c *RunCommand) startUi(monitor *monitoring.Monitor) error {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		for _, c := range monitor.ConnectionStore.Connections {
			fmt.Fprintf(writer, "%s (%s)\n", c.Name, c.Type)
			fmt.Fprintf(writer, "State: %s\n", c.State)
			fmt.Fprintf(writer, "Inbound Messages: %d\n", c.Stats.GetMessagesInbound())
			fmt.Fprintf(writer, "Outbound Messages: %d\n", c.Stats.GetMessagesOutbound())
			fmt.Fprintf(writer, "Bytes Processed: %d\n", c.Stats.GetBytesProcessed())
			fmt.Fprintf(writer, "Inflight Messages: %d\n", c.Stats.GetInFlightMessages())

			if c.LastLog.Log != nil {
				c.LastLog.Lock()
				fmt.Fprintf(writer, "Last Log: %s %s\n", c.LastLog.Log.Level, c.LastLog.Log.Message)
				c.LastLog.Unlock()
			}

			fmt.Fprintf(writer, "\n")
		}

		fmt.Fprintf(writer, "\n")

		for _, p := range monitor.ProcessStore.Processes {
			fmt.Fprintf(writer, "%s (%s)\n", p.Name, p.Type)
			fmt.Fprintf(writer, "State: %s\n", p.State)
			fmt.Fprintf(writer, "Inbound Messages: %d\n", p.Stats.GetMessagesInbound())
			fmt.Fprintf(writer, "Outbound Messages: %d\n", p.Stats.GetMessagesOutbound())
			fmt.Fprintf(writer, "Bytes Processed: %d\n", p.Stats.GetBytesProcessed())
			fmt.Fprintf(writer, "Inflight Messages: %d\n", p.Stats.GetInFlightMessages())

			if p.LastLog.Log != nil {
				p.LastLog.Lock()
				fmt.Fprintf(writer, "Last Log: %s %s\n", p.LastLog.Log.Level, p.LastLog.Log.Message)
				p.LastLog.Unlock()
			}

			fmt.Fprintf(writer, "\n")
		}
	})

	go http.ListenAndServe(":8888", nil)

	return nil
}

// loadConfig loads and returns the config from the configured file.
func (c *RunCommand) loadConfig() (*config.Config, error) {
	conf := config.NewConfig()

	data, err := c.readConfigFile()
	if err != nil {
		return nil, err
	}

	err = conf.LoadYAML(data)
	if err != nil {
		return nil, err
	}

	return conf, err
}

// readConfigFile returns the string data from the given file.
func (c *RunCommand) readConfigFile() (string, error) {
	if _, err := os.Stat(c.Config); os.IsNotExist(err) {
		return "", fmt.Errorf("config file not found, looking in: %s", c.Config)
	}

	data, err := ioutil.ReadFile(c.Config)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (c *RunCommand) startProcesses(conf *config.Config, sourceManager *connection.Manager, targetManager *connection.Manager, logBroker *broker.Broker) error {
	sourceConnections, err := sourceManager.Start()
	if err != nil {
		return fmt.Errorf("error starting sources: %s", err)
	}

	targetConnections, err := targetManager.Start()
	if err != nil {
		return fmt.Errorf("error starting targets: %s", err)
	}

	brokerSources, err := configureSources(conf.Sources, sourceConnections)
	if err != nil {
		return fmt.Errorf("error configuring sources: %s", err)
	}

	brokerTargets, err := configureTargets(conf.Targets, targetConnections)
	if err != nil {
		return fmt.Errorf("error configuring targets: %s", err)
	}

	for name, source := range brokerSources {
		logBroker.AddSource(name, source)
	}

	for name, target := range brokerTargets {
		logBroker.AddTarget(name, target)
	}

	err = logBroker.Start()
	if err != nil {
		return fmt.Errorf("error starting broker: %s", err)
	}

	return nil
}

func configureSources(sources map[string]config.Source, connectionDetails map[string]*connection.Details) (map[string]*broker.Source, error) {
	brokerSources := map[string]*broker.Source{}

	for name, c := range sources {
		if details, ok := connectionDetails[name]; ok {
			brokerSources[name] = &broker.Source{
				ConnectionDetails: details,
				Targets:           c.Targets,
			}
		} else {
			return nil, fmt.Errorf("unable to find connection for source '%s', check config", name)
		}
	}

	return brokerSources, nil
}

func configureTargets(targets map[string]config.Target, connectionDetails map[string]*connection.Details) (map[string]*broker.Target, error) {
	brokerTargets := map[string]*broker.Target{}

	for name, c := range targets {
		if details, ok := connectionDetails[name]; ok {
			rateLimitRules := createRateLimitRules(name, c.RateLimit)

			brokerTargets[name] = &broker.Target{
				ConnectionDetails: details,
				Config:            c,
				RateLimitRules:    rateLimitRules,
			}
		} else {
			return nil, fmt.Errorf("unable to find connection for source '%s', check config", name)
		}
	}

	return brokerTargets, nil
}

func createRateLimitRules(name string, rateLimits []config.RateLimit) []broker.RateLimitRule {
	var rateLimitRules []broker.RateLimitRule

	for _, rateLimit := range rateLimits {
		bytes, _ := bytefmt.ToBytes(rateLimit.Throughput)
		interval, _ := time.ParseDuration(rateLimit.Mode.Period)
		rateLimiter := limiter.New(name, uint64(bytes), interval, interval, rateLimit.Mode.Duration)

		rateLimitRules = append(rateLimitRules, broker.RateLimitRule{
			RateLimiter:     rateLimiter,
			BreachBehaviour: rateLimit.BreachBehaviour,
		})
	}

	return rateLimitRules
}
