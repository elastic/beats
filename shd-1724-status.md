##

if it's a config save/load problem, and it seems to happen when the agent loads
from disk then sends to filebeat, then if the agent starts without internet connection
it should send whatever ti could load from the disk.

tried to boot and reboot the VM without internet, so far, no luck

logged the config the agent is saving on disk -> needs to check the logs


