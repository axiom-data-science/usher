package usher

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestProcessFile(t *testing.T) {
	// Create a temporary source directory
	srcDir, err := os.MkdirTemp("", "test-src")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(srcDir)

	// Create a temporary destination directory
	destDir, err := os.MkdirTemp("", "test-dest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(destDir)

	// Create a temporary external executable script
	externalScript := []byte(`#!/bin/bash	
echo "external/executable/mapper/$1"
`)
	externalScriptPath := filepath.Join(srcDir, "external.sh")
	os.WriteFile(externalScriptPath, externalScript, 0755)

	// Set up table test conditions/expectations
	tests := []struct {
		config     Config
		destPrefix func(srcFile string) string
		filename   string
	}{
		//simple file mtime
		{
			config: Config{
				Globals: Globals{
					SrcDir:  srcDir,
					DestDir: destDir,
					Debug:   false,
					DryRun:  false,
					Copy:    false,
				},
				FileMapper: NewMtimeFileMapper("2006/01/02/"),
			},
			destPrefix: func(srcFile string) string {
				srcStat, err := os.Stat(srcFile)
				if err != nil {
					t.Fatal(err)
				}
				return srcStat.ModTime().Format("2006/01/02/")
			},
			filename: "mtime.txt",
		},
		//simple file mtime, copy true
		{
			config: Config{
				Globals: Globals{
					SrcDir:  srcDir,
					DestDir: destDir,
					Debug:   false,
					DryRun:  false,
					Copy:    true,
				},
				FileMapper: NewMtimeFileMapper("2006/01/02/"),
			},
			destPrefix: func(srcFile string) string {
				srcStat, err := os.Stat(srcFile)
				if err != nil {
					t.Fatal(err)
				}
				return srcStat.ModTime().Format("2006/01/02/")
			},
			filename: "mtime_copy_true.txt",
		},
		//dry run
		{
			config: Config{
				Globals: Globals{
					SrcDir:  srcDir,
					DestDir: destDir,
					Debug:   false,
					DryRun:  true,
					Copy:    false,
				},
				FileMapper: PassThroughFileMapper,
			},
			destPrefix: func(srcFile string) string {
				return "dry_run"
			},
			filename: "dry_run.txt",
		},
		//date in file name
		{
			config: Config{
				Globals: Globals{
					SrcDir:  srcDir,
					DestDir: destDir,
					Debug:   false,
					DryRun:  false,
					Copy:    false,
				},
				FileMapper: &DelegatingFileMapper{
					GetFileDestPathFunc: func(relSrcFile string, absSrcFile string, baseSrcFile string,
						mappedRootSrcPath string, mappedRootDestPath string) (string, error) {
						// parse time from filename
						fileTime, err := time.Parse("2006-01-02T150405",
							strings.TrimPrefix(strings.Split(baseSrcFile, ".")[0], "prefix-"))
						if err != nil {
							t.Fatal(err)
						}
						return fileTime.Format("2006/2006-01/") + baseSrcFile, nil
					},
				},
			},
			destPrefix: func(srcFile string) string {
				return "2019/2019-03"
			},
			filename: "prefix-2019-03-05T124500.suffix.txt",
		},
		//external executable
		{
			config: Config{
				Globals: Globals{
					SrcDir:  srcDir,
					DestDir: destDir,
					Debug:   false,
					DryRun:  false,
					Copy:    true,
				},
				FileMapper: NewExternalFileMapper(externalScriptPath),
			},
			destPrefix: func(srcFile string) string {
				return "external/executable/mapper"
			},
			filename: "external.txt",
		},
	}

	for _, tc := range tests {
		srcFile := filepath.Join(srcDir, tc.filename)
		if err := os.WriteFile(srcFile, []byte("test content"), 0644); err != nil {
			t.Fatal(err)
		}
		srcFileInfo, err := os.Stat(srcFile)
		if err != nil {
			t.Fatal(err)
		}

		srcStat, ok := srcFileInfo.Sys().(*syscall.Stat_t)
		if !ok {
			t.Fatal("Couldn't get stat info for ", tc.filename)
			return
		}

		// Call the processFile function
		processFile(tc.config, srcFile)

		// Check if the target file exists in the destination directory
		destFile := filepath.Join(tc.destPrefix(srcFile), tc.filename)
		destFileInfo, err := os.Stat(filepath.Join(destDir, destFile))
		if tc.config.DryRun {
			if err == nil {
				t.Fatal("DryRun should not have created the dest file")
			}
			continue
		}

		if err != nil {
			t.Errorf("Expected file %s to exist in destination directory, but got error: %v", destFile, err)
		}

		destStat, ok := destFileInfo.Sys().(*syscall.Stat_t)
		if !ok {
			t.Fatal("Couldn't get stat info for ", destFile)
			return
		}

		if tc.config.Copy == true {
			if srcStat.Ino == destStat.Ino {
				t.Fatal("src and dest files with config copy: true should not have the same inode")
			}
		} else {
			if srcStat.Ino != destStat.Ino {
				t.Fatal("src and dest files with config copy: false should have the same inode")
			}
		}
	}
}
