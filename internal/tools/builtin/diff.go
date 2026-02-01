package builtin

import (
	"fmt"
	"strings"
)

const contextLines = 3

func generateUnifiedDiff(path, oldContent, newContent string) string {
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)

	if len(oldLines) == 0 && len(newLines) == 0 {
		return ""
	}

	if len(oldLines) == 0 {
		return generateNewFileDiff(path, newContent, 1000)
	}

	hunks := computeHunks(oldLines, newLines)
	if len(hunks) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("--- %s\n", path))
	builder.WriteString(fmt.Sprintf("+++ %s\n", path))

	for _, hunk := range hunks {
		builder.WriteString(hunk)
	}

	return builder.String()
}

func generateNewFileDiff(path, content string, maxLines int) string {
	lines := splitLines(content)
	totalLines := len(lines)

	if totalLines == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("--- /dev/null\n")
	builder.WriteString(fmt.Sprintf("+++ %s\n", path))

	if totalLines <= maxLines {
		builder.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", totalLines))
		for _, line := range lines {
			builder.WriteString("+")
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	} else {
		firstCount := maxLines * 2 / 3
		lastCount := maxLines - firstCount

		builder.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@ (showing %d of %d lines)\n", firstCount, firstCount, totalLines))
		for i := 0; i < firstCount; i++ {
			builder.WriteString("+")
			builder.WriteString(lines[i])
			builder.WriteString("\n")
		}

		builder.WriteString(fmt.Sprintf("@@ ... %d lines omitted ... @@\n", totalLines-firstCount-lastCount))

		startLast := totalLines - lastCount
		builder.WriteString(fmt.Sprintf("@@ -0,0 +%d,%d @@\n", startLast+1, lastCount))
		for i := startLast; i < totalLines; i++ {
			builder.WriteString("+")
			builder.WriteString(lines[i])
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

func computeHunks(oldLines, newLines []string) []string {
	changes := findChanges(oldLines, newLines)
	if len(changes) == 0 {
		return nil
	}

	var hunks []string
	hunkStart := -1
	hunkEnd := -1

	for _, change := range changes {
		if hunkStart == -1 {
			hunkStart = max(0, change.oldStart-contextLines)
			hunkEnd = min(len(oldLines), change.oldEnd+contextLines)
		} else if change.oldStart-contextLines <= hunkEnd {
			hunkEnd = min(len(oldLines), change.oldEnd+contextLines)
		} else {
			hunks = append(hunks, formatHunk(oldLines, newLines, changes, hunkStart, hunkEnd))
			hunkStart = max(0, change.oldStart-contextLines)
			hunkEnd = min(len(oldLines), change.oldEnd+contextLines)
		}
	}

	if hunkStart != -1 {
		hunks = append(hunks, formatHunk(oldLines, newLines, changes, hunkStart, hunkEnd))
	}

	return hunks
}

type change struct {
	oldStart int
	oldEnd   int
	newStart int
	newEnd   int
}

func findChanges(oldLines, newLines []string) []change {
	lcs := computeLCS(oldLines, newLines)

	var changes []change
	oldIndex, newIndex := 0, 0
	lcsIndex := 0

	for lcsIndex < len(lcs) {
		oldMatch, newMatch := lcs[lcsIndex].oldIndex, lcs[lcsIndex].newIndex

		if oldIndex < oldMatch || newIndex < newMatch {
			changes = append(changes, change{
				oldStart: oldIndex,
				oldEnd:   oldMatch,
				newStart: newIndex,
				newEnd:   newMatch,
			})
		}

		oldIndex = oldMatch + 1
		newIndex = newMatch + 1
		lcsIndex++
	}

	if oldIndex < len(oldLines) || newIndex < len(newLines) {
		changes = append(changes, change{
			oldStart: oldIndex,
			oldEnd:   len(oldLines),
			newStart: newIndex,
			newEnd:   len(newLines),
		})
	}

	return changes
}

type lcsMatch struct {
	oldIndex int
	newIndex int
}

func computeLCS(oldLines, newLines []string) []lcsMatch {
	m, n := len(oldLines), len(newLines)
	if m == 0 || n == 0 {
		return nil
	}

	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if oldLines[i-1] == newLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	var result []lcsMatch
	i, j := m, n
	for i > 0 && j > 0 {
		if oldLines[i-1] == newLines[j-1] {
			result = append(result, lcsMatch{oldIndex: i - 1, newIndex: j - 1})
			i--
			j--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	for left, right := 0, len(result)-1; left < right; left, right = left+1, right-1 {
		result[left], result[right] = result[right], result[left]
	}

	return result
}

func formatHunk(oldLines, newLines []string, changes []change, hunkStart, hunkEnd int) string {
	relevantChanges := filterChangesInRange(changes, hunkStart, hunkEnd)
	if len(relevantChanges) == 0 {
		return ""
	}

	oldCount := hunkEnd - hunkStart
	addedLines, removedLines := 0, 0
	for _, c := range relevantChanges {
		removedLines += c.oldEnd - c.oldStart
		addedLines += c.newEnd - c.newStart
	}
	newCount := oldCount - removedLines + addedLines

	newStart := computeNewStart(oldLines, newLines, relevantChanges, hunkStart)

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", hunkStart+1, oldCount, newStart+1, newCount))

	oldPos := hunkStart
	changeIndex := 0

	for oldPos < hunkEnd || changeIndex < len(relevantChanges) {
		if changeIndex < len(relevantChanges) {
			c := relevantChanges[changeIndex]

			if oldPos == c.oldStart || (c.oldStart == c.oldEnd && oldPos >= c.oldStart && changeIndex < len(relevantChanges)) {
				for i := c.oldStart; i < c.oldEnd; i++ {
					builder.WriteString("-")
					builder.WriteString(oldLines[i])
					builder.WriteString("\n")
					oldPos++
				}

				for i := c.newStart; i < c.newEnd; i++ {
					builder.WriteString("+")
					builder.WriteString(newLines[i])
					builder.WriteString("\n")
				}

				changeIndex++

				if c.oldStart == c.oldEnd {
					continue
				}
			}
		}

		if oldPos < hunkEnd {
			builder.WriteString(" ")
			builder.WriteString(oldLines[oldPos])
			builder.WriteString("\n")
			oldPos++
		} else {
			break
		}
	}

	return builder.String()
}

func computeNewStart(oldLines, newLines []string, changes []change, hunkStart int) int {
	offset := 0
	for _, c := range changes {
		if c.oldStart < hunkStart {
			removed := c.oldEnd - c.oldStart
			added := c.newEnd - c.newStart
			offset += added - removed
		} else {
			break
		}
	}
	return hunkStart + offset
}

func filterChangesInRange(changes []change, start, end int) []change {
	var result []change
	for _, c := range changes {
		if c.oldEnd >= start && c.oldStart <= end {
			result = append(result, c)
		}
	}
	return result
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
