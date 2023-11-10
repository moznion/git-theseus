package gittheseus

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func rollbackTestRepo() error {
	cmd := exec.Command("git", "reset", "--hard", "b6f4744e3813777569a9ac390a5c2608b800bfb7")
	cmd.Dir = "./git-theseus-test-repo/"
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
func TestApp(t *testing.T) {
	defer func() {
		_ = rollbackTestRepo()
	}()

	err := rollbackTestRepo()
	assert.NoError(t, err)

	err = exec.Command("sh", "git-theseus-test-repo/code_generator.sh").Run()
	assert.NoError(t, err)

	cmd := exec.Command("go", "run", "../cmd/git-theseus/main.go", "--input-file", "git-theseus.json")
	cmd.Dir = "./git-theseus-test-repo/"
	err = cmd.Run()
	assert.NoError(t, err)

	gitUserName, _ := exec.Command("git", "config", "user.name").Output()
	gitEmail, _ := exec.Command("git", "config", "user.email").Output()

	cmd = exec.Command("git", "show", "HEAD^^", "--pretty=fuller")
	cmd.Dir = "./git-theseus-test-repo/"
	out, err := cmd.Output()
	assert.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`^commit [0-9a-f]{40}
Author:     moznion <moznion@mail.moznion.net>
AuthorDate: Wed Sep 27 18:56:49 2023 [+]0900
Commit:     `+strings.TrimSpace(string(gitUserName))+" <"+strings.TrimSpace(string(gitEmail))+`>
CommitDate: .+
\s*
    \[git-theseus\] First commit
\s*
    git-theseus does this migration commit.
    The original commit is b36384d2da65869dce07f09c204d2e5407ee0dad
\s*
diff --git a/bar_new b/bar_new
new file mode 100644
index 0000000..[0-9a-f]{7}
--- /dev/null
[+]{3} b/bar_new
@@ -0,0 [+]1,2 @@
[+]3-A # original file's line is 3
[+]3-B # original file's line is 3
\\ No newline at end of file
diff --git a/foo_new b/foo_new
new file mode 100644
index 0000000..[0-9a-f]{7}
--- /dev/null
[+]{3} b/foo_new
@@ -0,0 [+]1,4 @@
[+]1-A # original file's line is 1
[+]1-B # original file's line is 1
[+]2-A # original file's line is 2
[+]3-A # original file's line is 3
\\ No newline at end of file
$`), string(out))

	cmd = exec.Command("git", "show", "HEAD^", "--pretty=fuller")
	cmd.Dir = "./git-theseus-test-repo/"
	out, err = cmd.Output()
	assert.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`^commit [0-9a-f]{40}
Author:     dummy <dummy@example.com>
AuthorDate: Wed Sep 27 18:57:22 2023 [+]0900
Commit:     `+strings.TrimSpace(string(gitUserName))+" <"+strings.TrimSpace(string(gitEmail))+`>
CommitDate: .+
\s*
    \[git-theseus\] Second commit
\s*
    git-theseus does this migration commit.
    The original commit is 9c4fe1bc69832dd26f980c2c8530964d32d1e98b
\s*
diff --git a/foo_new b/foo_new
index [0-9a-f]{7}..[0-9a-f]{7} 100644
--- a/foo_new
[+]{3} b/foo_new
@@ -1,4 [+]1,7 @@
 1-A # original file's line is 1
 1-B # original file's line is 1
 2-A # original file's line is 2
-3-A # original file's line is 3
\\ No newline at end of file
[+]3-A # original file's line is 3
[+]5-A # original file's line is 5
[+]5-B # original file's line is 5
[+]6-A # original file's line is 6
\\ No newline at end of file
$`), string(out))

	cmd = exec.Command("git", "show", "HEAD", "--pretty=fuller")
	cmd.Dir = "./git-theseus-test-repo/"
	out, err = cmd.Output()
	assert.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`^commit [0-9a-f]{40}
Author:     moznion <moznion@mail.moznion.net>
AuthorDate: Wed Sep 27 18:57:38 2023 [+]0900
Commit:     `+strings.TrimSpace(string(gitUserName))+" <"+strings.TrimSpace(string(gitEmail))+`>
CommitDate: .+
\s*
    \[git-theseus\] Third commit
\s*
    git-theseus does this migration commit.
    The original commit is 7b0521555ba48ccc561dada09b2baf7039f87234
\s*
diff --git a/bar_new b/bar_new
index [0-9a-f]{7}..[0-9a-f]{7} 100644
--- a/bar_new
[+]{3} b/bar_new
@@ -1,2 [+]1,4 @@
[+]1-A # original file's line is 1
[+]2-A # original file's line is 2
 3-A # original file's line is 3
 3-B # original file's line is 3
\\ No newline at end of file
diff --git a/foo_new b/foo_new
index [0-9a-f]{7}..[0-9a-f]{7} 100644
--- a/foo_new
[+]{3} b/foo_new
@@ -2,6 [+]2,7 @@
 1-B # original file's line is 1
 2-A # original file's line is 2
 3-A # original file's line is 3
[+]4-A # original file's line is 4
 5-A # original file's line is 5
 5-B # original file's line is 5
 6-A # original file's line is 6
\\ No newline at end of file
$`), string(out))
}
