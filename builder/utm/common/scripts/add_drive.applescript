---
-- add_drive.applescript
-- This script adds a drive to a specified UTM virtual machine with given size.
-- Usage: osascript add_drive.applescript <VM_NAME> --size <SIZE>
-- Example: osascript add_drive.applescript test --size 65536

on run argv
  set vmName to item 1 of argv # Name of the VM
  -- Parse the --size argument
  set diskSize to item 3 of argv 

  tell application "UTM"
    -- Get the VM and its configuration
    set vm to virtual machine named vmName -- Name is assumed to be valid
    set config to configuration of vm

    -- Existing drives
    set vmDrives to drives of config
    --- create a new drive
    set newDrive to {guest size: diskSize}
    --- add the drive to the end of the list
    copy newDrive to end of vmDrives
    --- set drives with new drive list
    set drives of config to vmDrives

    --- save the configuration (VM must be stopped)
    update configuration of vm with config
  end tell
end run