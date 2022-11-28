package configuration

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strconv"
)

type Configuration struct {
	UseProductionLogger bool
	Port                int
	BearerToken         string
	ObjectFolder        string
	MaxObjectSizeBytes  int64

	NodeID          int
	NodeHosts       map[int]string
	PlacementGroups map[int][]int
}

func Parse() (Configuration, error) {
	values := Configuration{}

	flag.BoolVar(&values.UseProductionLogger, "useProductionLogger", false, "Determines weather the logger "+
		"should produce json output or human readable output")
	flag.IntVar(&values.Port, "port", 5000, "Port on which to serve http requests.")
	flag.StringVar(&values.BearerToken, "bearerToken", "", "BearerToken that is used to authorize all "+
		"requests. Every request will be accepted without authorization if the token is empty.")
	flag.StringVar(&values.ObjectFolder, "objectFolder", "./data", "Relative path to the folder that is "+
		"used for object storage and retrieval")
	flag.Int64Var(&values.MaxObjectSizeBytes, "maxObjectSizeBytes", 20000000, "Objects that are bigger than "+
		"the specified size can not be persisted. Note that this doesn't influence already created objects which will "+
		"still be available for download.")

	// these values must be parsed / validated manually
	var rawNodeID string // we cannot use an integer, because we have to detect absent values; the flag library doesn't support *int
	var rawNodes string
	var rawPlacementGroups string

	flag.StringVar(&rawNodeID, "nodeID", "", "non-negative integer which specifies the ID of the current node")
	flag.StringVar(&rawNodes, "nodes", "", "json encoded key value map which maps the ID of each node "+
		"to a host. The host must include the schema. Example: {\"0\": \"http://localhost:5000\"}")
	flag.StringVar(&rawPlacementGroups, "placementGroups", "", "json encoded map of placement groups "+
		"(keys) and the node IDs of that particular placement group. Example which maps node 1 and 2 to placement group "+
		"0: {\"0\": [1, 2]}")

	flag.Parse()

	nodeHosts, err := parseNodes(rawNodes)
	if err != nil {
		err = fmt.Errorf("parse network adresses of nodes (argument 2): %w", err)
		return Configuration{}, err
	}
	values.NodeHosts = nodeHosts

	nodeID, err := parseNodeID(err, rawNodeID, values.NodeHosts)
	if err != nil {
		err = fmt.Errorf("parse NodeID (argument 1): %w", err)
		return Configuration{}, err
	}
	values.NodeID = nodeID

	placementGroups, err := parsePlacementGroups(rawPlacementGroups, values.NodeHosts)
	if err != nil {
		err = fmt.Errorf("parse placement groups (argument 3): %w", err)
		return Configuration{}, err
	}
	values.PlacementGroups = placementGroups

	return values, nil
}

func parseNodes(rawNodes string) (map[int]string, error) {
	var parsedNodes map[int]string
	if err := json.Unmarshal([]byte(rawNodes), &parsedNodes); err != nil {
		return nil, fmt.Errorf("parse json '%v': %w", rawNodes, err)
	}

	for currentNodeID, currentNodeHost := range parsedNodes {
		if currentNodeID < 0 {
			return nil, fmt.Errorf("all node ids must be non negative integere values")
		}

		currentURL, err := url.ParseRequestURI(currentNodeHost)
		if err != nil {
			return nil, fmt.Errorf("parse URL %v of node %v: %w", currentNodeHost, currentNodeID, err)
		}

		if currentURL.Host == "" || currentURL.Path != "" {
			return nil, fmt.Errorf("network address ('%v') of node %v must contain a scheme and must not contain a "+
				"path. Example: http://foobar.com:5000", currentNodeHost, currentNodeID)
		}

		parsedNodes[currentNodeID] = currentURL.Host // the result should only contain the host
	}

	return parsedNodes, nil
}

func parseNodeID(err error, rawNodeID string, nodes map[int]string) (int, error) {
	nodeID, err := strconv.Atoi(rawNodeID)
	if err != nil {
		return 0, fmt.Errorf("rawNodeID '%v' is no valid integer: %w", rawNodeID, err)
	}
	if nodeID < 0 {
		return 0, fmt.Errorf("NodeID %v must be >= 0", nodeID)
	}

	if _, ok := nodes[nodeID]; !ok {
		return 0, fmt.Errorf("node %v is does not exist in the list of node network adresses", nodeID)
	}

	return nodeID, nil
}

func parsePlacementGroups(rawPlacementGroups string, nodes map[int]string) (map[int][]int, error) {
	var parsedPlacementGroups map[int][]int
	if err := json.Unmarshal([]byte(rawPlacementGroups), &parsedPlacementGroups); err != nil {
		err = fmt.Errorf("parse json string %v: %w", rawPlacementGroups, err)
		return nil, err
	}

	for placementGroup, nodesOfPG := range parsedPlacementGroups {
		for _, nodeOfPG := range nodesOfPG {
			if _, ok := nodes[nodeOfPG]; !ok {
				err := fmt.Errorf("node %v in placement group %v is not mentioned in the list of node "+
					"network adresses", nodeOfPG, placementGroup)
				return nil, err
			}
		}
	}

	return parsedPlacementGroups, nil
}