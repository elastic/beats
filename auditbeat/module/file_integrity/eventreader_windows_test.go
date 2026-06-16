// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build windows

package file_integrity

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/reader/etw"
	"github.com/elastic/elastic-agent-libs/logp"
)

var saveETWEvents = flag.Bool("save-etw-events", false, "Save ETW events to testdata directory")
var generateSamples = flag.Bool("generate-samples", false, "Generate sample events for testing")

const inactivityTimeout = 5 * time.Second
const initialInactivityTimeout = 15 * time.Second

// TestRegenerateSamples runs various file operations to generate test data.
//
// Usage:
//
//	go test -run TestRegenerateSamples -generate-samples -save-etw-events  # Save raw ETW events
func TestRegenerateSamples(t *testing.T) {
	if !*generateSamples {
		t.SkipNow()
	}
	testChangeOwner(t)
	testChangeTimestamps(t)
	testCreateADS(t)
	testCreateDirectory(t)
	testCreateFile(t)
	testCreateHardlink(t)
	testCreateShortcut(t)
	testCreateSymlink(t)
	testDeleteFile(t)
	testRenameFile(t)
	testSetEA(t)
	testWriteADS(t)
	testWriteFile(t)
}

func testCreateDirectory(t *testing.T) {
	runTest := func(t *testing.T, tempDir string) {
		subDir := filepath.Join(tempDir, "MySubDir")

		runPowershell(t, fmt.Sprintf(`
			Write-Host "-> Creating directory: %s"
			New-Item -Path '%s' -ItemType Directory | Out-Null
		`, subDir, subDir))
	}
	testFSNotifyReaderOperation(t, "create-directory", nil, runTest)
	testETWReaderOperation(t, "create-directory", nil, runTest)
}

func testCreateFile(t *testing.T) {
	runTest := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")

		runPowershell(t, fmt.Sprintf(`
			Write-Host "-> Creating file: %s"
			# Use New-Item for atomic file creation - more predictable than low-level APIs
			New-Item -Path '%s' -ItemType File | Out-Null
		`, mainFile, mainFile))
	}
	testFSNotifyReaderOperation(t, "create-file", nil, runTest)
	testETWReaderOperation(t, "create-file", nil, runTest)
}

func testWriteFile(t *testing.T) {
	setup := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")

		// Create file first (before starting readers) using reliable method
		runPowershell(t, fmt.Sprintf(`
			New-Item -Path '%s' -ItemType File | Out-Null
		`, mainFile))
	}
	runTest := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")
		runPowershell(t, fmt.Sprintf(`
			Write-Host "-> Writing to file: %s"
			# Use direct uncached file write
			$bytes = [System.Text.Encoding]::UTF8.GetBytes("main file data")
			[System.IO.File]::WriteAllBytes('%s', $bytes)
		`, mainFile, mainFile))
	}
	testFSNotifyReaderOperation(t, "write-file", setup, runTest)
	testETWReaderOperation(t, "write-file", setup, runTest)
}

func testCreateADS(t *testing.T) {
	setup := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")

		// Create file first (before starting readers) using reliable method
		runPowershell(t, fmt.Sprintf(`
			New-Item -Path '%s' -ItemType File | Out-Null
		`, mainFile))
	}
	runTest := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")
		runPowershell(t, fmt.Sprintf(`
			Write-Host "-> Creating ADS: %s:MyStream"
			# ADS creation requires PowerShell cmdlets - .NET File class doesn't support ADS paths
			Set-Content -Path '%s' -Stream MyStream -Value "ads data"
		`, mainFile, mainFile))
	}
	testFSNotifyReaderOperation(t, "create-ads", setup, runTest)
	testETWReaderOperation(t, "create-ads", setup, runTest)
}

func testWriteADS(t *testing.T) {
	setup := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")

		// Create file first (before starting readers) using reliable method
		runPowershell(t, fmt.Sprintf(`
			New-Item -Path '%s' -ItemType File | Out-Null
			# Create initial ADS using PowerShell cmdlets
			Set-Content -Path '%s' -Stream MyStream -Value "ads data"
		`, mainFile, mainFile))
	}
	runTest := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")

		runPowershell(t, fmt.Sprintf(`
			Write-Host "-> Writing to ADS: %s:MyStream"
			# ADS write requires PowerShell cmdlets - .NET File class doesn't support ADS paths
			Set-Content -Path '%s' -Stream MyStream -Value "new ads data"
		`, mainFile, mainFile))
	}
	testFSNotifyReaderOperation(t, "write-ads", setup, runTest)
	testETWReaderOperation(t, "write-ads", setup, runTest)
}

func testCreateShortcut(t *testing.T) {
	setup := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")

		// Create file first (before starting readers) using reliable method
		runPowershell(t, fmt.Sprintf(`
			New-Item -Path '%s' -ItemType File | Out-Null
		`, mainFile))
	}
	runTest := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")
		shortcutPath := filepath.Join(tempDir, "MyShortcut.lnk")
		absTarget, _ := filepath.Abs(mainFile)
		runPowershell(t, fmt.Sprintf(`
			Write-Host "-> Creating shortcut: %s"
			$WshShell = New-Object -ComObject WScript.Shell
			$Shortcut = $WshShell.CreateShortcut('%s')
			$Shortcut.TargetPath = '%s'
			$Shortcut.Save()
			# Force file system flush
			[System.GC]::Collect()
			[System.GC]::WaitForPendingFinalizers()
		`, shortcutPath, shortcutPath, absTarget))
	}
	testFSNotifyReaderOperation(t, "create-shortcut", setup, runTest)
	testETWReaderOperation(t, "create-shortcut", setup, runTest)
}

func testCreateHardlink(t *testing.T) {
	setup := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")

		// Create target file first (before starting readers) using reliable method
		runPowershell(t, fmt.Sprintf(`
			New-Item -Path '%s' -ItemType File | Out-Null
		`, mainFile))
	}
	runTest := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")
		hardlinkPath := filepath.Join(tempDir, "hardlink.txt")
		runPowershell(t, fmt.Sprintf(`
			Write-Host "-> Creating hard link: %s"
			# Use Win32 API for direct hardlink creation
			Add-Type -TypeDefinition '
				using System;
				using System.Runtime.InteropServices;
				public class Win32 {
					[DllImport("kernel32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
					public static extern bool CreateHardLink(string lpFileName, string lpExistingFileName, IntPtr lpSecurityAttributes);
				}
			'
			$result = [Win32]::CreateHardLink('%s', '%s', [IntPtr]::Zero)
			if (-not $result) {
				# Fallback to PowerShell method
				New-Item -Path '%s' -ItemType HardLink -Value '%s' | Out-Null
			}
		`, hardlinkPath, hardlinkPath, mainFile, hardlinkPath, mainFile))
	}
	testFSNotifyReaderOperation(t, "create-hardlink", setup, runTest)
	testETWReaderOperation(t, "create-hardlink", setup, runTest)
}

func testCreateSymlink(t *testing.T) {
	setup := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")

		// Create target file first (before starting readers) using reliable method
		runPowershell(t, fmt.Sprintf(`
			New-Item -Path '%s' -ItemType File | Out-Null
		`, mainFile))
	}
	runTest := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")
		symlinkPath := filepath.Join(tempDir, "symlink.txt")

		runPowershell(t, fmt.Sprintf(`
			Write-Host "-> Creating symbolic link: %s"
			# Use Win32 API for direct symlink creation
			Add-Type -TypeDefinition '
				using System;
				using System.Runtime.InteropServices;
				public class Win32 {
					[DllImport("kernel32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
					public static extern bool CreateSymbolicLink(string lpSymlinkFileName, string lpTargetFileName, int dwFlags);
					public const int SYMBOLIC_LINK_FLAG_FILE = 0x0;
				}
			'
			$result = [Win32]::CreateSymbolicLink('%s', '%s', [Win32]::SYMBOLIC_LINK_FLAG_FILE)
			if (-not $result) {
				# Fallback to PowerShell method
				New-Item -Path '%s' -ItemType SymbolicLink -Value '%s' | Out-Null
			}
		`, symlinkPath, symlinkPath, mainFile, symlinkPath, mainFile))
	}
	testFSNotifyReaderOperation(t, "create-symlink", setup, runTest)
	testETWReaderOperation(t, "create-symlink", setup, runTest)
}

func testChangeTimestamps(t *testing.T) {
	setup := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")

		// Create file first (before starting readers) using reliable method
		runPowershell(t, fmt.Sprintf(`
			New-Item -Path '%s' -ItemType File | Out-Null
		`, mainFile))
	}
	runTest := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")
		twoDaysAgo := time.Now().Add(-48 * time.Hour).Format(time.RFC3339)

		runPowershell(t, fmt.Sprintf(`
			Write-Host "-> Changing timestamps: %s"
			# Use Win32 API for direct timestamp manipulation
			Add-Type -TypeDefinition '
				using System;
				using System.Runtime.InteropServices;
				using Microsoft.Win32.SafeHandles;
				public class Win32 {
					[DllImport("kernel32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
					public static extern SafeFileHandle CreateFile(
						string lpFileName, uint dwDesiredAccess, uint dwShareMode,
						IntPtr lpSecurityAttributes, uint dwCreationDisposition,
						uint dwFlagsAndAttributes, IntPtr hTemplateFile);
					[DllImport("kernel32.dll", SetLastError = true)]
					public static extern bool SetFileTime(SafeFileHandle hFile, ref long lpCreationTime, IntPtr lpLastAccessTime, ref long lpLastWriteTime);
					public const uint GENERIC_WRITE = 0x40000000;
					public const uint FILE_SHARE_READ = 0x1;
					public const uint FILE_SHARE_WRITE = 0x2;
					public const uint OPEN_EXISTING = 3;
					public const uint FILE_ATTRIBUTE_NORMAL = 0x80;
				}
			'
			
			try {
				$twoDaysAgo = [datetime]::Parse('%s', [System.Globalization.CultureInfo]::InvariantCulture)
				$filetime = $twoDaysAgo.ToFileTime()
				
				$handle = [Win32]::CreateFile('%s', [Win32]::GENERIC_WRITE, 
					[Win32]::FILE_SHARE_READ -bor [Win32]::FILE_SHARE_WRITE,
					[IntPtr]::Zero, [Win32]::OPEN_EXISTING, [Win32]::FILE_ATTRIBUTE_NORMAL, [IntPtr]::Zero)
				
				if (-not $handle.IsInvalid) {
					[Win32]::SetFileTime($handle, [ref]$filetime, [IntPtr]::Zero, [ref]$filetime)
					$handle.Close()
				} else {
					# Fallback to PowerShell method
					(Get-Item '%s').CreationTime = $twoDaysAgo
					(Get-Item '%s').LastWriteTime = $twoDaysAgo
				}
			} catch {
				# Fallback to PowerShell method
				$twoDaysAgo = [datetime]::Parse('%s', [System.Globalization.CultureInfo]::InvariantCulture)
				(Get-Item '%s').CreationTime = $twoDaysAgo
				(Get-Item '%s').LastWriteTime = $twoDaysAgo
			}
		`, mainFile, twoDaysAgo, mainFile, mainFile, mainFile, twoDaysAgo, mainFile, mainFile))
	}
	testFSNotifyReaderOperation(t, "change-timestamps", setup, runTest)
	testETWReaderOperation(t, "change-timestamps", setup, runTest)
}

func testChangeOwner(t *testing.T) {
	setup := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")

		// Create file first (before starting readers) using reliable method
		runPowershell(t, fmt.Sprintf(`
			New-Item -Path '%s' -ItemType File | Out-Null
		`, mainFile))
	}
	runTest := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")
		runPowershell(t, fmt.Sprintf(`
			Write-Host "-> Changing owner: %s"
			# Force direct ACL manipulation without caching
			$acl = Get-Acl '%s'
			$owner = New-Object System.Security.Principal.NTAccount("SYSTEM")
			$acl.SetOwner($owner)
			Set-Acl -Path '%s' -AclObject $acl
			# Force file system flush
			[System.GC]::Collect()
			[System.GC]::WaitForPendingFinalizers()
		`, mainFile, mainFile, mainFile))
	}
	testFSNotifyReaderOperation(t, "change-owner", setup, runTest)
	testETWReaderOperation(t, "change-owner", setup, runTest)
}

func testSetEA(t *testing.T) {
	setup := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")

		// Create file first (before starting readers) using reliable method
		runPowershell(t, fmt.Sprintf(`
			New-Item -Path '%s' -ItemType File | Out-Null
		`, mainFile))
	}
	runTest := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")

		// Use fsutil to set extended attributes on the file
		// This is a Windows-specific operation that should trigger SetEA events
		runPowershell(t, fmt.Sprintf(`
			Write-Host "-> Setting extended attributes: %s"
			# Use fsutil to set extended attributes (EA)
			# This creates a simple EA with name "TestEA" and value "TestValue"
			$bytes = [System.Text.Encoding]::ASCII.GetBytes("TestValue")
			$tempEAFile = Join-Path $env:TEMP "ea_temp.bin"
			
			# Create EA structure: 4 bytes flags + 1 byte name length + name + 2 bytes value length + value
			$eaData = @()
			$eaData += 0x00, 0x00, 0x00, 0x00  # NextEntryOffset (0 = last entry)
			$eaData += 0x00                     # Flags
			$eaData += 0x06                     # EaNameLength (6 for "TestEA")
			$eaData += 0x09, 0x00               # EaValueLength (9 for "TestValue")
			$eaData += [System.Text.Encoding]::ASCII.GetBytes("TestEA")
			$eaData += 0x00                     # Null terminator for name
			$eaData += [System.Text.Encoding]::ASCII.GetBytes("TestValue")
			
			[System.IO.File]::WriteAllBytes($tempEAFile, $eaData)
			
			# Use NtSetEaFile through PowerShell Add-Type
			Add-Type -TypeDefinition '
				using System;
				using System.Runtime.InteropServices;
				using Microsoft.Win32.SafeHandles;
				
				public class NtApi {
					[DllImport("ntdll.dll")]
					public static extern int NtSetEaFile(
						SafeFileHandle FileHandle,
						IntPtr IoStatusBlock,
						IntPtr Buffer,
						uint Length);
						
					[DllImport("kernel32.dll", SetLastError = true, CharSet = CharSet.Auto)]
					public static extern SafeFileHandle CreateFile(
						string lpFileName,
						uint dwDesiredAccess,
						uint dwShareMode,
						IntPtr lpSecurityAttributes,
						uint dwCreationDisposition,
						uint dwFlagsAndAttributes,
						IntPtr hTemplateFile);
						
					public const uint GENERIC_WRITE = 0x40000000;
					public const uint FILE_SHARE_READ = 0x00000001;
					public const uint FILE_SHARE_WRITE = 0x00000002;
					public const uint OPEN_EXISTING = 3;
					public const uint FILE_ATTRIBUTE_NORMAL = 0x80;
					public const uint FILE_FLAG_NO_BUFFERING = 0x20000000;
				}
			'
			
			try {
				$handle = [NtApi]::CreateFile(
					'%s',
					[NtApi]::GENERIC_WRITE,
					[NtApi]::FILE_SHARE_READ -bor [NtApi]::FILE_SHARE_WRITE,
					[IntPtr]::Zero,
					[NtApi]::OPEN_EXISTING,
					[NtApi]::FILE_ATTRIBUTE_NORMAL -bor [NtApi]::FILE_FLAG_NO_BUFFERING,
					[IntPtr]::Zero
				)
				
				if ($handle.IsInvalid) {
					Write-Host "Failed to open file"
				} else {
					$eaBytes = [System.IO.File]::ReadAllBytes($tempEAFile)
					$pinnedArray = [System.Runtime.InteropServices.GCHandle]::Alloc($eaBytes, [System.Runtime.InteropServices.GCHandleType]::Pinned)
					$pointer = $pinnedArray.AddrOfPinnedObject()
					
					$iosb = [System.Runtime.InteropServices.Marshal]::AllocHGlobal(16)  # IO_STATUS_BLOCK
					$result = [NtApi]::NtSetEaFile($handle, $iosb, $pointer, $eaBytes.Length)
					
					[System.Runtime.InteropServices.Marshal]::FreeHGlobal($iosb)
					$pinnedArray.Free()
					$handle.Close()
					
					Write-Host "SetEA result: $result"
				}
			} catch {
				Write-Host "Exception during SetEA: $($_.Exception.Message)"
				# Fallback: try using attrib command which might trigger some attribute changes
				attrib +A '%s'
				Write-Host "Used fallback attrib command"
			} finally {
				if (Test-Path $tempEAFile) {
					Remove-Item $tempEAFile -Force
				}
				# Force file system flush
				[System.GC]::Collect()
				[System.GC]::WaitForPendingFinalizers()
			}
		`, mainFile, mainFile, mainFile))
	}
	testFSNotifyReaderOperation(t, "set-ea", setup, runTest)
	testETWReaderOperation(t, "set-ea", setup, runTest)
}

func testRenameFile(t *testing.T) {
	setup := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")

		// Create file first (before starting readers) using reliable method
		runPowershell(t, fmt.Sprintf(`
			New-Item -Path '%s' -ItemType File | Out-Null
		`, mainFile))
	}
	runTest := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")
		renamedFile := filepath.Join(tempDir, "renamed-file.txt")
		runPowershell(t, fmt.Sprintf(`
			Write-Host "-> Renaming file: %s to %s"
			# Use Win32 API for direct file rename
			Add-Type -TypeDefinition '
				using System;
				using System.Runtime.InteropServices;
				public class Win32 {
					[DllImport("kernel32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
					public static extern bool MoveFile(string lpExistingFileName, string lpNewFileName);
				}
			'
			$result = [Win32]::MoveFile('%s', '%s')
			if (-not $result) {
				# Fallback to PowerShell method
				Rename-Item -Path '%s' -NewName ('%s' | Split-Path -Leaf)
			}
		`, mainFile, renamedFile, mainFile, renamedFile, mainFile, renamedFile))
	}
	testFSNotifyReaderOperation(t, "rename-file", setup, runTest)
	testETWReaderOperation(t, "rename-file", setup, runTest)
}

func testDeleteFile(t *testing.T) {
	setup := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")

		// Create file first (before starting readers) using reliable method
		runPowershell(t, fmt.Sprintf(`
			New-Item -Path '%s' -ItemType File | Out-Null
		`, mainFile))
	}
	runTest := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")
		runPowershell(t, fmt.Sprintf(`
			Write-Host "-> Deleting file: %s"
			# Use Win32 API for direct file deletion
			Add-Type -TypeDefinition '
				using System;
				using System.Runtime.InteropServices;
				public class Win32 {
					[DllImport("kernel32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
					public static extern bool DeleteFile(string lpFileName);
				}
			'
			$result = [Win32]::DeleteFile('%s')
			if (-not $result) {
				# Fallback to PowerShell method
				Remove-Item -Path '%s' -Force
			}
		`, mainFile, mainFile, mainFile))
	}
	testFSNotifyReaderOperation(t, "delete-file", setup, runTest)
	testETWReaderOperation(t, "delete-file", setup, runTest)
}

func testDeleteSymlink(t *testing.T) {
	setup := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")
		symlinkPath := filepath.Join(tempDir, "symlink.txt")

		// Create file and symlink first (before starting readers) using reliable method
		runPowershell(t, fmt.Sprintf(`
			New-Item -Path '%s' -ItemType File | Out-Null
			# Create symlink using Win32 API
			Add-Type -TypeDefinition '
				using System;
				using System.Runtime.InteropServices;
				public class Win32 {
					[DllImport("kernel32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
					public static extern bool CreateSymbolicLink(string lpSymlinkFileName, string lpTargetFileName, int dwFlags);
					public const int SYMBOLIC_LINK_FLAG_FILE = 0x0;
				}
			'
			$result = [Win32]::CreateSymbolicLink('%s', '%s', [Win32]::SYMBOLIC_LINK_FLAG_FILE)
			if (-not $result) {
				New-Item -Path '%s' -ItemType SymbolicLink -Value '%s' | Out-Null
			}
		`, mainFile, symlinkPath, mainFile, symlinkPath, mainFile))
	}
	runTest := func(t *testing.T, tempDir string) {
		symlinkPath := filepath.Join(tempDir, "symlink.txt")

		runPowershell(t, fmt.Sprintf(`
			Write-Host "-> Deleting symbolic link: %s"
			# Use Win32 API for direct symlink deletion
			Add-Type -TypeDefinition '
				using System;
				using System.Runtime.InteropServices;
				public class Win32 {
					[DllImport("kernel32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
					public static extern bool DeleteFile(string lpFileName);
				}
			'
			$result = [Win32]::DeleteFile('%s')
			if (-not $result) {
				# Fallback to PowerShell method
				Remove-Item -Path '%s' -Force
			}
		`, symlinkPath, symlinkPath, symlinkPath))
	}
	testFSNotifyReaderOperation(t, "delete-symlink", setup, runTest)
	testETWReaderOperation(t, "delete-symlink", setup, runTest)
}

func testDeleteHardlink(t *testing.T) {
	setup := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")
		hardlinkPath := filepath.Join(tempDir, "hardlink.txt")

		// Create file and hardlink first (before starting readers) using reliable method
		runPowershell(t, fmt.Sprintf(`
			New-Item -Path '%s' -ItemType File | Out-Null
			# Create hardlink using Win32 API
			Add-Type -TypeDefinition '
				using System;
				using System.Runtime.InteropServices;
				public class Win32 {
					[DllImport("kernel32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
					public static extern bool CreateHardLink(string lpFileName, string lpExistingFileName, IntPtr lpSecurityAttributes);
				}
			'
			$result = [Win32]::CreateHardLink('%s', '%s', [IntPtr]::Zero)
			if (-not $result) {
				New-Item -Path '%s' -ItemType HardLink -Value '%s' | Out-Null
			}
		`, mainFile, hardlinkPath, mainFile, hardlinkPath, mainFile))
	}
	runTest := func(t *testing.T, tempDir string) {
		hardlinkPath := filepath.Join(tempDir, "hardlink.txt")
		runPowershell(t, fmt.Sprintf(`
			Write-Host "-> Deleting hard link: %s"
			# Use Win32 API for direct hardlink deletion
			Add-Type -TypeDefinition '
				using System;
				using System.Runtime.InteropServices;
				public class Win32 {
					[DllImport("kernel32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
					public static extern bool DeleteFile(string lpFileName);
				}
			'
			$result = [Win32]::DeleteFile('%s')
			if (-not $result) {
				# Fallback to PowerShell method
				Remove-Item -Path '%s' -Force
			}
		`, hardlinkPath, hardlinkPath, hardlinkPath))
	}
	testFSNotifyReaderOperation(t, "delete-hardlink", setup, runTest)
	testETWReaderOperation(t, "delete-hardlink", setup, runTest)
}

func testDeleteShortcut(t *testing.T) {
	setup := func(t *testing.T, tempDir string) {
		mainFile := filepath.Join(tempDir, "mainfile.txt")
		shortcutPath := filepath.Join(tempDir, "MyShortcut.lnk")
		absTarget, _ := filepath.Abs(mainFile)

		// Create file and shortcut first (before starting readers)
		runPowershell(t, fmt.Sprintf(`
			New-Item -Path '%s' -ItemType File | Out-Null
			$WshShell = New-Object -ComObject WScript.Shell
			$Shortcut = $WshShell.CreateShortcut('%s')
			$Shortcut.TargetPath = '%s'
			$Shortcut.Save()
		`, mainFile, shortcutPath, absTarget))
	}
	runTest := func(t *testing.T, tempDir string) {
		shortcutPath := filepath.Join(tempDir, "MyShortcut.lnk")
		runPowershell(t, fmt.Sprintf(`
			Write-Host "-> Deleting shortcut: %s"
			Remove-Item -Path '%s' -Force
		`, shortcutPath, shortcutPath))
	}
	testFSNotifyReaderOperation(t, "delete-shortcut", setup, runTest)
	testETWReaderOperation(t, "delete-shortcut", setup, runTest)
}

func testDeleteDirectory(t *testing.T) {
	setup := func(t *testing.T, tempDir string) {
		subDir := filepath.Join(tempDir, "MySubDir")

		// Create directory first (before starting readers)
		runPowershell(t, fmt.Sprintf(`
			New-Item -Path '%s' -ItemType Directory | Out-Null
		`, subDir))
	}
	runTest := func(t *testing.T, tempDir string) {
		subDir := filepath.Join(tempDir, "MySubDir")
		runPowershell(t, fmt.Sprintf(`
			Write-Host "-> Deleting directory: %s"
			Remove-Item -Path '%s' -Force
		`, subDir, subDir))
	}
	testFSNotifyReaderOperation(t, "delete-directory", setup, runTest)
	testETWReaderOperation(t, "delete-directory", setup, runTest)
}

func testFSNotifyReaderOperation(
	t *testing.T,
	operation string,
	setup func(t *testing.T, tempDir string),
	runTest func(t *testing.T, tempDir string),
) {
	t.Run(fmt.Sprintf("fsnotify/%s", operation), func(t *testing.T) {
		dir := t.TempDir()

		if setup != nil {
			t.Logf("Running setup for operation: %s", operation)
			setup(t, dir)
		}

		fsNotifyR, err := NewEventReader(Config{
			Paths:     []string{dir},
			Recursive: true,
		}, logp.NewLogger("test"))
		require.NoError(t, err)

		done := make(chan struct{})

		eventsFS, err := fsNotifyR.Start(done)
		require.NoError(t, err)

		var (
			rcvdFSEvents []any
			wg           sync.WaitGroup
			started      sync.WaitGroup
		)

		consumerCount := 1

		started.Add(consumerCount)
		wg.Add(consumerCount)

		go consumerFunc(t, done, eventsFS, &wg, &started, &rcvdFSEvents)

		started.Wait()
		runTest(t, dir)

		t.Log("Waiting for events to be processed...")
		wg.Wait()

		close(done)

		// Wait for the event channel to close, indicating the producer has shut down
		assert.Eventually(t, func() bool {
			select {
			case _, ok := <-eventsFS:
				return !ok // Channel is closed when ok is false
			default:
				return false // Channel is still open
			}
		}, 10*time.Second, 100*time.Millisecond, "event channel should close after done signal")

		os.RemoveAll(dir)

		writeSamples(t, rcvdFSEvents, fmt.Sprintf("%s_fsnotify.json", operation))
	})
}

func testETWReaderOperation(
	t *testing.T,
	operation string,
	setup func(t *testing.T, tempDir string),
	runTest func(t *testing.T, tempDir string),
) {
	t.Run(fmt.Sprintf("etw/%s", operation), func(t *testing.T) {
		dir := t.TempDir()

		if setup != nil {
			t.Logf("Running setup for operation: %s", operation)
			setup(t, dir)
		}

		etwR, err := newETWReader(Config{
			Paths:         []string{dir},
			Recursive:     true,
			FlushInterval: 2 * time.Second,
		}, logp.NewLogger("test"))
		require.NoError(t, err)
		if *saveETWEvents {
			etwR.(*etwReader).etwEventsC = make(chan *etw.RenderedEtwEvent)
		}

		done := make(chan struct{})

		eventsETW, err := etwR.Start(done)
		require.NoError(t, err)

		var (
			rcvdETWEvents         []any
			rcvdETWOriginalEvents []any
			wg                    sync.WaitGroup
			started               sync.WaitGroup
		)

		consumerCount := 1
		if *saveETWEvents {
			consumerCount++
		}

		started.Add(consumerCount)
		wg.Add(consumerCount)

		if *saveETWEvents {
			go consumeOriginalFunc(t, done, etwR.(*etwReader).etwEventsC, &wg, &started, &rcvdETWOriginalEvents)
		}
		go consumerFunc(t, done, eventsETW, &wg, &started, &rcvdETWEvents)

		started.Wait()
		runTest(t, dir)

		t.Log("Waiting for events to be processed...")
		wg.Wait()

		close(done)

		// Wait for the event channel to close, indicating the producer has shut down
		assert.Eventually(t, func() bool {
			select {
			case _, ok := <-eventsETW:
				return !ok // Channel is closed when ok is false
			default:
				return false // Channel is still open
			}
		}, 10*time.Second, 100*time.Millisecond, "event channel should close after done signal")

		os.RemoveAll(dir)

		// Process and compare/update golden files
		if *saveETWEvents {
			writeSamples(t, rcvdETWOriginalEvents, fmt.Sprintf("%s_etw-original.json", operation))
		}
		writeSamples(t, rcvdETWEvents, fmt.Sprintf("%s_etw.json", operation))
	})
}

func consumerFunc(t *testing.T, done <-chan struct{}, eventsC <-chan Event, wg, started *sync.WaitGroup, events *[]any) {
	timer := time.NewTimer(initialInactivityTimeout)
	defer timer.Stop()
	defer wg.Done()
	started.Done()
	start := time.Now()

	t.Log("Event consumer started, waiting for events...")
loop:
	for {
		select {
		case e := <-eventsC:
			*events = append(*events, buildMetricbeatEvent(&e, false))
			timer.Reset(inactivityTimeout)
		case <-timer.C:
			// Exit loop after a period of inactivity.
			break loop
		case <-done:
			// Exit loop if the test is finishing.
			break loop
		}
	}
	t.Log("Event consumer stopped. Took", time.Since(start))
}

func consumeOriginalFunc(t *testing.T, done <-chan struct{}, eventsC <-chan *etw.RenderedEtwEvent, wg, started *sync.WaitGroup, events *[]any) {
	timer := time.NewTimer(10 * time.Second) // initial timeout
	defer timer.Stop()
	defer wg.Done()
	started.Done()
	start := time.Now()

	t.Log("Event consumer started, waiting for events...")
loop:
	for {
		select {
		case e := <-eventsC:
			*events = append(*events, e)
			timer.Reset(inactivityTimeout)
		case <-timer.C:
			// Exit loop after a period of inactivity.
			break loop
		case <-done:
			// Exit loop if the test is finishing.
			break loop
		}
	}
	t.Log("Event consumer stopped. Took", time.Since(start))
}

// writeSamples writes raw events (used for ETW original events)
func writeSamples(t *testing.T, receivedEvents []any, filename string) {
	goldenPath := filepath.Join("testdata", "samples", filename)

	os.MkdirAll(filepath.Join("testdata", "samples"), 0755)
	data, err := json.MarshalIndent(receivedEvents, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(goldenPath, data, 0644))
	t.Logf("Updated raw events file: %s with %d events", goldenPath, len(receivedEvents))
}

// runPowershell executes a given command string in PowerShell and fails the test on error.
func runPowershell(t *testing.T, command string) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", fmt.Sprintf("%s\nStart-Sleep -Milliseconds 100\n", command))
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Log the command and its output for easier debugging on failure.
		t.Fatalf("Powershell command failed: %v\nOutput:\n%s\n\nCommand:\n%s", err, string(output), command)
	}
}
