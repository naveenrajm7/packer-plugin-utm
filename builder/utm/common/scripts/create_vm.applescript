---
-- create_vm.applescript
-- This script creates a new VM with the specified properties.
-- Usage: osascript create_vm.applescript --name <VM_NAME> --backend <BACKEND> --arch <ARCH> 
-- Example: osascript create_vm.applescript --name "MyVM" --backend "QeMu" --arch "aarch64" 
on run argv
    -- Initialize variables
    set vmName to ""
    set vmBackend to ""
    set vmArch to ""
    set vmIcon to ""

    -- Parse arguments
    repeat with i from 1 to (count argv)
        set currentArg to item i of argv
        if currentArg is "--name" then
            set vmName to item (i + 1) of argv
        else if currentArg is "--backend" then
            set vmBackend to item (i + 1) of argv as string
        else if currentArg is "--arch" then
            set vmArch to item (i + 1) of argv
        else if currentArg is "--icon" then
            set vmIcon to item (i + 1) of argv
        end if
    end repeat

    -- Create a new VM with the specified properties
    tell application "UTM"
      set vm to make new virtual machine with properties ¬
        { backend:vmBackend, ¬
          configuration:{ ¬
            name:vmName, ¬
            architecture:vmArch ¬
          } ¬
        }
      
      -- UTM by default creates a new VM with iso and disk drives
      -- Remove all drives to have an empty VM
      set config to configuration of vm
      set drives of config to {}

      -- Set the VM icon if provided
      if vmIcon is not "" then
          set icon of config to vmIcon
      end if

      -- Update the VM configuration
      update configuration of vm with config

      -- Return the ID of the new VM
      return id of vm
    end tell
end run