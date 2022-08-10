# list of fields we want to keep
keep = ["ecs", "cloud", "docker", "kubernetes", "kibana", "gcp", "google_workspace"] # log?, misp? osquery? elasticsearch? process? host?

# opening rendered fields
with open("../filebeat/build/fields/fields.all.yml") as file:
  data = file.readlines()
  data = [line.rstrip() for line in data]

fields = []
output = []

# splitting fields per type
for line in data:
  if line.startswith("-"):
    fields.append([line])
  elif len(fields) > 0:
    fields[-1].append(line)

# removing field type we don't need
for field in fields:
  key = field[0].split(" ")
  if key[-1] in keep:
    output += field

with open('fields.yml', 'w') as f:
  for item in output:
    f.write("%s\n" % item)
