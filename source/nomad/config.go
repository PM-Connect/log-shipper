package nomad

import (
	"fmt"
	nomadAPI "github.com/hashicorp/nomad/api"
	"strings"
)

type AllocationConfig struct {
	JobConfig    *JobConfig
	GroupConfigs map[string]*GroupConfig
	TaskConfigs  map[string]*TaskConfig
	Alloc        *nomadAPI.Allocation
}

type JobConfig struct {
	ID      string
	Enabled bool
}

type GroupConfig struct {
	ID      string
	Enabled bool
	Files   map[string]*SourceConfig
}

type TaskConfig struct {
	ID               string
	Enabled          bool
	Outputs          map[string]*SourceConfig
	WatchNomadEvents bool
}

type SourceConfig struct {
	Type         string
	LogDelimiter string
}

func NewFromAllocation(alloc *nomadAPI.Allocation) *AllocationConfig {
	groupConfigs := map[string]*GroupConfig{}
	taskConfigs := map[string]*TaskConfig{}

	for _, group := range alloc.Job.TaskGroups {
		groupConfigs[fmt.Sprintf("%s/%s", *alloc.Job.Name, *group.Name)] = metaToGroupConfig(group.Meta)

		for _, task := range group.Tasks {
			taskConfigs[fmt.Sprintf("%s/%s/%s", *alloc.Job.Name, *group.Name, task.Name)] = metaToTaskConfig(task.Meta)
		}
	}

	jobConfig := metaToJobConfig(alloc.Job.Meta)

	return &AllocationConfig{
		JobConfig:    jobConfig,
		GroupConfigs: groupConfigs,
		TaskConfigs:  taskConfigs,
		Alloc:        alloc,
	}
}

func metaToGroupConfig(meta map[string]string) *GroupConfig {
	config := &GroupConfig{
		Enabled: true,
	}

	if id, ok := meta["log_shipper_id"]; ok {
		config.ID = id
	}

	if enabled, ok := meta["log_shipper_enabled"]; ok && enabled == "false" {
		config.Enabled = false
	}

	if enabled, ok := meta["log_shipper_enabled"]; ok && enabled == "true" {
		config.Enabled = true
	}

	if files, ok := meta["log_shipper_files"]; ok {
		fileConfigs := map[string]*SourceConfig{}

		for _, s := range strings.Split(files, ",") {
			parts := strings.Split(s, ":")

			if len(parts) < 0 {
				continue
			}

			fileConfig := &SourceConfig{
				LogDelimiter: "\n",
				Type:         "allocation-log-file",
			}

			if len(parts) > 1 {
				fileConfig.Type = parts[1]
			}

			if len(parts) > 2 {
				fileConfig.LogDelimiter = parts[2]
			}

			fileConfigs[parts[0]] = fileConfig
		}

		config.Files = fileConfigs
	}

	return config
}

func metaToTaskConfig(meta map[string]string) *TaskConfig {
	config := &TaskConfig{
		Enabled:          true,
		WatchNomadEvents: true,
	}

	if id, ok := meta["log_shipper_id"]; ok {
		config.ID = id
	}

	if enabled, ok := meta["log_shipper_enabled"]; ok && enabled == "false" {
		config.Enabled = false
	}

	if enabled, ok := meta["log_shipper_enabled"]; ok && enabled == "true" {
		config.Enabled = true
	}

	if watch, ok := meta["log_shipper_watch_events"]; ok && watch == "true" {
		config.WatchNomadEvents = true
	}

	if watch, ok := meta["log_shipper_watch_events"]; ok && watch == "false" {
		config.WatchNomadEvents = false
	}

	if outputs, ok := meta["log_shipper_outputs"]; ok {
		outputConfigs := map[string]*SourceConfig{}

		for _, s := range strings.Split(outputs, ",") {
			parts := strings.Split(s, ":")

			if len(parts) < 0 {
				continue
			}

			outputConfig := &SourceConfig{
				LogDelimiter: "\n",
				Type:         "allocation-log-file",
			}

			if len(parts) > 1 {
				outputConfig.Type = parts[1]
			}

			if len(parts) > 2 {
				outputConfig.LogDelimiter = parts[2]
			}

			outputConfigs[parts[0]] = outputConfig
		}

		config.Outputs = outputConfigs
	}

	return config
}

func metaToJobConfig(meta map[string]string) *JobConfig {
	config := &JobConfig{
		Enabled: true,
	}

	if id, ok := meta["log_shipper_id"]; ok {
		config.ID = id
	}

	if enabled, ok := meta["log_shipper_enabled"]; ok && enabled == "false" {
		config.Enabled = false
	}

	if enabled, ok := meta["log_shipper_enabled"]; ok && enabled == "true" {
		config.Enabled = true
	}

	return config
}
