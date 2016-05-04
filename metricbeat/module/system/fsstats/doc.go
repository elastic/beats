/*
Package fsstats provides a MetricSet for fetching aggregated filesystem stats.

An example event looks as following:

	{
	  "@timestamp": "2016-05-03T15:11:04.610Z",
	  "beat": {
	    "hostname": "ruflin",
	    "name": "ruflin"
	  },
	  "metricset": "fsstats",
	  "module": "system",
	  "rtt": 84,
	  "system-fsstats": {
	    "count": 4,
	    "total_files": 60982450,
	    "total_size": {
	      "free": 32586960896,
	      "total": 249779548160,
	      "used": 217192587264
	    }
	  },
	  "type": "metricsets"
	}
*/
package fsstats
