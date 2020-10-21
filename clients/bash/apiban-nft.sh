#! /bin/bash
# * This file is part of APIBAN.org.
# *
# * apiban-iptables-client is free software; you can redistribute it and/or modify
# * it under the terms of the GNU General Public License as published by
# * the Free Software Foundation; either version 2 of the License, or
# * (at your option) any later version
# *
# * apiban-iptables-client is distributed in the hope that it will be useful,
# * but WITHOUT ANY WARRANTY; without even the implied warranty of
# * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# * GNU General Public License for more details.
# *
# * You should have received a copy of the GNU General Public License
# * along with this program; if not, write to the Free Software
# * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301  USA
NOW=$(date +"%Y-%m-%d %H:%M:%S")

# APIKEY and last known ID are stored in config file
CONFIG=apibanconfig.sys

# Output to a LOD
LOG=apiban-client.log

if [ ! -e "${CONFIG}" ] ; then
    # cant find config file
    echo "does $CONFIG exist?"
    echo "unable to locate config file $CONFIG"
    exit 0
fi

# APIKEY and last known ID are stored in apibanconfig.sys
source $CONFIG

# Exit if no APIKEY
if [ -v "$APIKEY" ] ; then
    echo "$NOW - Cannot determine APIKEY. Exiting." >> $LOG
    exit 0
fi

# If no LKID, make it 100
if [ -v "$LKID" ] ; then
    LKID="100"
fi

# check if chain APIBAN exists
CURRIPS=$(nft list chain ip filter APIBAN | awk '$1 !="-P"' | awk '{print $3}' | awk '{gsub("/32", "");print}' | grep -v filter | grep -v {)
if [ -z "$CURRIPS" ] ; then
    echo "$NOW - Making target chain, resetting LKID." >> $LOG
    LKID=100
    nft add chain ip filter APIBAN
    nft insert rule ip filter INPUT counter jump APIBAN
    nft insert rule ip filter FORWARD counter jump APIBAN
fi

BANLIST=$(curl -s https://apiban.org/api/$APIKEY/banned/$LKID)
IPADDRESS=$(echo $BANLIST | jq -r ".ipaddress? | .[]")
CURRID=$(echo $BANLIST | jq -r ".ID?")

# No new bans
if [ "$CURRID" = "none" ] ; then
    echo "$NOW - No new bans since $LKID. Exiting." >> $LOG
    exit 0
fi

# If CURRID is not numeric, exit.
re='^[0-9]+$'
if ! [[ $CURRID =~ $re ]] ; then
    echo "$NOW - Unexpected response from API ERR1 $CURRID. Exiting." >> $LOG
    exit 1
fi

# update LKID
sed -i "s/^\(LKID=\).*$/\1${CURRID}/" $CONFIG

# parse through IPs
IPADDRESSARR=(${IPADDRESS//$'\"'/})
for i in "${IPADDRESSARR[@]}"
do
  NOW=$(date +"%Y-%m-%d %H:%M:%S")
  if [[ $CURRIPS =~ "$i" ]]; then
    echo "$NOW - $i already in APIBAN chain. Bad LKID?" >> $LOG
  else
    nft insert rule ip filter APIBAN ip saddr $i counter drop
    echo "$NOW - Adding $i to nftables" >> $LOG
  fi
done

echo "$NOW - All done. Exiting." >> $LOG
