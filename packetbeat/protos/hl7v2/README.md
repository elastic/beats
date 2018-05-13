### Save the full HL7 request

send_request: true


### Save the full HL7 response

send_response: true


### Set the segment newline char/s if different to the standard \r

newline_chars: \r


### Set the segment selection mode, Include (only the segments specified will be matched) or Exclude (everything except the segments specified will be matched)

segment_selection_mode: Include


### Set the field selection mode, Include (only the fields specified will be matched) or Exclude (everything except the fields specified will be matched)

field_selection_mode: Include


### Set the component selection mode, Include (only the components specified will be matched) or Exclude (everything except the components specified will be matched)

component_selection_mode: Include


### Segments to include or exclude

segments: [MSH,MSA]


### Fields to include or exclude

fields: [MSH-3,MSH-4,MSH-5,MSH-6,MSH-9,MSH-10,MSA-1,MSA-2]


### Map field/component names to custom names

name_mapping:
  `-` MSH-3: MSH-3 Sending Application
  `-` MSH-4: MSH-4 Sending Facility
  `-` MSH-5: MSH-5 Receiving Application
  `-` MSH-6: MSH-6 Receiving Facility
  `-` MSH-9: MSH-9 Message Type
  `-` MSH-9-1: MSH-9.1 Message Type
  `-` MSH-9-2: MSH-9.2 Event Type
  `-` MSH-10: MSH-10 Message Control ID
  `-` MSA-1: MSA-1 Acknowledgement Code
  `-` MSA-2: MSA-2 Message Control ID

  