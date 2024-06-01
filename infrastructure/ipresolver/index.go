package ipresolver

import (
	"authone.usepolymer.co/infrastructure/ipresolver/maxmind"
	"authone.usepolymer.co/infrastructure/ipresolver/types"
)

var IPResolverInstance types.IPResolver = &maxmind.MaxMindIPResolver{}
