# APIBAN #

REST API for identifying IP addresses sending unwanted SIP traffic

**APIBAN** helps prevent unwanted SIP traffic by identifying addresses of known bad actors before they attack your system. Bad actors are collected through globally deployed honeypots and curated by [LOD](https://www.lod.com/).

Visit <https://www.apiban.org/> for more information.

## Block/Identify Traffic ##

This API allows you to integrate and interact with **APIBAN** data.

The data is provided in standard JSON responses and uses HTTP Status Codes to help determine results.

**NOTE:** If you are looking to protect your PBX or SIP server without programming, you should use the [**APIBAN** client](https://github.com/palner/apiban/tree/master/clients/go) to automatically block traffic.

## Using The API ##

1. [obtain an API KEY](https://apiban.org/getkey.html) (the API KEY is used for all API requests)
2. Integrate
    * To protect your PBX automatically, without programming, use our [go api client](https://github.com/palner/apiban/tree/master/clients/go) to integrate with iptables.
    * To integrate directly with [Kamailio](https://www.kamailio.org), see [Integration with Kamailio](#integration-into-kamailio).
    * To integrate with [HOMER](http://sipcapture.org/), see [Integration with HOMER](#integration-with-homer).
    * To integrate with SIP3, see [Integration with SIP3](#integration-with-sip3).
    * To integrate with IPTABLES, see [Integration with IPTABLES](#integration-with-iptables).
    * To integrate with [OpenSIPS](https://opensips.org), see [Integration with OpenSIPS](#integration-with-opensips).

Once you have an API KEY, you can use the API to:

* pull the full banned list
* check specific IPs
* pull changes since last full list

The full **APIBAN** API documentation is available at <https://apiban.org/doc.html>.

## Integration into Kamailio ##

**NOTE:** If you are looking to protect your PBX or SIP server without programming, you should use the [**APIBAN** client](https://github.com/palner/apiban/tree/master/clients/go) to automatically block traffic.

[Kamailio](https://github.com/kamailio/kamailio) is an open source implementation of a SIP Signaling Server. SIP is an open standard protocol specified by the IETF. The core specification document is RFC3261.

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

**Bonus:** Want to run **APIBAN** at start-up? Using the *htable:mod-init* event_route built into Kamailio, you can pre-load the **APIBAN** htable at start-up:

```
event_route[htable:mod-init] {
	# pre load apiban
	route(APIBAN);
}
```

## Integration with HOMER ##

[HOMER](https://github.com/sipcapture/homer) is a robust, carrier-grade, scalable Packet and Event capture system and VoiP/RTC Monitoring Application based on the HEP/EEP protocol and ready to process & store insane amounts of signaling, rtc events, logs and statistics with instant search, end-to-end analysis and drill-down capabilities.

Homer implements [APIBan](https://github.com/sipcapture/hepsub-apiban) interactions through a dedicated [HEPSub agent](https://github.com/sipcapture/hepsub-apiban) interactively retrieving and caching APIBan API information in memory, and providing total flexibility, extensibility and customization for HEP users and integrators.

Read more: <https://github.com/sipcapture/hepsub-apiban>

## Integration with SIP3 ##

[SIP3](https://sip3.io/) is an end-to-end solution for real-time monitor, analysis and troubleshooting of network performance in large volumes of traffic.

*Thanks to the SIP3 architecture design you can have a monitoring set in place that works in front of iptables. So even if the traffic has been blocked you will still be able detect fraud attempts and whitelist wrongly blocked IP addresses.*

Read more: <https://sip3.io/docs/tutorials/HowToInroduceUserDefinedAttribute.html>

## Integration with IPTABLES ##

**APIBAN** provides two open source [clients](https://github.com/palner/apiban/tree/master/clients) for integrated into IPTABLES.

## Integration with OpenSIPS ##

You will need to load the following modules (if not already loaded):

```
loadmodule "json.so"
loadmodule "cachedb_local.so"
loadmodule "rest_client.so"
```

The following collections and URLs should be created (adjust the size of collection `apiban` as needed):

```
modparam("cachedb_local", "cache_collections", "apiban = 11;apibanctl = 1")
modparam("cachedb_local", "cachedb_url", "local:apiban:///apiban")
modparam("cachedb_local", "cachedb_url", "local:apibanctl:///apibanctl")
```

We'll adjust some rest client timeouts:

```
modparam("rest_client", "curl_timeout", 60)
modparam("rest_client", "connection_timeout", 10)
```

Let's create the `[APIBAN]` route at the end of your script. When invoked, it will get the batches of IP addresses and add them to the cache. Note that APIBAN sends batches of 250 entries per batch.
The control ID is used to download an incremental list. On startup or restart, the full list is loaded.

```
route[APIBAN] {
  $var(apikey) = "<<<< ENTER YOU APIBAN KEY HERE >>>>";

  $var(loop) = true;

  while ($var(loop)) {
    if (!cache_fetch("local:apibanctl", "ID", $var(apiban_id)))
      $var(apiban_id) = 0;

    if($var(apiban_id) == "0") {
      # First run, we will get the full IP list
      $var(apiget) = "https://apiban.org/api/" + $var(apikey) + "/banned";
    } else {
      # ID exists, we pull incemental list
      $var(apiget) = "https://apiban.org/api/" + $var(apikey) + "/banned/" + $var(apiban_id);
    }

    xlog("L_INFO","APIBAN: Sending API request to APIBAN\n");
    rest_get("$var(apiget)",$var(resp_body),,$var(rcode));

    if ($var(rcode) == 200) {
      $json(apiban_resp) := $var(resp_body);
      $json(ip_array) := $(json(apiban_resp/ipaddress));
      cache_store("local:apibanctl", "ID", "$(json(apiban_resp/ID))");

      $var(index_count) = 0;
      for ($var(ip) in $(json(ip_array)[*])) {
        cache_store("local:apiban", "$var(ip)", "1");
        xlog("L_INFO","APIBAN: Adding IP address $var(ip)\n");
        $var(index_count) = $var(index_count) + 1;
      }
      if ($var(index_count) < 250) {
        # Last batch has less than 250 records, loop can stop
        break;
      } 
    } else {
      # There are no records to process in ths request
      break;
    }
  }
}
```

Let's create a timer-route in order to get updates from APIBAN. Change the interval of 180 seconds to whatever works best for you. Remember that APIBAN has a limit of 11 requests every 2 minutes, adjust accordingly if you are using the same key for several instances.
```
timer_route[apiban_update, 180] {
  route(APIBAN);
}
```

[Optional]: Let's create a startup-route (or modify your current one) in order to get the data from APIBAN when OpenSIPS is starting up.
```
startup_route {
  route(APIBAN);
}
```

Now you  are able to pull the list of IP banned addresses from APIBAN, now let's make good use of it. Somewhere in the top of your main route, add:

```
if (cache_fetch("local:apiban", "$si", $var(ip_val))) {
  xlog("Request from IP $si blocked. IP is listed in APIBAN\n");
  exit;
}
```

## Getting Help ##

Help is provided by LOD (<https://www.lod.com>) and an APIBAN room ([#apiban:matrix.lod.com](https://matrix.to/#/#apiban:matrix.lod.com)) is available on the LOD Matrix homeserver. The software is provided under the GPLv2 license.
