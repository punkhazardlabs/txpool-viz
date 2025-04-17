package utils

import (
	"fmt"
)

const (
	redisStreamPrefix       = "txpool:%s:stream"      // Per-client stream (list of incoming tx hashes)
	redisClientMetaPrefix   = "txpool:%s:meta"        // Per-client high-level tx & metadata records
	redisUniversalSortedSet = "txpool:universal"      // Global ZSET of tx hashes ordered by received time
	redisGasIndexPrefix     = "txpool:%s:index:gas"   // Sorted by gas price
	redisNonceIndexPrefix   = "txpool:%s:index:nonce" // Sorted by nonce
	redisTypeIndexPrefix    = "txpool:%s:index:type"  // Sorted by tx type
)

func RedisStreamKey(client string) string {
	return fmt.Sprintf(redisStreamPrefix, client)
}


func RedisClientMetaKey(client string) string {
	return fmt.Sprintf(redisClientMetaPrefix, client)
}

func RedisUniversalKey() string {
	return redisUniversalSortedSet
}

func RedisGasIndexKey(client string) string {
	return fmt.Sprintf(redisGasIndexPrefix, client)
}

func RedisNonceIndexKey(client string) string {
	return fmt.Sprintf(redisNonceIndexPrefix, client)
}

func RedisTypeIndexKey(client string) string {
	return fmt.Sprintf(redisTypeIndexPrefix, client)
}
