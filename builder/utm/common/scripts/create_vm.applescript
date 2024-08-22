on run argv
  -- Initialize variables
  set vmName to ""
  set vmBackend to ""
  set vmArch to ""

  -- Parse arguments
  repeat with i from 1 to (count argv)
    set currentArg to item i of argv
    if currentArg is "--name" then
      set vmName to item (i + 1) of argv
    else if currentArg is "--backend" then
      set vmBackend to item (i + 1) of argv as string
    else if currentArg is "--arch" then
      set vmArch to item (i + 1) of argv
    end if
  end repeat

  -- Create a new VM with the specified properties
  tell application "UTM"
    set vm to make new virtual machine with properties {backend:vmBackend, configuration:{name:vmName, architecture:vmArch}}
  end tell
end run