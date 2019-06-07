package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/pm-connect/log-shipper/broker"
	"github.com/pm-connect/log-shipper/config"
	"github.com/pm-connect/log-shipper/connection"
	"github.com/pm-connect/log-shipper/limiter"
	"github.com/pm-connect/log-shipper/monitoring"
	"github.com/pm-connect/log-shipper/source/dummy"
	"github.com/pm-connect/log-shipper/source/nomad"
	"github.com/pm-connect/log-shipper/target/blackhole"
	"github.com/pm-connect/log-shipper/target/logzio"
	"github.com/pm-connect/log-shipper/target/stdout"
	"github.com/pm-connect/log-shipper/web"
	log "github.com/sirupsen/logrus"
)

// RunCommand contains the config and methods for the Run command.
type RunCommand struct {
	Config            string `help:"Specify the path to the config file."`
	Workers           int    `help:"Specify the number of works to run."`
	Web               bool   `help:"Start the web server. If UI is on, this is forced to true."`
	Ui                bool   `help:"Start with the ui and api enabled."`
	Port              int    `help:"The port to run the ui and api on."`
	Consul            string `help:"The address to consul for clustering."`
	ConsulServiceName string `help:"The name of the log-shipper service in consul."`
}

// NewRunCommand created a new instance of the RunCommand ready to use.
func NewRunCommand() *RunCommand {
	return &RunCommand{
		Config:            "./config.yaml",
		Workers:           1,
		Web:               true,
		Ui:                false,
		Port:              80,
		Consul:            "127.0.0.1:8500",
		ConsulServiceName: "log-shipper",
	}
}

// Run starts the command.
func (c *RunCommand) Run() error {
	conf, err := c.loadConfig()
	if err != nil {
		log.Fatalf("error reading config: %s", err)
	}

	log.SetLevel(log.WarnLevel)

	monitor := monitoring.NewMonitor(log.StandardLogger())

	sourceManager := connection.NewManager()
	targetManager := connection.NewManager()

	logBroker := broker.NewBroker(c.Workers, monitor)

	if c.Web || c.Ui {
		c.startHttp(monitor)
	}

	for name, source := range conf.Sources {
		switch source.Provider {
		case "dummy":
			sourceManager.AddConnection(name, &dummy.Source{})
		case "nomad":
			sourceManager.AddConnection(name, &nomad.Source{
				Config:     source.Config,
				ConsulAddr: c.Consul,
			})
		}
	}

	for name, target := range conf.Targets {
		switch target.Provider {
		case "blackhole":
			targetManager.AddConnection(name, &blackhole.Target{})
		case "stdout":
			targetManager.AddConnection(name, &stdout.Target{})
		case "logzio":
			targetManager.AddConnection(name, &logzio.Target{
				Config: target.Config,
			})
		}
	}

	err = c.startProcesses(conf, sourceManager, targetManager, logBroker)
	if err != nil {
		log.Fatalf("error starting processes: %s", err)
	}

	return nil
}

func (c *RunCommand) startHttp(monitor *monitoring.Monitor) {
	go web.StartServer(c.Port, monitor, c.Ui, c.Consul, c.ConsulServiceName)
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
				Config:            c,
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

		_ = rateLimiter.Init()

		rateLimitRules = append(rateLimitRules, broker.RateLimitRule{
			RateLimiter:     rateLimiter,
			BreachBehaviour: rateLimit.BreachBehaviour,
		})
	}

	return rateLimitRules
}
