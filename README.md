# apiban #

REST API for sharing IP addresses sending unwanted SIP traffic

Visit <https://www.apiban.org/> for more information.

## Integration into Kamailio ##

### Blocking Banned IPs ###

A loop is used to cycle through the banned IPs. On first download, this list can be quite large and `max_while_loops` will need to be large enough to handle the list.

```
max_while_loops=250
```

You will need to load the following modules (if not already loaded):

```
loadmodule "http_client.so"
loadmodule "jansson.so"
loadmodule "rtimer.so"
```

The following htables should be created (you can increase the size of `apiban` as needed):

```
modparam("htable", "htable", "apiban=>size=11;")
modparam("htable", "htable", "apibanctl=>size=1;initval=0;")
```

In this example, let's set an rtimer to run every 5 minutes:

```
modparam("rtimer", "timer", "name=apiban;interval=300;mode=1;")
modparam("rtimer", "exec", "timer=apiban;route=APIBAN")
```

Let's create that `[APIBAN]` route to get the ipaddresses and add them to the apiban htable. The control ID is used to download an incremental list. On startup or restart, the full list is loaded.

```
route[APIBAN] {
	// check if we already have an APIBAN id... if so, get the updates and
	// if not, get the full list of banned ips.

	// replace MYAPIKEY with your apiban.org API key.
	$var(apikey) = "MYAPIKEY";

	if($sht(apibanctl=>ID) == 0) {
		$var(apiget) = "https://apiban.org/api/" + $var(apikey) + "/banned";
	} else {
		$var(apiget) = "https://apiban.org/api/" + $var(apikey) + "/banned/" + $sht(apibanctl=>ID);
	}

	xlog("L_INFO","APIBAN: Sending API request to $var(apiget)\n");
	http_client_query("$var(apiget)", "$var(banned)");

	// if we dont get a 200 OK from the webserver we will log and exit
	if($rc!=200) {
		xlog("L_INFO","APIBAN: Non 200 response. $var(banned)\n");
		xlog("L_INFO","APIBAN: $sht(apibanctl=>blocks) attacks blocked since $(Tb{s.ftime,%Y-%m-%d %H:%M:%S})\n");
		exit;
	}

	// lets loop through the ipaddresses we received from our API request
	$var(count) = 0;
	jansson_array_size("ipaddress", $var(banned), "$var(size)");
	while($var(count) < $var(size)) {
		jansson_get("ipaddress[$var(count)]", $var(banned), "$var(blockaddr)");
		// add the blocked ipaddress to the apiban htable and log
		$sht(apiban=>$var(blockaddr)) = 1;
		xlog("L_INFO","APIBAN: Adding block ipaddress[$var(count)] == $var(blockaddr)\n");

		$var(count) = $var(count) + 1;
	}

	// lets get our control ID and use it for incremental downloads
	jansson_get("ID", $var(banned), "$var(apiid)");
	xlog("L_INFO","APIBAN: New ID is $var(apiid)\n");
	$sht(apibanctl=>ID) = $var(apiid);
}
```

Lastly, we can use these IPs to block unwanted traffic. For example, if you were using ipban as demonstrated in the `[REQINIT]` route of the default config, you can just add this block:

```
		if($sht(apiban=>$si)!=$null) {
			// ip is blocked from apiban.org
			xdbg("request from apiban.org blocked IP - $rm from $fu (IP:$si:$sp)\n");
			$sht(apibanctl=>blocks) = $sht(apibanctl=>blocks) + 1;
			exit;
		}
```

**Bonus:** Want to run APIBAN at start-up? Using the *htable:mod-init* event_route built into Kamailio, you can pre-load the APIBAN htable at start-up:

```
event_route[htable:mod-init] {
	# pre load apiban
	route(APIBAN);
}
```
## Integration with HOMER ##

[HOMER](https://github.com/sipcapture/homer) implements [APIBan](https://github.com/sipcapture/hepsub-apiban) interactions through a dedicated [HEPSub agent](https://github.com/sipcapture/hepsub-apiban) interactively retrieving and caching APIBan API information in memory, and providing total flexibility, extensibility and customization for HEP users and integrators.

## Integration with SIP3 ##

[SIP3](https://sip3.io/) is an end-to-end solution for real-time monitor, analysis and troubleshooting of network performance in large volumes of traffic.

*Thanks to the SIP3 architecture design you can have a monitoring set in place that works in front of iptables. So even if the traffic has been blocked you will still be able detect fraud attempts and whitelist wrongly blocked IP addresses.*

Read more: <https://sip3.io/docs/tutorials/HowToInroduceUserDefinedAttribute.html>

## Getting Help ##

Help is provided by LOD (<https://www.lod.com>) and an APIBAN room ([#apiban:matrix.lod.com](https://matrix.to/#/#apiban:matrix.lod.com)) is available on the LOD Matrix homeserver. The software is provided under the GPLv2 license.
