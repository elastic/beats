Log to track "changes" from logstash-forward to filebeat

Questions:

 * File size limit for config file was removed. In case this is required, it must be handled by libbeat
 * Config file: Type is required and moved out of fields. Fields can optional for additional information
 * Config: input field was introduced with option "stdin" or "log". Default is "log". Idea is that in the future
   also full files could be read (fsriver) (InputBeat). Fifoin or streams could be also added. (see also https://github.com/elastic/logstash-forwarder/issues/525)
 * Go trough error messages and check if the texts are good

Notes:
* Should every config entry have a name -> make it possible to know from which config entry something comes.
  Can it be that config files overlap -> double indexing of a file?
* All command line options must be available as config options for the beats
* On debug we should print out all config options on startup -> any good idea how to do this recursively?
* Add type options to config file per harvester

     # * file: Sends the full file on change. This replaces fsRiver
     # * meta: Reads new meta information on file change

Next with priority
* Multi line support
* Filtering support
** Filter option for every prospector:

      # Regexp log line filter (not implemented yet)
      filter: "regexp"


