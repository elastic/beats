The script directory contains various scripts which exist to help testing aws module in metricbeat.
Below is a brief description of each file / folder.


| File / Folder              | Description                                                                                                      |
|----------------------------|------------------------------------------------------------------------------------------------------------------|
| get_temp_creds.go          | Use MFA token to get temporary AWS credentials.                                                                  |
| sqs_send_receive_delete.go | Get a list of SQS queues for a given region, send messages, receive messages and delete message from each queue. |
| test_cloudwatch_metricset.yml| Config file for testing using cloudwatch metricset to collect EC2 metrics and tags. |
