########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

## NOTE: Run with Admin Powershell Terminal!
$wsl_ip = (wsl hostname -I).split(" ")[0]
echo "Ubuntu IP: $wsl_ip"

# WSL Port Forwarding Port 22 AND 2223
netsh interface portproxy add v4tov4 listenaddress=0.0.0.0 listenport=22 connectport=22 connectaddress=$wsl_ip
netsh interface portproxy add v4tov4 listenaddress=0.0.0.0 listenport=2223 connectport=2223 connectaddress=$wsl_ip
netsh interface portproxy show all
#  WSL Add firewall rules

netsh advfirewall firewall add rule name="Open SSH Tunnel port 2223 for WSL2" dir=in action=allow protocol=TCP localport=2223
netsh advfirewall firewall add rule name="Open SSH port 22 for WSL2" dir=in action=allow protocol=TCP localport=22
