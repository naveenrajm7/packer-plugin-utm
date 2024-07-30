on run argv
  set vmName to item 1 of argv # Name of the VM
  tell application "UTM"
    set vm to virtual machine named vmName
    set config to configuration of vm

    -- Initialize the network interfaces list to empty
    set updatedNetworkInterfaces to {}

    -- Update the config with the empty network interfaces
    set network interfaces of config to updatedNetworkInterfaces

    -- Update the VM configuration with the new network interface
    update configuration of vm with config
  end tell
end run