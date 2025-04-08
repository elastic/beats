---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/bandwidth-throttling.html
---

# Auditbeat uses too much bandwidth [bandwidth-throttling]

If you need to limit bandwidth usage, we recommend that you configure the network stack on your OS to perform bandwidth throttling.

For example, the following Linux commands cap the connection between Auditbeat and Logstash by setting a limit of 50 kbps on TCP connections over port 5044:

```shell
tc qdisc add dev $DEV root handle 1: htb
tc class add dev $DEV parent 1:1 classid 1:10 htb rate 50kbps ceil 50kbps
tc filter add dev $DEV parent 1:0 prio 1 protocol ip handle 10 fw flowid 1:10
iptables -A OUTPUT -t mangle -p tcp --dport 5044 -j MARK --set-mark 10
```

Using OS tools to perform bandwidth throttling gives you better control over policies. For example, you can use OS tools to cap bandwidth during the day, but not at night. Or you can leave the bandwidth uncapped, but assign a low priority to the traffic.

