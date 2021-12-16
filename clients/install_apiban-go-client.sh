#-- install script for apiban go client
#-- pgpx.io
echo ""
echo ""
echo ""
echo ""
echo ""
echo ""
echo ""
echo " ppppp   ppppppppp      ggggggggg   ggggg"
echo " p::::ppp:::::::::p    g:::::::::ggg::::g"
echo " p:::::::::::::::::p  g:::::::::::::::::g"
echo " pp::::::ppppp::::::pg::::::ggggg::::::gg"
echo "  p:::::p     p:::::pg:::::g     g:::::g "
echo "  p:::::p     p:::::pg:::::g     g:::::g "
echo "  p:::::p     p:::::pg:::::g     g:::::g "
echo "  p:::::p    p::::::pg::::::g    g:::::g "
echo "  p:::::ppppp:::::::pg:::::::ggggg:::::g "
echo "  p::::::::::::::::p  g::::::::::::::::g "
echo "  p::::::::::::::pp    gg::::::::::::::g "
echo "  p::::::pppppppp        gggggggg::::::g "
echo "  p:::::p                        g:::::g "
echo "  p:::::p            gggggg      g:::::g "
echo " p:::::::p           g:::::gg   gg:::::g "
echo " p:::::::p            g::::::ggg:::::::g "
echo " p:::::::p             gg:::::::::::::g  "
echo " ppppppppp               ggg::::::ggg    "
echo "                            gggggg       "
echo ""
echo ""
echo ""
echo " need support? https://palner.com and https://lod.com"
echo ""
echo " Copyright (C) 2021	The Palner Group, Inc. (palner.com)"
echo ""
echo " apiban-iptables-client is free software; you can redistribute it and/or modify"
echo " it under the terms of the GNU General Public License as published by"
echo " the Free Software Foundation; either version 2 of the License, or"
echo " (at your option) any later version"
echo ""
echo " apiban-iptables-client is distributed in the hope that it will be useful,"
echo " but WITHOUT ANY WARRANTY; without even the implied warranty of"
echo " MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the"
echo " GNU General Public License for more details."
echo ""
echo " You should have received a copy of the GNU General Public License"
echo " along with this program; if not, write to the Free Software"
echo " Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301  USA"
echo ""
#-- functions
usage() {
 cat << _EOF_
Usage: ${0} APIBANKEY
...with APIBANKEY being your... apiban api key. ;)

_EOF_
}

echo "-> checking variables"
#-- check arguments and environment
if [ "$#" -ne "1" ]; then
  echo "Expected 1 argument, got $#" >&2
  usage
  exit 2
fi
APIKEY=$1

echo "-> creating apiban directory and downloading client"
mkdir /usr/local/bin/apiban
cd /usr/local/bin/apiban
wget https://github.com/palner/apiban/raw/v0.7.0/clients/go/apiban-iptables-client &>/dev/null
if [ "$?" -eq "0" ]
then
  echo "  -o downloaded"
else
  echo "  -x download FAILED!!"
  exit 1
fi

echo "-> setting configuration to use your apikey"
echo "{\"APIKEY\":\"$APIKEY\",\"LKID\":\"100\",\"VERSION\":\"0.7\",\"FLUSH\":\"200\"}" > config.json
chmod +x /usr/local/bin/apiban/apiban-iptables-client
echo "-> setting log rotation"
cat > /etc/logrotate.d/apiban-client << EOF
/var/log/apiban-client.log {
        daily
        copytruncate
        rotate 7
        compress
}
EOF
echo "-> setting up service"
cd /lib/systemd/system/
wget https://raw.githubusercontent.com/palner/apiban/master/clients/go/systemd/apiban-iptables.service
if [ "$?" -eq "0" ]
then
  echo "  -o downloaded"
else
  echo "  -x download FAILED!!"
  exit 1
fi
wget https://raw.githubusercontent.com/palner/apiban/master/clients/go/systemd/apiban-iptables.timer
systemctl enable apiban-iptables.service
if [ "$?" -eq "0" ]
then
  echo "  -o downloaded"
else
  echo "  -x download FAILED!!"
  exit 1
fi
systemctl enable apiban-iptables.timer
systemctl start apiban-iptables.timer
systemctl start apiban-iptables.service
echo "-> all done."
