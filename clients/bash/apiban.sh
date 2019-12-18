#! /bin/bash
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
APIKEY=`grep "APIKEY" apibanconfig.sys | cut -d '=' -f 2`
LKID=`grep "LKID" apibanconfig.sys | cut -d '=' -f 2`

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
CURRIPS=$(iptables -S APIBAN | awk '$1 !="-P"' | awk '{print $4}' | awk '{gsub("/32", "");print}')
if [ -z "$CURRIPS" ] ; then
    echo "$NOW - Making target chain, resetting LKID." >> $LOG
    LKID=100
    iptables -N APIBAN
    iptables -I INPUT -j APIBAN
fi

IPADDRESS=$(curl -s https://apiban.org/api/$APIKEY/banned/$LKID | jq -r ".ipaddress?")
CURRID=$(curl -s https://apiban.org/api/$APIKEY/banned/$LKID | jq -r ".ID?")

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
IPADDRESS=${IPADDRESS//$'\n'/}
IPADDRESS=${IPADDRESS//$'\"'/}
IPADDRESS=${IPADDRESS//$'['/}
IPADDRESS=${IPADDRESS//$']'/}
IPADDRESS=${IPADDRESS//$', '/}

IPADDRESSARR=($IPADDRESS)

for i in "${IPADDRESSARR[@]}"
do
  NOW=$(date +"%Y-%m-%d %H:%M:%S")
  if [[ $CURRIPS =~ "$i" ]]; then
    echo "$NOW - $i already in APIBAN chain. Bad LKID?" >> $LOG
  else
    iptables -I APIBAN -s $i -j DROP
    echo "$NOW - Adding $i to iptables" >> $LOG
  fi
done

echo "$NOW - All done. Exiting." >> $LOG