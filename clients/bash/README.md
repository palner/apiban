# Client - bash #

Use the GO client if you can... the bash script is suitable for a template. **Not recommended for production.**

Bash script to check apiban API and block returned IP addresses with **iptables**.

## How to use ##

1. Download apiban.sh and apibanconfig.sys
2. Make sure `jq` is installed on your system (`apt install jq`)
3. Replace `MYAPIKEY` in apibanconfig.sys with your apiban api key
4. Run `chmod +x apiban.sh`
5. Run `./apiban.sh` as needed (cron recommended)

## How it works ##

The client pulls the API key and last known ID from the **apibanconfig.sys** file.

When the script is executed, it first checks to see if the **APIBAN** chain exists in iptables. If the chain does not exist, it is recreated and the **LKID** is reset (allowing a full dump).

IP addresses are added to APIBAN chain and actions are logged in **apiban-client.log**.
