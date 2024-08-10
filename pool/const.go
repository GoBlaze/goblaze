package pool

import "github.com/GoBlaze/goblaze/constants"

const cacheLinePadSize = constants.CacheLinePadSize

type cacheLinePadding struct {
	_ [cacheLinePadSize]byte
}
