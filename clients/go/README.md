# Apiban Clients #

## Using the GO executable ##

1. Create the folder `/usr/local/bin/apiban`
2. Download apiban-iptables-client to `/usr/local/bin/apiban/`
3. Download `config.json` to `/usr/local/bin/apiban/`
4. Update `config.json` with your APIBAN key
5. Run `chmod +x /usr/local/bin/apiban/apiban-iptables-client`
6. Test with `./usr/local/bin/apiban/apiban-iptables-client`

### Notes ###

Log output is saved to `/var/log/apiban-client`. Want to rotate the log? Here's an example...

```bash
cat > /etc/logrotate.d/apiban-client << EOF
/var/log/apiban-client.log {
        daily
        copytruncate
        rotate 7
        compress
}
EOF
```

Example crontab running every 4 min...

```bash
# update apiban iptables
*/4 * * * * /usr/local/bin/apiban/apiban-iptables-client >/dev/null 2>&1
```

## How it works ##

The client pulls the API key and last known ID from the **config.json** file.

When executed, the client first checks to see if the **APIBAN** chain exists in iptables. If the chain does not exist, the APIBAN chain is recreated and the **LKID** is reset (allowing a full dump).

IP addresses are added to APIBAN chain and actions are logged in **apiban-client.log**.

By using the last known ID (LKID), only new addresses are pulled (if any); making the process incredibly more efficient. The client will not add duplicate addresses and a full download can be run manually by adding `FULL` as a command line argument (example: `./usr/local/bin/apiban-iptables-client FULL`). The FULL option is great should the system (or iptables) have been restarted.

## License / Warranty ##

apiban-iptables-client is free software; you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation; either version 2 of the License, or (at your option) any later version

apiban-iptables-client is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.
