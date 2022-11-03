## Steps

1. Get an EC2 instance running Amazon Linux
2. Install the necessary packages

   ```
   sudo yum update -y
   sudo yum install -y git go jq htop tmux
   # Optionally install an text editor like Emacs
   sudo yum install -y emacs-nox
   ```

3. Download Elasticsearch and Kibana

   ```
   wget https://artifacts.elastic.co/downloads/elasticsearch/elasticsearch-8.4.3-linux-x86_64.tar.gz
   wget https://artifacts.elastic.co/downloads/kibana/kibana-8.4.3-linux-x86_64.tar.gz
   tar -xf elasticsearch-8.4.3-linux-x86_64.tar.gz
   tar -xf kibana-8.4.3-linux-x86_64.tar.gz
   ```

4. Start Elasticsearch and Kibana
Start Elasticsearch, save the credentials that will be printed out,
then start Kibana and configure it using the enrollment token. To
access Kibana use port fowarding.

5. Port fowarding to access Kibana
   ```
   ssh -L 5601:127.0.0.1:5601 ec2-user@42.42.42.42 -i ~/.ssh/my-keypair.pem
   ```
6. Clone Beats and go to the folder with the script
   ```
   git clone https://github.com/elastic/beats.git
   cd beats/dev-tools/performance/filebeat/
   ```
7. Export environmet variables to configure the script:
    * `ES_USER` - Elasticsearch user
    * `ES_PASS` - Elasticsearch password
    * `KIBANA_HOST` - Kibana host
    * `JSON_LOGS` - `true` or `false`, if `true` JSON logs will be used
    * `VERIFICATION_MODE` set it to `none` if using self-signed
      certificates (like on a local deployment of ES/Kibana)

8. Run the script

   ```
   ./run.sh
   ```

   The script will do everything needed:
   1. Download/Install `flog` to generate the log files
   2. Generate the log files using a hardcoded seed (so they're
      reproducible)
   3. Setup Stack Monitoring on ES
   4. Extract the Cluster UUID
   5. Start a monitoring Metricbeat (and load the dashboards)
   6. Run Filebeat logging to a file and to stdout (using `tee`)
   7. Once the data has been ingested, hit `CTRL+C` **ONCE** and wait
      for Filebeat to exit, and the script to kill Metricbeat

    If running the tests again, make sure to delete the data stream
    created as well as the index template.

## Debug information
aka ignore it if you're just reproducing the experiment, the script
will do eveyrhting for you)

Monitoring needs to be enabled in the cluster:

PUT _cluster/settings
{
  "persistent": {
    "xpack.monitoring.collection.enabled": true
  }
}


Get cluster UUID

GET /
