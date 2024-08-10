package goblaze

import "github.com/GoBlaze/goblaze/constants"

var goblazeW = `
  
                                                
   ▄▄ •       ▄▄▄▄· ▄▄▌   ▄▄▄· ·▄▄▄▄•▄▄▄ .
▐█ ▀ ▪▪     ▐█ ▀█▪██•  ▐█ ▀█ ▪▀·.█▌▀▄.▀·
▄█ ▀█▄ ▄█▀▄ ▐█▀▀█▄██▪  ▄█▀▀█ ▄█▀▀▀•▐▀▀▪▄
▐█▄▪▐█▐█▌.▐▌██▄▪▐█▐█▌▐▌▐█ ▪▐▌█▌▪▄█▀▐█▄▄▌
·▀▀▀▀  ▀█▄▀▪·▀▀▀▀ .▀▀▀  ▀  ▀ ·▀▀▀ • ▀▀▀                                            
         `

const (
	static nodeType = iota
	root
	param
	catchAll
)

const (
	DefaultBodyLimit       = 4 * 1024 * 1024
	DefaultConcurrency     = 256 * 1024
	DefaultReadBufferSize  = 4096
	DefaultWriteBufferSize = 4096
)

var DefaultColors = Colors{
	Black:   "\u001b[90m",
	Red:     "\u001b[91m",
	Green:   "\u001b[92m",
	Yellow:  "\u001b[93m",
	Blue:    "\u001b[94m",
	Magenta: "\u001b[95m",
	Cyan:    "\u001b[96m",
	White:   "\u001b[97m",
	Reset:   "\u001b[0m",
}

const cacheLinePadSize = constants.CacheLinePadSize

type cacheLinePadding struct{ _ [cacheLinePadSize]byte }
