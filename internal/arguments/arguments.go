package arguments

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

type Arguments struct {
	nodeID          int
	nodeHosts       map[int]string
	placementGroups map[int][]int
}

func Parse(args []string) (Arguments, error) {
	if len(args) != 3 {
		err := errors.New("this program must be called with exactly three arguments. 1) the current node id " +
			"(positive integer) 2) network addresses for all nodes (json string) 3) mapping of placement groups to nodes " +
			"(json string)")
		return Arguments{}, err
	}

	rawNodeID := args[0]
	rawNodes := args[1]
	rawPlacementGroups := args[2]

	nodes, err := parseNodes(rawNodes)
	if err != nil {
		err = fmt.Errorf("parse network adresses of nodes (argument 2): %w", err)
		return Arguments{}, err
	}

	nodeID, err := parseNodeID(err, rawNodeID, nodes)
	if err != nil {
		err = fmt.Errorf("parse nodeID (argument 1): %w", err)
		return Arguments{}, err
	}

	placementGroups, err := parsePlacementGroups(rawPlacementGroups, nodes)
	if err != nil {
		err = fmt.Errorf("parse placement groups (argument 3): %w", err)
		return Arguments{}, err
	}

	a := Arguments{
		nodeID:          nodeID,
		nodeHosts:       nodes,
		placementGroups: placementGroups,
	}
	return a, nil
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
		return 0, fmt.Errorf("nodeID %v must be >= 0", nodeID)
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
