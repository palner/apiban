# Apiban Clients #

Sample clients have been provided in a simple bash script or golang (go). The basic concept would be to create a chain in IPTABLES called APIBAN and have the clients executed via `crontab`.

The GO client has been tested more than the bash script. It is recommended that the bash only be used as a template.

The GO client is provided as both source code to build and an executable suitable for most nix environments. It assumes that it will be run in `/usr/local/bin/apiban/`.

_**UPDATES 2020-09-01**_

* added nftables client for bash(tested on debian 9/10)
* Go nftables client coming soon!

## Using the GO executable ##

You can build the client using go, or just use the pre-built executable: (for Raspberry Pi users, there's a compiled executable in the release assets or see below for building on a Pi)

### Quick and Easy Install Instructions ###

1. Create the folder `/usr/local/bin/apiban`
    * `mkdir /usr/local/bin/apiban`
2. Download apiban-iptables-client to `/usr/local/bin/apiban/`
    * `cd /usr/local/bin/apiban`
    * `wget https://github.com/palner/apiban/raw/v0.7.0/clients/go/apiban-iptables-client`
3. Download `config.json` to `/usr/local/bin/apiban/`
    * `cd /usr/local/bin/apiban`
    * `wget https://raw.githubusercontent.com/palner/apiban/v0.7.0/clients/go/apiban-iptables/config.json`
4. Using your favorite text editor, update `config.json` with your APIBAN key
5. Give apiban-iptables-client execute permission
    * `chmod +x /usr/local/bin/apiban/apiban-iptables-client`
6. Test
    * `./usr/local/bin/apiban/apiban-iptables-client`

### Notes ###

**If upgrading from an older version, please add "FLUSH":"200" to your config.json.**

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
PATH=/usr/local/sbin:/usr/local/bin:/sbin:/bin:/usr/sbin:/usr/bin
*/4 * * * * /usr/local/bin/apiban/apiban-iptables-client >/dev/null 2>&1
```

## Building on Raspbian Buster ##

Since the version of `go` that's in Buster is too old to build `apiban-iptables-client`, here's a simple workaround for `go`.

```
cd /usr/local/src
wget https://golang.org/dl/go1.14.7.linux-armv6l.tar.gz
tar -xzvf go1.14.7.linux-armv6l.tar.gz
ln -sfn /usr/local/src/go/bin/go /usr/bin/go
```

Then building of `apiban-iptables-client` is now possible.

```
cd /usr/local/src
git clone https://github.com/palner/apiban
cd apiban/clients/go/apiban-iptables
go build apiban-iptables-client.go
```

### How it works ###

The client pulls the API key and last known ID from the **config.json** file.

When executed, the client first checks to see if the **APIBAN** chain exists in iptables. If the chain does not exist, the APIBAN chain is recreated and the **LKID** is reset (allowing a full dump).

IP addresses are added to APIBAN chain and actions are logged in **apiban-client.log**.

By using the last known ID (LKID), only new addresses are pulled (if any); making the process incredibly more efficient. The client will not add duplicate addresses and a full download can be run manually by adding `FULL` as a command line argument (example: `./usr/local/bin/apiban-iptables-client FULL`). The FULL option is great should the system (or iptables) have been restarted.

## License / Warranty ##

apiban-iptables-client is free software; you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation; either version 2 of the License, or (at your option) any later version

apiban-iptables-client is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.
