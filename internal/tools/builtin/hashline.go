package builtin

import (
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"strconv"
)

func computeLineHash(line string) string {
	crc := crc32.ChecksumIEEE([]byte(line))
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, crc)
	encoded := base64.RawURLEncoding.EncodeToString(buf)
	if len(encoded) >= 3 {
		return encoded[:3]
	}
	return encoded
}

func detectCollisions(lines []string) map[string][]int {
	hashToLines := make(map[string][]int)

	for i, line := range lines {
		hash := computeLineHash(line)
		hashToLines[hash] = append(hashToLines[hash], i)
	}

	collisions := make(map[string][]int)
	for hash, lineIndices := range hashToLines {
		if len(lineIndices) > 1 {
			collisions[hash] = lineIndices
		}
	}

	return collisions
}

func disambiguateHash(hash string, occurrence int) string {
	if occurrence == 0 {
		return hash
	}
	return hash + "." + strconv.Itoa(occurrence)
}

func generateHashMap(lines []string) (map[int]string, string) {
	collisions := detectCollisions(lines)
	hashMap := make(map[int]string)

	occurrenceCount := make(map[string]int)

	for i, line := range lines {
		lineNum := i + 1
		baseHash := computeLineHash(line)

		if _, hasCollision := collisions[baseHash]; hasCollision {
			occurrence := occurrenceCount[baseHash]
			hashMap[lineNum] = disambiguateHash(baseHash, occurrence)
			occurrenceCount[baseHash]++
		} else {
			hashMap[lineNum] = baseHash
		}
	}

	var warning string
	if len(collisions) > 0 {
		warning = "Warning: Hash collisions detected and disambiguated with suffixes"
	}

	return hashMap, warning
}
