package api

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/labstack/echo"
	"net/http"
)

type GetNodesResponse struct {
	Nodes []Node `json:"nodes"`
}

type Node struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
}

func GetNodesRoute(consulAddr string, serviceName string) echo.HandlerFunc {
	if len(consulAddr) == 0 {
		return func(context echo.Context) error {
			return context.JSON(http.StatusOK, []string{})
		}
	}

	config := api.DefaultConfig()

	config.Address = consulAddr

	client, err := api.NewClient(config)

	if err != nil {
		panic(err)
	}

	return func(context echo.Context) error {
		services, _, err := client.Catalog().Service(serviceName, "", nil)

		if err != nil {
			return context.JSON(http.StatusInternalServerError, err)
		}

		var nodes []Node

		for _, service := range services {
			node := Node{
				ID:      service.ServiceID,
				Address: fmt.Sprintf("%s:%d", service.ServiceAddress, service.ServicePort),
				Host:    service.ServiceAddress,
				Port:    service.ServicePort,
			}

			nodes = append(nodes, node)
		}

		return context.JSON(http.StatusOK, GetNodesResponse{Nodes: nodes})
	}
}
