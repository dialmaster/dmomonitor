Overview:

This package is designed to be used in conjunction with DynMiner2.exe to facilitate showing the current status 
of running miners, including their hashrate, submits, rejects and accepts. It can also be used to notity via
Telegram when a miner stops reporting in.

It has additional functionality to accept Wallet names and connect to a fullnode to give reporting on 
mining statistics. This additional functionality is really only useful for solo mining at this time.


Files:

This contains binaries for both Windows and Linux for the monitoring program.
It also contains a Windows ONLY build of the 2.05 DynMiner2.exe and dyn_miner2.cl for use with this monitor
Hopefully the changes needed for the miner are integrated into the Foundation build for the miner soon...

Windows: dmo-monitor.exe
Linux: dmo-monitor
The js/ and templates/ directories and their contents are required as well and should be placed in the same place
relative to the executable as they are in this git repository.


Setup:

In order to set this up, there are only a few things you need to do:

 1) Copy config.yaml to myconfig.yaml
 Edit/setup the myconfig.yaml configuration file with your specific config. There are notes in the configuration file 
 that you can use as a guide for doing this.

 2) Make sure you are using a version of DynMiner2.exe that supports reporting. I have included a version built from the Foundation
 2.05 in this repo for Windows. You will need to ensure that you use the dyn_miner2.cl that is included with it as well.
 
 3) You will likely need to open the port in your firewall that you configure the dmo-monitor to use. The default is 11235
 If you want to access it from outside your local network (away from home), then you will need to forward that port on your
 router to the machine that is running dmo-monitor as well.


Usage:

 1) Once you have setup config.yaml, you can simply run dmo-monitor in a console window. It uses ANSI codes for formatting, so
 I recommend you use a terminal application that supports them, but it is not required. You can run the dmo-monitor without
 setting up your miners or anything else, but at that point it won't start reporting anything.

 2) When you run your DynMiner2.exe program you will just need to pass 2 additional command line options in order to connect 
to the dmo-monitor: The URL for your monitoring program and a 'vanity name' to display in the monitor, 
eg these are the two additional parameters to add to your DynMiner2.exe line:
-statrpcurl http://123.456.789.111:11235/minerstats -minername TestMiner

So the full line would look like:
DynMiner2.exe -mode solo -server http://192.168.1.169:6433 -user username -pass password -wallet dy1qvesdfsdfsdfsdfsdfsdfefczsvf -miner GPU,16384,8,0,0 -statrpcurl http://192.168.1.133:11235/minerstats -minername TestMiner

The -statrpcurl argument should use the IP address of the machine the dmo-monitor is running on and the port configured for that dmo-monitor in config.yaml. The '/minerstats' part should be left as-is

Once you run the executable it will immediately start to display statistics in the console. 

You can also access a webview that is mobile friendly by visiting your server IP address/port 
from the machine that is running dmo-monitor via:
http://localhost:11235/ 
(11235 is the default port setup in the included config, but you can change it to whatever you like)
You can also access this view from other computers or phones by using the IP address for the computer it is running dmo-monitor.
This will look like (as an example):
http://192.168.1.150:11235/

If you want to access it from outside your house, you will need to have forwarded the port on your router and then you would access it using 
the IP address assigned by your ISP.


NOTES:

Displaying Wallet Statistics:
If you want to display statistics on your mining wallets, the WalletsToMonitor will need to be Wallet Names that are setup on 
the full node you are mining against. There is no support for pool mining for these statistics yet. 
There is no support for using receiving addresses yet either.

IF ALL YOU WANT TO DO IS MONITOR YOUR ACTIVE MINERS and their hashrates, submits, rejects and accepts, 
then you should just leave the WalletsToMonitor config option BLANK like:
WalletsToMonitor: 



To build the executable:
Go Version used for build: 1.17.5

go get gopkg.in/yaml.v2
go build


