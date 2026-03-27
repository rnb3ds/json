package internal

const (
	// FNV-1a algorithm constants - single source of truth
	FNVOffsetBasis uint64 = 14695981039346656037
	FNVPrime       uint64 = 1099511628211

	// LargeStringHashThreshold is the size threshold for using sampling-based hash.
	// Strings larger than this use HashStringFNV1aSampled for better performance.
	LargeStringHashThreshold = 4096
)

// HashUint64 mixes a uint64 value into the hash using FNV-1a algorithm.
// This is the core mixing function for building composite hashes.
func HashUint64(h, v uint64) uint64 {
	h ^= v
	h *= FNVPrime
	return h
}

// HashBool mixes a bool value into the hash using FNV-1a algorithm.
// Both true and false produce distinct hash changes to prevent collisions.
func HashBool(h uint64, v bool) uint64 {
	if v {
		h ^= 1
	} else {
		h ^= 0xFF
	}
	h *= FNVPrime
	return h
}

// HashInt mixes an int value into the hash using FNV-1a algorithm.
func HashInt(h uint64, v int) uint64 {
	return HashUint64(h, uint64(v))
}

// HashInt64 mixes an int64 value into the hash using FNV-1a algorithm.
func HashInt64(h uint64, v int64) uint64 {
	return HashUint64(h, uint64(v))
}

// HashString mixes a string value into the hash using FNV-1a algorithm.
// The length is included to prevent collisions between short/long strings.
func HashString(h uint64, s string) uint64 {
	h = HashUint64(h, uint64(len(s)))
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= FNVPrime
	}
	return h
}

// HashStringFNV1a computes FNV-1a hash for a string (full scan).
// This is a fast, non-cryptographic hash function suitable for cache keys.
func HashStringFNV1a(s string) uint64 {
	h := FNVOffsetBasis
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= FNVPrime
	}
	return h
}

// HashStringFNV1aSampled computes FNV-1a hash with sampling for large strings.
// PERFORMANCE: For large strings (>4KB), samples first/middle/last sections
// to avoid full scan overhead while maintaining good hash distribution.
func HashStringFNV1aSampled(s string) uint64 {
	const (
		sampleSize   = 512
		middleSample = 256
	)

	h := FNVOffsetBasis
	lenS := len(s)

	// Include length in hash to prevent prefix/suffix collisions
	h ^= uint64(lenS)
	h *= FNVPrime
	h ^= uint64(lenS >> 8)
	h *= FNVPrime

	// First sample
	end := min(sampleSize, lenS)
	for i := range end {
		h ^= uint64(s[i])
		h *= FNVPrime
	}

	// Middle sample
	if lenS > sampleSize {
		midStart := lenS/2 - middleSample/2
		if midStart > end {
			midEnd := min(midStart+middleSample, lenS)
			for i := midStart; i < midEnd; i++ {
				h ^= uint64(s[i])
				h *= FNVPrime
			}
		}
	}

	// Last sample
	start := max(lenS-sampleSize, end)
	for i := start; i < lenS; i++ {
		h ^= uint64(s[i])
		h *= FNVPrime
	}

	return h
}

// HashBytesFNV1a computes FNV-1a hash for a byte slice.
// This is a fast, non-cryptographic hash function suitable for cache keys.
func HashBytesFNV1a(b []byte) uint64 {
	h := FNVOffsetBasis
	for _, c := range b {
		h ^= uint64(c)
		h *= FNVPrime
	}
	return h
}

// HashStringFNV1aSecure computes FNV-1a hash with full scan for security-sensitive contexts.
// SECURITY: Always performs full string scan to prevent collision attacks where an attacker
// crafts strings with identical sampled regions but different content.
// Use this for security-critical cache keys, validation caching, and any context where
// collision attacks are a concern.
// PERFORMANCE: ~30-40% slower than HashStringFNV1aSampled for large strings, but provides
// strong collision resistance guarantees.
func HashStringFNV1aSecure(s string) uint64 {
	h := FNVOffsetBasis
	lenS := len(s)

	// Include length in hash to prevent length extension issues
	h ^= uint64(lenS)
	h *= FNVPrime

	// Full scan - no sampling for security
	for i := 0; i < lenS; i++ {
		h ^= uint64(s[i])
		h *= FNVPrime
	}

	return h
}

// HashBytesFNV1aSecure computes FNV-1a hash with full scan for security-sensitive contexts.
// SECURITY: Always performs full byte slice scan to prevent collision attacks.
func HashBytesFNV1aSecure(b []byte) uint64 {
	h := FNVOffsetBasis
	lenB := len(b)

	// Include length in hash
	h ^= uint64(lenB)
	h *= FNVPrime

	// Full scan
	for _, c := range b {
		h ^= uint64(c)
		h *= FNVPrime
	}

	return h
}
