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

In version 2, encryption is added to version 1.  The
segments are made of a header followed by an initialization vector,
and then encrypted frames.  The header consists of one field, the
version number which is an unsigned 32-bit integer in little-endian
format.  The initialization vector is 128-bits in length.  The count
was dropped from version 1 for 2 reasons.  The first, if it was
outside the encrypted portion of the segment then it would be easy for
an attacker to modify.  The second, is that adding it to the encrypted
segment in a meaningful way was problematic.  The count is not known
until the last frame is written.  With encryption you cannot seek to
the beginning of the segment and update the value.  Adding the count
to the end is less useful because you have to decrypt the entire
segment before it can be read.

![Segment Schema Version 2](./schemaV2.svg)

The frames for version 2, consist of a header, followed by the
serialized event and a footer.  The header contains one field which is
the size of the frame, which is an unsigned 32-bit integer in
little-endian format.  The serialization format is CBOR.  The footer
contains 2 fields, the first of which is a checksum which is an
unsigned 32-bit integer in little-endian format, followed by a repeat
of the size from the header.  This is the same as version 1.

![Frame Version 2](./frameV2.svg)

## Version 3

In version 2, compression is added to version 2.  The
segments are made of a header followed by an initialization vector,
and then encrypted frames.  The header consists of one field, the
version number which is an unsigned 32-bit integer in little-endian
format.  The initialization vector is 128-bits in length.

![Segment Schema Version 3](./schemaV3.svg)

The frames for version 2, consist of a header, followed by the
compressed serialized event and a footer.  The header contains one
field which is the size of the frame, which is an unsigned 32-bit
integer in little-endian format.  The compression is LZ4 with fast
compression.  The serialization format is CBOR.  The footer contains 2
fields, the first of which is a checksum which is an unsigned 32-bit
integer in little-endian format, followed by a repeat of the size from
the header.

![Frame Version 3](./frameV3.svg)
