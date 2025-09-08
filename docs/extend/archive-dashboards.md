---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/archive-dashboards.html
---

# Archiving Your Beat Dashboards [archive-dashboards]

The Kibana dashboards for the Elastic Beats are saved under the `kibana` directory. To create a zip archive with the dashboards, including visualizations and searches and the index pattern, you can run the following command in the Beat repository:

```shell
make package-dashboards
```

The Makefile is part of libbeat, which means that community Beats contributors can use the commands shown here to archive dashboards. The dashboards must be available under the `kibana` directory.

Another option would be to create a repository only with the dashboards, and use the GitHub release functionality to create a zip archive.

Share the Kibana dashboards archive with the community, so other users can use your cool Kibana visualizations!

