package distribution

type Distribution struct {
	IsPrimary             bool
	IsInPlacementGroup    bool
	CorrectPlacementGroup uint32
	PrimaryHost           string
	SlaveHosts            []string
}
