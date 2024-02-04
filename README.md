# git-theseus [![.github/workflows/check.yml](https://github.com/moznion/git-theseus/actions/workflows/check.yml/badge.svg)](https://github.com/moznion/git-theseus/actions/workflows/check.yml)

A git tool to reconstruct the commit logs.

## Usage

```
$ git theseus -h
Usage of /usr/local/bin/git-theseus:
  -dryrun
        a parameter to instruct it to run as dryrun mode (i.e. no destructive operation on git)
  -input-file string
        [mandatory] a file path to the JSON file
```

the schema of the input JSON file is like the following:

```
{
  "${git-commit-id}": {
    "${target/file/path}": [ ${line-num} ],
    ...
  },
  ...
}
```

example is here: [git-theseus.example.json](./git-theseus.example.json)

## Motivation

When using code transformation tools (e.g., [decaffeinate/decaffeinate](https://github.com/decaffeinate/decaffeinate)) and/or code formatters, a well-known issue arises: git commit logs for the modified code can become cluttered with messages like "transformed!" or "formatted!", obscuring the original reasons for changes. This makes it challenging to track the rationale behind each modification on a line-by-line basis after the code has been transformed, as the original commit messages are lost.

This tool is designed to address these problems by mapping git commits line-by-line from the original to the transformed files and reconstructing the git commit history using this mapped information.

## Example

The example project is here: [git-theseus-test-repo](./git-theseus-test-repo)

For example, let's think doing the code transformation.

`foo` file is like the following and it has the commit history:

```
$ git -P blame foo
^b36384d (moznion 2023-09-27 18:56:49 +0900 1) 1
^b36384d (moznion 2023-09-27 18:56:49 +0900 2) 2
^b36384d (moznion 2023-09-27 18:56:49 +0900 3) 3
7b052155 (moznion 2023-09-27 18:57:38 +0900 4) 4
9c4fe1bc (dummy   2023-09-27 18:57:22 +0900 5) 5
9c4fe1bc (dummy   2023-09-27 18:57:22 +0900 6) 6
```

And the git commit history is like:

```
...
7b05215 Third commit
9c4fe1b Second commit
b36384d First commit
```

After the code transformation, the file `foo_new`, which was transformed from `foo`, is as follows:

**foo_new:**

```
1-A # original file's line is 1
1-B # original file's line is 1
2-A # original file's line is 2
3-A # original file's line is 3
4-A # original file's line is 4
5-A # original file's line is 5
5-B # original file's line is 5
6-A # original file's line is 6
```

In this case, the JSON input file would appear as follows:

```
{
  "b36384d2da65869dce07f09c204d2e5407ee0dad": {
    "foo_new": [1, 2, 3, 4]
  },
  "9c4fe1bc69832dd26f980c2c8530964d32d1e98b": {
    "foo_new": [6, 7, 8]
  },
  "7b0521555ba48ccc561dada09b2baf7039f87234": {
    "foo_new": [5]
  }
}
```

and after running `git-theseus`, the git-blame output for `foo-new` appears like bellow:

```
538f95d5 (moznion 2023-09-27 18:56:49 +0900 1) 1-A # original file's line is 1
538f95d5 (moznion 2023-09-27 18:56:49 +0900 2) 1-B # original file's line is 1
538f95d5 (moznion 2023-09-27 18:56:49 +0900 3) 2-A # original file's line is 2
538f95d5 (moznion 2023-09-27 18:56:49 +0900 4) 3-A # original file's line is 3
0a37f199 (moznion 2023-09-27 18:57:38 +0900 5) 4-A # original file's line is 4
a3099a67 (dummy   2023-09-27 18:57:22 +0900 6) 5-A # original file's line is 5
a3099a67 (dummy   2023-09-27 18:57:22 +0900 7) 5-B # original file's line is 5
a3099a67 (dummy   2023-09-27 18:57:22 +0900 8) 6-A # original file's line is 6
```

and detailed each commit is:

```
commit 51b51c0e7cd51abce2520109288d63d554209aa9 (HEAD -> main)
Author:     moznion <moznion@mail.moznion.net>
AuthorDate: Wed Sep 27 18:57:38 2023 +0900
Commit:     moznion <moznion@mail.moznion.net>
CommitDate: Sat Feb 3 20:04:21 2024 -0800

    [git-theseus] Third commit

    git-theseus does this migration commit.
    The original commit is 7b0521555ba48ccc561dada09b2baf7039f87234

commit 52f2eb5ab13e50ef19a98cdfeb7398e65564ecc7
Author:     dummy <dummy@example.com>
AuthorDate: Wed Sep 27 18:57:22 2023 +0900
Commit:     moznion <moznion@mail.moznion.net>
CommitDate: Sat Feb 3 20:04:21 2024 -0800

    [git-theseus] Second commit

    git-theseus does this migration commit.
    The original commit is 9c4fe1bc69832dd26f980c2c8530964d32d1e98b

commit d278a77d2054c633c46b7fb2f474f9a96f4b9056
Author:     moznion <moznion@mail.moznion.net>
AuthorDate: Wed Sep 27 18:56:49 2023 +0900
Commit:     moznion <moznion@mail.moznion.net>
CommitDate: Sat Feb 3 20:04:21 2024 -0800

    [git-theseus] First commit

    git-theseus does this migration commit.
    The original commit is b36384d2da65869dce07f09c204d2e5407ee0dad

```

As you can see, it restored the commit logs associated with the original file's changes, line by line, along with the original author information.

## How does it work

1. Collect the commit hashes from an input JSON file and sort them by commit order, starting with the oldest.
2. Load the contents of the files described in the input JSON file.
3. Starting with the oldest commit, apply the following processing steps:
   1. Look up the files and their related line numbers in the JSON using the commit hash.
   2. For each file, perform the following:
      1. Accumulate the file lines that are associated with the looked-up line numbers or the lines processed in the previous iteration.
      2. Write the accumulated lines to the specified file path.
      3. Execute "git add" for the file.
   3. Extract the original commit log using the commit hash and interpolate it into the commit message template.
   4. Execute "git commit" for the added files.
4. Finally, as a precaution, restore the original contents to the files.


## Author

moznion (<moznion@mail.moznion.net>)

