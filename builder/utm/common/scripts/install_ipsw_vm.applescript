on run argv
    tell application "UTM"
        set vm to virtual machine id (item 1 of argv)
        install vm
    end tell
end run
