package hashline

import (
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
)

func ComputeLineHash(line string) string {
	crc := crc32.ChecksumIEEE([]byte(line))
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, crc)
	encoded := base64.RawURLEncoding.EncodeToString(buf)
	return encoded[:3]
}

func GenerateHashMap(lines []string) map[int]string {
	hashMap := make(map[int]string, len(lines))

	for i, line := range lines {
		hashMap[i+1] = ComputeLineHash(line)
	}

	return hashMap
}
