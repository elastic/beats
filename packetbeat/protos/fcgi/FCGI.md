
FCGI Spec


FastCGI records. 

    size: 8 - 65543 Bytes

    typedef struct {
        unsigned char version;
        unsigned char type;
        unsigned char requestIdB1;
        unsigned char requestIdB0;
        unsigned char contentLengthB1;
        unsigned char contentLengthB0;
        unsigned char paddingLength;
        unsigned char reserved;
        unsigned char contentData[contentLength];
        unsigned char paddingData[paddingLength];
    } FCGI_Record;


* version: Identifies the FastCGI protocol version. This specification documents FCGI_VERSION_1.
* type: Identifies the FastCGI record type, i.e. the general function that the record performs. 

** App type records:
*** FCGI_BEGIN_REQUEST       1      app
*** FCGI_ABORT_REQUEST       2      app
*** FCGI_END_REQUEST         3      
*** FCGI_PARAMS              4      app             stream
*** FCGI_STDIN               5      app             stream
*** FCGI_STDOUT              6                      stream 
*** FCGI_STDERR              7                      stream
*** FCGI_DATA                8      app             stream

** Mgmnt type records:
*** FCGI_GET_VALUES          9      app     mgmnt 
*** FCGI_GET_VALUES_RESULT  10              mgmnt
*** FCGI_UNKNOWN_TYPE       11              mgmnt

* requestId: Identifies the FastCGI request to which the record belongs.
* contentLength: The number of bytes in the contentData component of the record.
* paddingLength: The number of bytes in the paddingData component of the record.
* contentData: Between 0 and 65535 bytes of data, interpreted according to the record type.
* paddingData: Between 0 and 255 bytes of data, which are ignored.

Consistency_tip: if ( mgmnt record ) requestIdB1 n requestIdB0 == 0 


=== App type records ===

FCGI_BEGIN_REQUEST       1

    size: 8 Bytes

    typedef struct {
        unsigned char roleB1;
        unsigned char roleB0;
        unsigned char flags;
        unsigned char reserved[5];
    } FCGI_BeginRequestBody;


* roles: 
FCGI_RESPONDER  1
FCGI_AUTHORIZER 2
FCGI_FILTER     3
* flags: closes the connection if (flags & FCGI_KEEP_CONN (byte) 1 ) == 0
* reserved: ignored



FCGI_END_REQUEST        3
    
    size: 8 bytes

        typedef struct {
            unsigned char appStatusB3;
            unsigned char appStatusB2;
            unsigned char appStatusB1;
            unsigned char appStatusB0;
            unsigned char protocolStatus;
            unsigned char reserved[3];
        } FCGI_EndRequestBody;




Name-Value Pairs 

        typedef struct {
            unsigned char nameLengthB0;  /* nameLengthB0  >> 7 == 0 */
            unsigned char valueLengthB0; /* valueLengthB0 >> 7 == 0 */
            unsigned char nameData[nameLength];
            unsigned char valueData[valueLength];
        } FCGI_NameValuePair11;

        typedef struct {
            unsigned char nameLengthB0;  /* nameLengthB0  >> 7 == 0 */
            unsigned char valueLengthB3; /* valueLengthB3 >> 7 == 1 */
            unsigned char valueLengthB2;
            unsigned char valueLengthB1;
            unsigned char valueLengthB0;
            unsigned char nameData[nameLength];
            unsigned char valueData[valueLength
                    ((B3 & 0x7f) << 24) + (B2 << 16) + (B1 << 8) + B0];
        } FCGI_NameValuePair14;

        typedef struct {
            unsigned char nameLengthB3;  /* nameLengthB3  >> 7 == 1 */
            unsigned char nameLengthB2;
            unsigned char nameLengthB1;
            unsigned char nameLengthB0;
            unsigned char valueLengthB0; /* valueLengthB0 >> 7 == 0 */
            unsigned char nameData[nameLength
                    ((B3 & 0x7f) << 24) + (B2 << 16) + (B1 << 8) + B0];
            unsigned char valueData[valueLength];
        } FCGI_NameValuePair41;

        typedef struct {
            unsigned char nameLengthB3;  /* nameLengthB3  >> 7 == 1 */
            unsigned char nameLengthB2;
            unsigned char nameLengthB1;
            unsigned char nameLengthB0;
            unsigned char valueLengthB3; /* valueLengthB3 >> 7 == 1 */
            unsigned char valueLengthB2;
            unsigned char valueLengthB1;
            unsigned char valueLengthB0;
            unsigned char nameData[nameLength
                    ((B3 & 0x7f) << 24) + (B2 << 16) + (B1 << 8) + B0];
            unsigned char valueData[valueLength
                    ((B3 & 0x7f) << 24) + (B2 << 16) + (B1 << 8) + B0];
        } FCGI_NameValuePair44;
