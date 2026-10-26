on run argv
    tell application "UTM"
        set vm to virtual machine id (item 1 of argv)
        set installStatus to query install vm
        return ((installing of installStatus) as text) & "|" & ((progress of installStatus) as text)
    end tell
end run
