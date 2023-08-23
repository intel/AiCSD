########################################################################
 # Copyright (c) Intel Corporation 2023
 # SPDX-License-Identifier: BSD-3-Clause
########################################################################

## NOTE: Run with Admin Powershell Terminal!

# WSL Port Forwarding Removal of Port 22 and 2223
netsh interface portproxy delete v4tov4 listenport=22 listenaddress=0.0.0.0
netsh interface portproxy delete v4tov4 listenport=2223 listenaddress=0.0.0.0

#  WSL Remove firewall rules
netsh advfirewall firewall delete rule name="Open SSH port 22 for WSL2"
netsh advfirewall firewall delete rule name="Open SSH Tunnel port 2223 for WSL2"