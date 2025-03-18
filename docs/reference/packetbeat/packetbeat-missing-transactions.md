---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/packetbeat-missing-transactions.html
---

# Packetbeat is missing long running transactions [packetbeat-missing-transactions]

Packetbeat has an internal timeout that it uses to time out transactions and TCP connections when no packets have been seen for a long time.

To process long running transactions, you can specify a larger value for the [`transaction_timeout`](/reference/packetbeat/common-protocol-options.md#transaction-timeout-option) option. However, keep in mind that very large timeout values can increase memory usage if messages are lost or transaction response messages are not sent.

