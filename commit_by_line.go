package gittheseus

type CommitByLine struct {
	CommitID string   `json:"commit_id"`
	LineNums []uint64 `json:"line_nums"`
	FilePath string   `json:"file_path"`
}

type CommitByLines []CommitByLine

func (cs CommitByLines) extractCommitsByID(commitID string) CommitByLines {
	var extracted CommitByLines

	for _, commit := range cs {
		if commit.CommitID != commitID {
			continue
		}
		extracted = append(extracted, commit)
	}

	return extracted
}

type aggregatedCommits map[filePath]lineNums

func (cs CommitByLines) aggregateByFilePath() aggregatedCommits {
	agg := aggregatedCommits{}

	for _, c := range cs {
		if _, ok := agg[filePath(c.FilePath)]; !ok {
			agg[filePath(c.FilePath)] = make(lineNums, 0)
		}
		agg[filePath(c.FilePath)] = append(agg[filePath(c.FilePath)], c.LineNums...)
	}

	return agg
}
