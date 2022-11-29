package configuration

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
)

const (
	relativePersistedConfigurationPath = "persistedConfiguration.json"
	relativeObjectStoragePath          = "data"
)

type Configuration struct {
	UseProductionLogger bool
	Port                int
	UserBearerToken     string
	ClusterBearerToken  string
	MaxObjectSizeBytes  int64

	ObjectFolder    string
	NodeID          int
	NodeHosts       []string
	PlacementGroups [][]int
}

type persistedConfiguration struct {
	NodeID          int
	NodeHosts       []string
	PlacementGroups [][]int
}

func Parse() (Configuration, error) {
	values := Configuration{}

	flag.BoolVar(&values.UseProductionLogger, "useProductionLogger", false, "Determines weather the logger "+
		"should produce json output or human readable output")
	flag.IntVar(&values.Port, "port", 5000, "Port on which to serve http requests.")
	flag.StringVar(&values.UserBearerToken, "userBearerToken", "", "this token is used to authorize all "+
		" user requests. Every request will be accepted without authorization if the token is empty.")
	flag.StringVar(&values.ClusterBearerToken, "clusterBearerToken", "", "this token is used internally "+
		"by all OSDs to authenticate each other. If no value is specified the userBearerToken will be used.")
	flag.Int64Var(&values.MaxObjectSizeBytes, "maxObjectSizeBytes", 20000000, "Objects that are bigger than "+
		"the specified size can not be persisted. Note that this doesn't influence already created objects which will "+
		"still be available for download.")

	// these values must be parsed / validated manually
	var dataFolder string
	var rawNodeID string // we cannot use an integer, because we have to detect absent values; the flag library doesn't support *int
	var rawNodes string
	var rawPlacementGroups string

	flag.StringVar(&dataFolder, "dataFolder", ".", "Relative path to the folder that is "+
		"used to store information.")
	flag.StringVar(&rawNodeID, "nodeID", "", "non-negative integer which specifies the ID of the current node")
	flag.StringVar(&rawNodes, "nodes", "", "json encoded list of hosts for each node. The position in "+
		"the list is equal to the nodeID of the node. The host must include the schema. "+
		"Example: [\"http://localhost:5000\", \"http://localhost:5001\"] -> The host of node 1 is localhost:5001")
	flag.StringVar(&rawPlacementGroups, "placementGroups", "", "json encoded list of placement groups. "+
		"Each placement group contains the IDs of the nodes which belong to this placement group. "+
		"Example which maps node 1 and 2 to placement group 0 and node 3 and 4 to placement group 1:"+
		"[[1, 2], [3, 4]]")

	flag.Parse()

	if values.ClusterBearerToken == "" {
		values.ClusterBearerToken = values.UserBearerToken
	}

	values.ObjectFolder = filepath.Join(dataFolder, relativeObjectStoragePath)
	persistedConfigurationPath := filepath.Join(dataFolder, relativePersistedConfigurationPath)

	pc, err := loadPersistedConfiguration(persistedConfigurationPath)
	if err != nil {
		err = fmt.Errorf("load persisted configuration: %w", err)
		return Configuration{}, err
	}

	if pc != nil && rawNodeID == "" && rawNodes == "" && rawPlacementGroups == "" {
		// an old instance persisted configuration data and the data hasn't been specified for the current instance
		// -> use the old data
		values.NodeID = pc.NodeID
		values.NodeHosts = pc.NodeHosts
		values.PlacementGroups = pc.PlacementGroups

		return values, nil
	}

	// use the configuration data of this instance

	nodeHosts, err := parseNodes(rawNodes)
	if err != nil {
		err = fmt.Errorf("parse network adresses of nodes: %w", err)
		return Configuration{}, err
	}
	values.NodeHosts = nodeHosts

	nodeID, err := parseNodeID(err, rawNodeID, values.NodeHosts)
	if err != nil {
		err = fmt.Errorf("parse NodeID: %w", err)
		return Configuration{}, err
	}
	values.NodeID = nodeID

	placementGroups, err := parsePlacementGroups(rawPlacementGroups, values.NodeHosts)
	if err != nil {
		err = fmt.Errorf("parse placement groups: %w", err)
		return Configuration{}, err
	}
	values.PlacementGroups = placementGroups

	newPersistedConfiguration := persistedConfiguration{
		NodeID:          values.NodeID,
		NodeHosts:       values.NodeHosts,
		PlacementGroups: values.PlacementGroups,
	}

	// make sure that the configuration for these fields hasn't changed
	if pc != nil && !reflect.DeepEqual(*pc, newPersistedConfiguration) {
		err = fmt.Errorf("immutable configuration values have changed from %#v to %#v", *pc, newPersistedConfiguration)
		return Configuration{}, err
	}

	if pc == nil { // persist the configuration for the next instance
		if err := persistConfiguration(newPersistedConfiguration, persistedConfigurationPath); err != nil {
			err = fmt.Errorf("persist configuration: %w", err)
			return Configuration{}, err
		}
	}

	return values, nil
}

func loadPersistedConfiguration(persistedConfigurationPath string) (*persistedConfiguration, error) {
	binaryContent, err := os.ReadFile(persistedConfigurationPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		err = fmt.Errorf("open %v: %w", persistedConfigurationPath, err)
		return nil, err
	}

	var pc persistedConfiguration
	if err := json.Unmarshal(binaryContent, &pc); err != nil {
		err = fmt.Errorf("parse content of %v: %w", persistedConfigurationPath, err)
		return nil, err
	}

	return &pc, nil
}

func persistConfiguration(pc persistedConfiguration, persistedConfigurationPath string) error {
	encodedData, err := json.MarshalIndent(pc, "", "  ") // use two spaces for indention
	if err != nil {
		return fmt.Errorf("json encode: %w", err)
	}

	if err := os.WriteFile(persistedConfigurationPath, encodedData, 0666); err != nil {
		return fmt.Errorf("write to file %v: %w", persistedConfigurationPath, err)
	}

	return nil
}

func parseNodes(rawNodes string) ([]string, error) {
	if rawNodes == "" {
		err := errors.New("nodes must not be empty")
		return nil, err
	}

	var parsedNodes []string
	if err := json.Unmarshal([]byte(rawNodes), &parsedNodes); err != nil {
		return nil, fmt.Errorf("parse json '%v': %w", rawNodes, err)
	}

	for currentNodeID, currentNodeHost := range parsedNodes {
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

func parseNodeID(err error, rawNodeID string, nodes []string) (int, error) {
	if rawNodeID == "" {
		err = errors.New("nodeID must not be empty")
		return 0, err
	}

	nodeID, err := strconv.Atoi(rawNodeID)
	if err != nil {
		return 0, fmt.Errorf("rawNodeID '%v' is no valid integer: %w", rawNodeID, err)
	}
	if nodeID < 0 {
		return 0, fmt.Errorf("NodeID %v must be >= 0", nodeID)
	}

	if nodeID >= len(nodes) {
		err = fmt.Errorf("node %v has no associated host because the length of the host slice is %v", nodeID, len(nodes))
		return 0, err
	}

	return nodeID, nil
}

func parsePlacementGroups(rawPlacementGroups string, nodes []string) ([][]int, error) {
	if rawPlacementGroups == "" {
		err := errors.New("placementGroups must not be empty")
		return nil, err
	}

	var parsedPlacementGroups [][]int
	if err := json.Unmarshal([]byte(rawPlacementGroups), &parsedPlacementGroups); err != nil {
		err = fmt.Errorf("parse json string %v: %w", rawPlacementGroups, err)
		return nil, err
	}

	for placementGroup, nodesOfPG := range parsedPlacementGroups {
		for _, nodeOfPG := range nodesOfPG {
			if nodeOfPG >= len(nodes) {
				err := fmt.Errorf("node %v in placement group %v is not mentioned in the list of node "+
					"network adresses", nodeOfPG, placementGroup)
				return nil, err
			}
		}
	}

	return parsedPlacementGroups, nil
}
