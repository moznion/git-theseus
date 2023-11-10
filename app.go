package gittheseus

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type filePath = string
type lineNums = []uint64
type commitID = string
type CommitToDiffs = map[commitID]*FilepathToLines
type FilepathToLines = map[filePath]lineNums

func Run(filePath string, dryrun bool) error {
	fileContents, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to load the file '%s': %w", filePath, err)
	}

	var commits CommitToDiffs
	err = json.Unmarshal(fileContents, &commits)
	if err != nil {
		return fmt.Errorf("failed to load the file contents of '%s': %w", filePath, err)
	}

	sortedCommitIDs, err := sortCommitIDsTopologicallyAsAsc(extractCommitIDs(commits))
	if err != nil {
		return fmt.Errorf("failed to sort the commit IDs topologically: %w", err)
	}

	err = doCommit(sortedCommitIDs, commits, dryrun)
	if err != nil {
		return fmt.Errorf("git-commit operation failed: %w", err)
	}

	return nil
}

type lineNumSet map[uint64]struct{}
type executedMemo map[filePath]lineNumSet

func doCommit(sortedCommitIDs []string, commits CommitToDiffs, dryrun bool) error {
	memo := executedMemo{}

	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get the current working path: %w", err)
	}

	repo, err := git.PlainOpen(workingDir)
	if err != nil {
		return fmt.Errorf("failed to open the git repository on '%s': %w", workingDir, err)
	}

	gitWorktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get the git worktree info: %w", err)
	}

	type fileContents struct {
		bytes []byte
		mode  fs.FileMode
	}

	originalFileContentsMap := make(map[string]*fileContents)
	for _, fp2lines := range commits {
		for fp := range *fp2lines {
			err := func() error {
				f, err := os.Open(fp) // to read the stats of a file
				if err != nil {
					return fmt.Errorf("failed to open the file '%s': %w", fp, err)
				}
				defer func() {
					_ = f.Close()
				}()
				fileStat, _ := f.Stat()

				_, ok := originalFileContentsMap[fp]
				if ok {
					return nil
				}

				originalFileContents, err := os.ReadFile(fp)
				if err != nil {
					return fmt.Errorf("failed to open the file '%s': %w", fp, err)
				}

				originalFileContentsMap[fp] = &fileContents{
					bytes: originalFileContents,
					mode:  fileStat.Mode(),
				}

				return nil
			}()
			if err != nil {
				return err
			}
		}
	}

	sizeOfSortedCommitIDs := len(sortedCommitIDs)
	for i := range sortedCommitIDs {
		commitID := sortedCommitIDs[i]
		isLastCommit := sizeOfSortedCommitIDs <= i+1

		filepath2lines, ok := commits[commitID]
		if !ok {
			continue
		}

		for fp, lns := range *filepath2lines {
			err := func() error {
				if _, ok := memo[fp]; !ok {
					memo[fp] = lineNumSet{}
				}

				for _, ln := range lns {
					memo[fp][ln] = struct{}{}
				}

				originalFileContents, ok := originalFileContentsMap[fp]
				if !ok {
					return fmt.Errorf("unexpected error; no original file contents of '%s'", fp)
				}

				scanner := bufio.NewScanner(bytes.NewReader(originalFileContents.bytes))
				scanner.Split(bufio.ScanLines)

				var lines []string
				lineNum := uint64(1)
				for scanner.Scan() {
					func() {
						defer func() {
							lineNum++
						}()
						if _, ok := memo[fp][lineNum]; !ok {
							return
						}
						lines = append(lines, scanner.Text())
					}()
				}

				err = os.WriteFile(fp, []byte(strings.Join(lines, "\n")), originalFileContents.mode)
				if err != nil {
					return fmt.Errorf("failed to write the file contents onto '%s': %w", fp, err)
				}
				defer func() {
					if isLastCommit {
						return
					}

					err = os.WriteFile(fp, originalFileContents.bytes, originalFileContents.mode)
					if err != nil {
						log.Fatalf("failed to roll-back the file: %s", err)
					}
				}()

				_, err = gitWorktree.Add(fp)
				if err != nil {
					return fmt.Errorf("failed to operate git-add for '%s': %w", fp, err)
				}

				return nil
			}()

			if err != nil {
				return err
			}
		}

		commitLogs, err := repo.Log(&git.LogOptions{
			From: plumbing.NewHash(commitID),
		})
		if err != nil {
			return fmt.Errorf("failed to retrieve the commit logs of '%s': %w", commitLogs, err)
		}

		commitLog, err := commitLogs.Next()
		if err != nil {
			return fmt.Errorf("failed to retrieve the commit log of '%s': %w", commitLogs, err)
		}

		commitOpt := &git.CommitOptions{}
		err = commitOpt.Validate(repo)
		if err != nil {
			return fmt.Errorf("failed to validate the git commit option: %w", err)
		}
		commitOpt.Author = &commitLog.Author

		committedID := plumbing.NewHash("0000000000000000000000000000000000000000")
		commitMessage := fmt.Sprintf(`[git-theseus] %s
git-theseus does this migration commit.
The original commit is %s`, commitLog.Message, commitLog.Hash)

		if dryrun {
			fmt.Printf(`commit %s
Author: %s <%s>
Date:   %s

    %s
`, committedID, commitOpt.Author.Name, commitOpt.Author.Email, commitOpt.Author.When.Format("Mon Jan 02 15:04:05 2006 -0700"), commitMessage)

			err = exec.Command("git", "reset").Run()
			if err != nil {
				return fmt.Errorf("failed to discard the staged changes: %w", err)
			}
		} else {
			committedID, err = gitWorktree.Commit(commitMessage, commitOpt)
			if err != nil {
				return fmt.Errorf("failed to do git-commit: %w", err)
			}
		}

		log.Printf("commited: %s (dryrun: %v)", committedID, dryrun)
	}

	for fp, originalFileContents := range originalFileContentsMap {
		err = os.WriteFile(fp, originalFileContents.bytes, originalFileContents.mode)
		if err != nil {
			log.Fatalf("failed to restore the file on wrap-up: %s", err)
		}
	}

	return nil
}

func extractCommitIDs(commits CommitToDiffs) []string {
	i := 0
	commitIDs := make([]string, len(commits))
	for cid := range commits {
		commitIDs[i] = cid
		i++
	}

	return commitIDs
}

func sortCommitIDsTopologicallyAsAsc(commitIDs []string) ([]string, error) {
	// ref: https://stackoverflow.com/questions/22714371/how-can-i-sort-a-set-of-git-commit-ids-in-topological-order
	args := []string{"rev-list", "--topo-order", "--reverse", "--no-walk"}
	args = append(args, commitIDs...)

	cmd := exec.Command("git", args...)
	sortedOut, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute the command '%s', %w", cmd, err)
	}

	sorted := make([]string, len(commitIDs))
	scanner := bufio.NewScanner(bytes.NewReader(sortedOut))
	scanner.Split(bufio.ScanLines)

	i := 0
	for scanner.Scan() {
		sorted[i] = scanner.Text()
		i++
	}

	return sorted, nil
}
