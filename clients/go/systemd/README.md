# systemd #

## Installation ##

Installation of the systemd service

⚠️ CAUTION: Please ensure that you understand the actions of the following commands before executing them ⚠️

### Quick and Easy Install Instructions ###

1. Download apiban-iptables.service and apiban-iptables.timer to `/lib/systemd/system/`
    * `cd /lib/systemd/system/`
    * `wget https://raw.githubusercontent.com/palner/apiban/master/clients/go/systemd/apiban-iptables.service`
    * `wget https://raw.githubusercontent.com/palner/apiban/master/clients/go/systemd/apiban-iptables.timer`
2. Enable Service and Timer
    * `systemctl enable apiban-iptables.service`
    * `systemctl enable apiban-iptables.timer`
3. Test (Should indicate "enabled")
    * `systemctl list-unit-files apiban-iptables.service`
    * `systemctl list-unit-files apiban-iptables.timer`
4. Start Timer
    * `systemctl start apiban-iptables.timer`
5. Test
    * `systemctl list-timers apiban-iptables.timer`

### Logs ###

* For service
    * `journalctl -u apiban-iptables`
    
* For timer
    * `journalctl -u apiban-iptables.timer`
