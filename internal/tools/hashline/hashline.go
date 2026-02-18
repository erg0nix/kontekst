package hashline

import (
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
)

// ComputeLineHash returns a short 3-character CRC32-based hash for a single line of text.
func ComputeLineHash(line string) string {
	crc := crc32.ChecksumIEEE([]byte(line))
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, crc)
	encoded := base64.RawURLEncoding.EncodeToString(buf)
	return encoded[:3]
}

// GenerateHashMap returns a map from 1-indexed line numbers to their computed hashes.
func GenerateHashMap(lines []string) map[int]string {
	hashMap := make(map[int]string, len(lines))

	for i, line := range lines {
		hashMap[i+1] = ComputeLineHash(line)
	}

	return hashMap
}
