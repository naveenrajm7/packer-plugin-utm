on run argv
    set vmName to item 1 of argv
    set ipswPath to item 2 of argv
    set cpuCount to (item 3 of argv) as integer
    set memoryMiB to (item 4 of argv) as integer
    tell application "UTM"
        set vm to make new virtual machine with properties {backend:apple, configuration:{name:vmName, os:macOS, macRecoveryIpsw:POSIX file ipswPath, cpu cores:cpuCount, memory:memoryMiB}}
        return id of vm
    end tell
end run
