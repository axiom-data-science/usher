# usher

(re-)organize files in a source monitored directory to a target destination directory.

Source files are mapped to destination files using mappers. Mappers can be written directory in go
or in any scriping language and called as an external executable. See External Mappers below for more details.

## Features

* Recursively process entire source directory or watch for new files
* Use source file attributes last modified time, date in filename, or file content to determine target directory hierarchy and/or filename
* Use hard links to avoid duplicating file storage between source and target directory (source files can be deleted without affecting target directory!)
* Monitor multiple source directories with the same process using a config file
* Single binary executable distribution

## Example usage

```
Flags:
  -h, --help             Show context-sensitive help.
      --config=STRING    Path to configuration yaml file.
      --debug            Enable debug mode.
      --dry-run          Show actions that would be performed without doing anything.
      --copy             Copy files to destination instead of using hard links.
      --mapper=STRING    FileMapper type or external executable to map files from src to dest.

Commands:
  watch      Recursively watch a directory for new files and process when they are detected.
  process    Recursively process all files in a directory and exit.
```

```
$ mkdir test-data
$ touch -d 2024-06-15 test-data/file1
$ touch -d 2022-03-22 test-data/file2
$ touch -d 2014-08-09 test-data/file3
$ ls -l test-data
total 0
-rw-r--r-- 1 user user 0 Jun 15 00:00 file1
-rw-r--r-- 1 user user 0 Mar 22  2022 file2
-rw-r--r-- 1 user user 0 Aug  9  2014 file3
$ ./usher-example process ./test-data ./out
using file mapper mtime_yyyy_mm_dd
2024/06/15 23:47:37 /some/path/test-data/file1 --link--> /some/path/out/2024/06/15/file1
2024/06/15 23:47:37 /some/path/test-data/file2 --link--> /some/path/out/2022/03/22/file2
2024/06/15 23:47:37 /some/path/test-data/file3 --link--> /some/path/out/2014/08/09/file3
$ tree out
out
├── 2014
│   └── 08
│       └── 09
│           └── file3
├── 2022
│   └── 03
│       └── 22
│           └── file2
└── 2024
    └── 06
        └── 15
            └── file1

10 directories, 3 files
$ ./usher-example process --copy test-data out
using file mapper mtime_yyyy_mm_dd
2024/06/15 23:49:33 /some/path/test-data/file1 --copy--> /some/path/out/2024/06/15/file1
2024/06/15 23:49:33 /some/path/test-data/file2 --copy--> /some/path/out/2022/03/22/file2
2024/06/15 23:49:33 /some/path/test-data/file3 --copy--> /some/path/out/2014/08/09/file3
$ ./usher-example process --dry-run test-data out
using file mapper mtime_yyyy_mm_dd
2024/06/15 23:59:16 /some/path/test-data/file1 --link-dry-run--> /some/path/out/2024/06/15/file1
2024/06/15 23:59:16 /some/path/test-data/file2 --link-dry-run--> /some/path/out/2022/03/22/file2
2024/06/15 23:59:16 /some/path/test-data/file3 --link-dry-run--> /some/path/out/2014/08/09/file3
$ ./usher-example process --debug --dry-run test-data out
known mappers: mtime_yyyy_mm_dd
using file mapper mtime_yyyy_mm_dd
debug: true
dryrun: true
copy: false
mapper: ""
src: /some/path/test-data
dest: /some/path/out
rootpathmappings: {}
watch:
  eventbuffersize: 0

2024/06/15 23:59:30 /some/path/test-data/file1 --link-dry-run--> /some/path/out/2024/06/15/file1
2024/06/15 23:59:30 /some/path/test-data/file2 --link-dry-run--> /some/path/out/2022/03/22/file2
2024/06/15 23:59:30 /some/path/test-data/file3 --link-dry-run--> /some/path/out/2014/08/09/file3

```


## Config files

In addition to command line arguments, config files can be used to supply configuration.
The location of the config file can be specified using `--config`, and `usher-config.yml`
will be used by default if it exists.

This is especially useful when mapping subdirectories under a monitored root directory
to different target directories.

Example config file with a single mapper, increased event buffer size to handle
busy or burst file writes in the source directory, and mulitple directory mappings:

```
mapper: mymapper

watch:
  eventbuffersize: 1000

rootpathmappings:
  dataset1: official-name-for-dataset-1
  asdfadsf: official-name-for-dataset-2
  source-dir/can/be-nested: dest/dir/can/be/nested/too
```

## External mappers

Mappers written directly in go will perform much more efficiently, but mappers can also be
written in any language and called by usher as an external executable.

Given a source file `subdir/exfile` inside a monitored directory `/path/to/src/dir`, the following arguments
are provided when calling an external executable mapper:

* relative source file (`subdir/exfile`)
* absolute source file (`/path/to/src/dir/subdir/exfile`)
* source file basename (`exfile`)

Additionally, if mapped root paths are used, the mapped root source path and dest path are passed as additional arguments:

```
$ cat usher-config.yml
mapper: ./usher-mapper.sh

rootpathmappings:
  subdir: dest/dir
$ cat usher-mapper.sh
#!/bin/bash
# use the content of the first line of the file as a subdirectory
head -n 1 "$2"
$ rm -rf test-dir
$ mkdir -p test-dir/subdir/nestedsubdir/
$ cat <<EOF > test-data/subdir/nestedsubdir/file
important-value
asdf
asdf
EOF
$ ./usher-example process /path/to/src/dir /some/output/dir
2024/06/16 00:36:27 /path/to/src/dir/subdir/nestedsubdir/file --link--> /some/output/dir/dest/dir/important-value
```

* relative source file (`nestedsubdir/exfile`)
* absolute source file (`/path/to/src/dir/subdir/nestedsubdir/file`)
* source file basename (`file`)
* mapped root source path (`subdir`)
* mapped root dest path (`dest/dir`)
