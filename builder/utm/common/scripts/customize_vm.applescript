on run argv
    tell application "UTM"
        set vmName to item 1 of argv -- VM name is given as the first argument
        set cpuCount to 0
        set memorySize to 0
        set vmNotes to ""
        set directoryShareMode to null

        -- Parse arguments
        repeat with i from 2 to (count argv)
            set currentArg to item i of argv
            if currentArg is "--cpus" then
                set cpuCount to item (i + 1) of argv
            else if currentArg is "--memory" then
                set memorySize to item (i + 1) of argv
            else if currentArg is "--notes" then
                set vmNotes to item (i + 1) of argv
            else if currentArg is "--directory-share-mode" then
                set directoryShareMode to item (i + 1) of argv
            end if
        end repeat
        
        -- Get the VM and its configuration
        set vm to virtual machine named vmName -- Name is assumed to be valid
        set config to configuration of vm

        -- Set CPU count if provided
        if cpuCount is not 0 then
            set cpu cores of config to cpuCount
        end if
        
        -- Set memory size if provided
        if memorySize is not 0 then
            set memory of config to memorySize
        end if
        
        -- Set the notes if --notes is provided (existing notes will be overwritten)
        if vmNotes is not "" then
            set notes of config to vmNotes
        end if

        -- Set Directory Sharing mode if provided
        if directoryShareMode is not null then
            set directory share mode of config to directoryShareMode -- mode is assumed to be enum value
        end if

        -- Save the configuration
        update configuration of vm with config
 
    end tell
end run