---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/error-found-unexpected-character.html
---

# Found unexpected or unknown characters [error-found-unexpected-character]

Either there is a problem with the structure of your config file, or you have used a path or expression that the YAML parser cannot resolve because the config file contains characters that aren’t properly escaped.

If the YAML file contains paths with spaces or unusual characters, wrap the paths in single quotation marks (see [Wrap paths in single quotation marks](/reference/heartbeat/yaml-tips.md#wrap-paths-in-quotes)).

Also see the general advice under [*Avoid YAML formatting problems*](/reference/heartbeat/yaml-tips.md).

