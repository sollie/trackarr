package cache

import (
	"github.com/sollie/trackarr/config"
)

/* Structs */

type CacheItem struct {
	Name    string
	Data    []byte
	Release *config.ReleaseInfo
}
