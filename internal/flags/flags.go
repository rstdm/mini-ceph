package flags

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"strconv"
)

type FlagValues struct {
	UseProductionLogger bool
	Port                int
	BearerToken         string
	ObjectFolder        string
	MaxObjectSizeBytes  int64

	NodeID          *int
	NodeHosts       *map[int]string
	PlacementGroups *map[int][]int
}

func Parse() (FlagValues, error) {
	values := FlagValues{}

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
	flag.Parse()

	if len(flag.Args()) != 3 {
		err := errors.New("this program must be called with exactly three flags. 1) the current node id " +
			"(positive integer) 2) network addresses for all nodes (json string) 3) mapping of placement groups to nodes " +
			"(json string)")
		return FlagValues{}, err
	}

	rawNodeID := flag.Arg(0)
	rawNodes := flag.Arg(1)
	rawPlacementGroups := flag.Arg(2)

	nodes, err := parseNodes(rawNodes)
	if err != nil {
		err = fmt.Errorf("parse network adresses of nodes (argument 2): %w", err)
		return FlagValues{}, err
	}

	nodeID, err := parseNodeID(err, rawNodeID, nodes)
	if err != nil {
		err = fmt.Errorf("parse NodeID (argument 1): %w", err)
		return FlagValues{}, err
	}

	placementGroups, err := parsePlacementGroups(rawPlacementGroups, nodes)
	if err != nil {
		err = fmt.Errorf("parse placement groups (argument 3): %w", err)
		return FlagValues{}, err
	}

	/*a := FlagValues{ // TODO
		NodeID:          NodeID,
		NodeHosts:       nodes,
		PlacementGroups: PlacementGroups,
	}*/
	_ = nodes
	_ = nodeID
	_ = placementGroups

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
