# Disk Queue On Disk Structures

The disk queue is a directory on disk that contains files.  Each
file is called a segment.  The name of the file is the segment id in
base 10 with the ".seg" suffix.  For example: "42.seg".  Each segment
contains multiple frames.  Each frame contains one event.

There are currently 3 versions of the disk queue, and the current code
base is able to write versions 1 & 2, while it is able to read version
0, 1, and 2.

## Version 0

In version 0, the segments are made up of a header, followed by
frames.  The header contains one field which is an unsigned 32-bit
integer in little-endian byte order, which signifies the version number.

![Segment Schema Version 0](./schemaV0.svg)

The frames for version 0, consist of a header, followed by the
serialized event and a footer.  The header contains one field which is
the size of the frame, which is an unsigned 32-bit integer in
little-endian byte order.  The serialization format is JSON.  The
footer contains 2 fields, the first of which is a checksum which is an
unsigned 32-bit integer in little-endian format, followed by a repeat
of the size from the header.

![Frame Version 0](./frameV0.svg)

## Version 1

In version 1, the segments are made up of a header, followed by
frames.  The header contains two fields.  The first field in the
version number, which is an unsigned 32-bit integer in little-endian
format.  The second field is a count of the number of frames in the
segment, which is an unsigned 32-bit integer in little-endian format.

![Segment Schema Version 1](./schemaV1.svg)

The frames for version 1, consist of a header, followed by the
serialized event and a footer.  The header contains one field which is
the size of the frame, which is an unsigned 32-bit integer in
little-endian format.  The serialization format is CBOR.  The footer
contains 2 fields, the first of which is a checksum which is an
unsigned 32-bit integer in little-endian format, followed by a repeat
of the size from the header.

![Frame Version 1](./frameV1.svg)

## Version 2

In version 2, the segments are made of a header followed by an
optional initialization vector, and then frames.  The header consists
of three fields.  The first field in the version number, which is an
unsigned 32-bit integer in little-endian format.  The second field is
a count of the number of frames in the segment, which is an unsigned
32-bit integer in little-endian format.  The third field holds bit
flags, which signify options.  The size of options is 32-bits in
little-endian format.

If no fields are set in the options field, then un-encrypted frames
follow the header.

If the options field has the first bit set, then encryption is
enabled.  In which case, the next 128-bits are the initialization
vector and the rest of the file is encrypted frames.

If the options field has the second bit set, then compression is
enabled.  In which case, LZ4 compressed frames follow the header.

If both the first and second bit of the options field are set, then
both compression and encryption are enabled.  The next 128-bits are
the initialization vector and the rest of the file is LZ4 compressed
frames.

If the options field has the third bit set, then Google Protobuf is
used to serialize the data in the frame instead of CBOR.

![Segment Schema Version 2](./schemaV2.svg)

The frames for version 2, consist of a header, followed by the
serialized event and a footer.  The header contains one field which is
the size of the frame, which is an unsigned 32-bit integer in
little-endian format.  The serialization format is CBOR or Google
Protobuf.  The footer contains 2 fields, the first of which is a
checksum which is an unsigned 32-bit integer in little-endian format,
followed by a repeat of the size from the header.  The only difference
from Version 1 is the option for the serialization format to be CBOR
or Google Protobuf.

![Frame Version 2](./frameV2.svg)
