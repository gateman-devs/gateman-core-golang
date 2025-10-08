package ipresolver

import (
	"gateman.io/infrastructure/ipresolver/maxmind"
	"gateman.io/infrastructure/ipresolver/types"
)

var IPResolverInstance types.IPResolver = &maxmind.MaxMindIPResolver{}
