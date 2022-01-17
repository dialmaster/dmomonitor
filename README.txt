Usage:

This utility is designed to be used to display the realtime mining status of DynMiner2.exe miners
as well as DMO coin mining statistics for specific DMO Wallets linked to a running DMO full node.

In order to use it, you will need to do 2 things:
1) Setup the config.yaml configuration file with information about your fullnode and the wallets you want to display statistics for.
2) Run your miners with 2 additional configuration options to let them connect to this monitoring program. The miners will just need to be given 2 additional command line options in order to connect to the server. The URL for your monitoring program and a 'vanity name' to display in the monitor, eg:
-statrpcurl http://123.456.789.111:11235/minerstats -minername TestMiner

3) Open the configured ServerPort on the host machine that is running the monitor
4) Run the dmo-monitor executable

Once you run the executable it will immediately start to display statistics in the console. Text is formatted and colored using ANSI escape codes, so if your terminal does not support those formatting will be a little wonky, but that's fine because the web interface is much nicer looking anyhow.

This has been tested with powershell, git bash and cmd on Windows.

You can also access a webview that is mobile friendly by visiting your server IP address/port like:

localhost:11235 (11235 is the default port setup in the included config, but you can change it to whatever you like)
You can also access this view from other computers or phones by using the IP address for the computer it is running, or by forwarding out the configured port on your router firewall and the IP address assigned by your ISP.


To build the executable:
Simply run "go build"

