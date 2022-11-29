package distribution

import (
	"encoding/binary"
	"fmt"
	"github.com/rstdm/glados/internal/api/object/hash"
)

type Handler struct {
	nodeID          int
	nodeHosts       []string
	placementGroups [][]int
}

func NewHandler(nodeID int, nodeHosts []string, placementGroups [][]int) *Handler {
	handler := &Handler{
		nodeID:          nodeID,
		nodeHosts:       nodeHosts,
		placementGroups: placementGroups,
	}
	return handler
}

func (h *Handler) GetDistribution(objectHash string) (Distribution, error) {
	binaryHash, err := hash.GetBinaryHash(objectHash)
	if err != nil {
		err = fmt.Errorf("get binary hash from object hash: %w", err)
		return Distribution{}, err
	}

	pgIdx := binary.BigEndian.Uint32(binaryHash) % uint32(len(h.placementGroups))
	pg := h.placementGroups[pgIdx]

	isPrimary := pg[0] == h.nodeID
	primaryHost := h.nodeHosts[pg[0]]
	isInPG := isPrimary
	if !isInPG {
		for _, pgNode := range pg {
			if h.nodeID == pgNode {
				isInPG = true
				break
			}
		}
	}

	var slaveHostsIDs []int
	if isPrimary {
		slaveHostsIDs = pg[1:]
	} else {
		slaveHostsIDs = pg
	}

	var slaveHosts []string
	for _, slaveHostID := range slaveHostsIDs {
		slaveHosts = append(slaveHosts, h.nodeHosts[slaveHostID])
	}

	distribution := Distribution{
		IsPrimary:             isPrimary,
		IsInPlacementGroup:    isInPG,
		CorrectPlacementGroup: pgIdx,
		PrimaryHost:           primaryHost,
		SlaveHosts:            slaveHosts,
	}
	return distribution, nil
}
