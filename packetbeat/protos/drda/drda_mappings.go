package drda

//http://svn.apache.org/repos/asf/db/derby/code/trunk/java/client/org/apache/derby/client/net/CodePoint.java
const DRDA_MAGIC = 0xD0

const DRDA_CP_DATA = 0x0000
const DRDA_CP_CODPNT = 0x000C
const DRDA_CP_FDODSC = 0x0010
const DRDA_CP_TYPDEFNAM = 0x002F
const DRDA_CP_TYPDEFOVR = 0x0035
const DRDA_CP_CODPNTDR = 0x0064
const DRDA_CP_EXCSAT = 0x1041
const DRDA_CP_SYNCCTL = 0x1055
const DRDA_CP_SYNCRSY = 0x1069
const DRDA_CP_ACCSEC = 0x106D
const DRDA_CP_SECCHK = 0x106E
const DRDA_CP_SYNCLOG = 0x106F
const DRDA_CP_RSCTYP = 0x111F
const DRDA_CP_RSNCOD = 0x1127
const DRDA_CP_RSCNAM = 0x112D
const DRDA_CP_PRDID = 0x112E
const DRDA_CP_PRCCNVCD = 0x113F
const DRDA_CP_VRSNAM = 0x1144
const DRDA_CP_SRVCLSNM = 0x1147
const DRDA_CP_SVRCOD = 0x1149
const DRDA_CP_SYNERRCD = 0x114A
const DRDA_CP_SRVDGN = 0x1153
const DRDA_CP_SRVRLSLV = 0x115A
const DRDA_CP_SPVNAM = 0x115D
const DRDA_CP_EXTNAM = 0x115E
const DRDA_CP_SRVNAM = 0x116D
const DRDA_CP_SECMGRNM = 0x1196
const DRDA_CP_DEPERRCD = 0x119B
const DRDA_CP_CCSIDSBC = 0x119C
const DRDA_CP_CCSIDDBC = 0x119D
const DRDA_CP_CCSIDMBC = 0x119E
const DRDA_CP_USRID = 0x11A0
const DRDA_CP_PASSWORD = 0x11A1
const DRDA_CP_SECMEC = 0x11A2
const DRDA_CP_SECCHKCD = 0x11A4
const DRDA_CP_SVCERRNO = 0x11B4
const DRDA_CP_SECTKN = 0x11DC
const DRDA_CP_NEWPASSWORD = 0x11DE
const DRDA_CP_MGRLVLRM = 0x1210
const DRDA_CP_MGRDEPRM = 0x1218
const DRDA_CP_SECCHKRM = 0x1219
const DRDA_CP_CMDATHRM = 0x121C
const DRDA_CP_AGNPRMRM = 0x1232
const DRDA_CP_RSCLMTRM = 0x1233
const DRDA_CP_PRCCNVRM = 0x1245
const DRDA_CP_CMDCMPRM = 0x124B
const DRDA_CP_SYNTAXRM = 0x124C
const DRDA_CP_CMDNSPRM = 0x1250
const DRDA_CP_PRMNSPRM = 0x1251
const DRDA_CP_VALNSPRM = 0x1252
const DRDA_CP_OBJNSPRM = 0x1253
const DRDA_CP_CMDCHKRM = 0x1254
const DRDA_CP_TRGNSPRM = 0x125F
const DRDA_CP_AGENT = 0x1403
const DRDA_CP_MGRLVLLS = 0x1404
const DRDA_CP_SUPERVISOR = 0x143C
const DRDA_CP_SECMGR = 0x1440
const DRDA_CP_EXCSATRD = 0x1443
const DRDA_CP_CMNAPPC = 0x1444
const DRDA_CP_DICTIONARY = 0x1458
const DRDA_CP_MGRLVLN = 0x1473
const DRDA_CP_CMNTCPIP = 0x1474
const DRDA_CP_FDODTA = 0x147A
const DRDA_CP_CMNSYNCPT = 0x147C
const DRDA_CP_ACCSECRD = 0x14AC
const DRDA_CP_SYNCPTMGR = 0x14C0
const DRDA_CP_RSYNCMGR = 0x14C1
const DRDA_CP_CCSIDMGR = 0x14CC
const DRDA_CP_MONITOR = 0x1900
const DRDA_CP_MONITORRD = 0x1C00
const DRDA_CP_XAMGR = 0x1C01
const DRDA_CP_ACCRDB = 0x2001
const DRDA_CP_BGNBND = 0x2002
const DRDA_CP_BNDSQLSTT = 0x2004
const DRDA_CP_CLSQRY = 0x2005
const DRDA_CP_CNTQRY = 0x2006
const DRDA_CP_DRPPKG = 0x2007
const DRDA_CP_DSCSQLSTT = 0x2008
const DRDA_CP_ENDBND = 0x2009
const DRDA_CP_EXCSQLIMM = 0x200A
const DRDA_CP_EXCSQLSTT = 0x200B
const DRDA_CP_OPNQRY = 0x200C
const DRDA_CP_PRPSQLSTT = 0x200D
const DRDA_CP_RDBCMM = 0x200E
const DRDA_CP_RDBRLLBCK = 0x200F
const DRDA_CP_REBIND = 0x2010
const DRDA_CP_DSCRDBTBL = 0x2012
const DRDA_CP_EXCSQLSET = 0x2014
const DRDA_CP_DSCERRCD = 0x2101
const DRDA_CP_QRYPRCTYP = 0x2102
const DRDA_CP_RDBINTTKN = 0x2103
const DRDA_CP_PRDDTA = 0x2104
const DRDA_CP_RDBCMTOK = 0x2105
const DRDA_CP_RDBCOLID = 0x2108
const DRDA_CP_PKGID = 0x2109
const DRDA_CP_PKGCNSTKN = 0x210D
const DRDA_CP_RTNSETSTT = 0x210E
const DRDA_CP_RDBACCCL = 0x210F
const DRDA_CP_RDBNAM = 0x2110
const DRDA_CP_OUTEXP = 0x2111
const DRDA_CP_PKGNAMCT = 0x2112
const DRDA_CP_PKGNAMCSN = 0x2113
const DRDA_CP_QRYBLKSZ = 0x2114
const DRDA_CP_UOWDSP = 0x2115
const DRDA_CP_RTNSQLDA = 0x2116
const DRDA_CP_RDBALWUPD = 0x211A
const DRDA_CP_SQLCSRHLD = 0x211F
const DRDA_CP_STTSTRDEL = 0x2120
const DRDA_CP_STTDECDEL = 0x2121
const DRDA_CP_PKGDFTCST = 0x2125
const DRDA_CP_QRYBLKCTL = 0x2132
const DRDA_CP_CRRTKN = 0x2135
const DRDA_CP_PRCNAM = 0x2138
const DRDA_CP_PKGSNLST = 0x2139
const DRDA_CP_NBRROW = 0x213A
const DRDA_CP_TRGDFTRT = 0x213B
const DRDA_CP_QRYRELSCR = 0x213C
const DRDA_CP_QRYROWNBR = 0x213D
const DRDA_CP_QRYRFRTBL = 0x213E
const DRDA_CP_MAXRSLCNT = 0x2140
const DRDA_CP_MAXBLKEXT = 0x2141
const DRDA_CP_RSLSETFLG = 0x2142
const DRDA_CP_TYPSQLDA = 0x2146
const DRDA_CP_OUTOVROPT = 0x2147
const DRDA_CP_RTNEXTDTA = 0x2148
const DRDA_CP_QRYATTSCR = 0x2149
const DRDA_CP_QRYATTUPD = 0x2150
const DRDA_CP_QRYSCRORN = 0x2152
const DRDA_CP_QRYROWSNS = 0x2153
const DRDA_CP_QRYBLKRST = 0x2154
const DRDA_CP_QRYRTNDTA = 0x2155
const DRDA_CP_QRYROWSET = 0x2156
const DRDA_CP_QRYATTSNS = 0x2157
const DRDA_CP_QRYINSID = 0x215B
const DRDA_CP_QRYCLSIMP = 0x215D
const DRDA_CP_QRYCLSRLS = 0x215E
const DRDA_CP_QRYOPTVAL = 0x215F
const DRDA_CP_DIAGLVL = 0x2160
const DRDA_CP_ACCRDBRM = 0x2201
const DRDA_CP_QRYNOPRM = 0x2202
const DRDA_CP_RDBNACRM = 0x2204
const DRDA_CP_OPNQRYRM = 0x2205
const DRDA_CP_PKGBNARM = 0x2206
const DRDA_CP_RDBACCRM = 0x2207
const DRDA_CP_BGNBNDRM = 0x2208
const DRDA_CP_PKGBPARM = 0x2209
const DRDA_CP_DSCINVRM = 0x220A
const DRDA_CP_ENDQRYRM = 0x220B
const DRDA_CP_ENDUOWRM = 0x220C
const DRDA_CP_ABNUOWRM = 0x220D
const DRDA_CP_DTAMCHRM = 0x220E
const DRDA_CP_QRYPOPRM = 0x220F
const DRDA_CP_RDBNFNRM = 0x2211
const DRDA_CP_OPNQFLRM = 0x2212
const DRDA_CP_SQLERRRM = 0x2213
const DRDA_CP_RDBUPDRM = 0x2218
const DRDA_CP_RSLSETRM = 0x2219
const DRDA_CP_RDBAFLRM = 0x221A
const DRDA_CP_CMDVLTRM = 0x221D
const DRDA_CP_CMMRQSRM = 0x2225
const DRDA_CP_RDBATHRM = 0x22CB
const DRDA_CP_SQLAM = 0x2407
const DRDA_CP_SQLCARD = 0x2408
const DRDA_CP_SQLCINRD = 0x240B
const DRDA_CP_SQLRSLRD = 0x240E
const DRDA_CP_RDB = 0x240F
const DRDA_CP_FRCFIXROW = 0x2410
const DRDA_CP_SQLDARD = 0x2411
const DRDA_CP_SQLDTA = 0x2412
const DRDA_CP_SQLDTARD = 0x2413
const DRDA_CP_SQLSTT = 0x2414
const DRDA_CP_OUTOVR = 0x2415
const DRDA_CP_LMTBLKPRC = 0x2417
const DRDA_CP_FIXROWPRC = 0x2418
const DRDA_CP_SQLSTTVRB = 0x2419
const DRDA_CP_QRYDSC = 0x241A
const DRDA_CP_QRYDTA = 0x241B
const DRDA_CP_CSTMBCS = 0x2435
const DRDA_CP_SRVLST = 0x244E
const DRDA_CP_SQLATTR = 0x2450

// --- Product-specific 0xC000-0xFFFF ---
// Piggy-backed session data (product-specific)
const DRDA_CP_PBSD = 0xC000

// Isolation level as a byte (product-specific)
const DRDA_CP_PBSD_ISO = 0xC001

// Current schema as UTF8 String (product-specific)
const DRDA_CP_PBSD_SCHEMA = 0xC002

const DRDA_DSSFMT_SAME_CORR = 0x01
const DRDA_DSSFMT_CONTINUE = 0x02
const DRDA_DSSFMT_CHAINED = 0x04
const DRDA_DSSFMT_RESERVED = 0x08

const DRDA_DSSFMT_RQSDSS = 0x01
const DRDA_DSSFMT_RPYDSS = 0x02
const DRDA_DSSFMT_OBJDSS = 0x03
const DRDA_DSSFMT_CMNDSS = 0x04
const DRDA_DSSFMT_NORPYDSS = 0x05

const DRDA_TEXT_DDM = "DDM"
const DRDA_TEXT_PARAM = "Parameter"

var drda_description = map[uint16]string{
	DRDA_CP_DATA:        "Data",
	DRDA_CP_CODPNT:      "Code Point",
	DRDA_CP_FDODSC:      "FD:OCA Data Descriptor",
	DRDA_CP_TYPDEFNAM:   "Data Type Definition Name",
	DRDA_CP_TYPDEFOVR:   "TYPDEF Overrides",
	DRDA_CP_CODPNTDR:    "Code Point Data Representation",
	DRDA_CP_EXCSAT:      "Exchange Server Attributes",
	DRDA_CP_SYNCCTL:     "Sync Point Control Request",
	DRDA_CP_SYNCRSY:     "Sync Point Resync Command",
	DRDA_CP_ACCSEC:      "Access Security",
	DRDA_CP_SECCHK:      "Security Check",
	DRDA_CP_SYNCLOG:     "Sync Point Log",
	DRDA_CP_RSCTYP:      "Resource Type Information",
	DRDA_CP_RSNCOD:      "Reason Code Information",
	DRDA_CP_RSCNAM:      "Resource Name Information",
	DRDA_CP_PRDID:       "Product-Specific Identifier",
	DRDA_CP_PRCCNVCD:    "Conversation Protocol Error Code",
	DRDA_CP_VRSNAM:      "Version Name",
	DRDA_CP_SRVCLSNM:    "Server Class Name",
	DRDA_CP_SVRCOD:      "Severity Code",
	DRDA_CP_SYNERRCD:    "Syntax Error Code",
	DRDA_CP_SRVDGN:      "Server Diagnostic Information",
	DRDA_CP_SRVRLSLV:    "Server Product Release Level",
	DRDA_CP_SPVNAM:      "Supervisor Name",
	DRDA_CP_EXTNAM:      "External Name",
	DRDA_CP_SRVNAM:      "Server Name",
	DRDA_CP_SECMGRNM:    "Security Manager Name",
	DRDA_CP_DEPERRCD:    "Manager Dependency Error Code",
	DRDA_CP_CCSIDSBC:    "CCSID for Single-Byte Characters",
	DRDA_CP_CCSIDDBC:    "CCSID for Double-byte Characters",
	DRDA_CP_CCSIDMBC:    "CCSID for Mixed-byte Characters",
	DRDA_CP_USRID:       "User ID at the Target System",
	DRDA_CP_PASSWORD:    "Password",
	DRDA_CP_SECMEC:      "Security Mechanism",
	DRDA_CP_SECCHKCD:    "Security Check Code",
	DRDA_CP_SVCERRNO:    "Security Service ErrorNumber",
	DRDA_CP_SECTKN:      "Security Token",
	DRDA_CP_NEWPASSWORD: "New Password",
	DRDA_CP_MGRLVLRM:    "Manager-Level Conflict",
	DRDA_CP_MGRDEPRM:    "Manager Dependency Error",
	DRDA_CP_SECCHKRM:    "Security Check",
	DRDA_CP_CMDATHRM:    "Not Authorized to Command",
	DRDA_CP_AGNPRMRM:    "Permanent Agent Error",
	DRDA_CP_RSCLMTRM:    "Resource Limits Reached",
	DRDA_CP_PRCCNVRM:    "Conversational Protocol Error",
	DRDA_CP_CMDCMPRM:    "Command Processing Completed",
	DRDA_CP_SYNTAXRM:    "Data Stream Syntax Error",
	DRDA_CP_CMDNSPRM:    "Command Not Supported",
	DRDA_CP_PRMNSPRM:    "Parameter Not Supported",
	DRDA_CP_VALNSPRM:    "Parameter Value Not Supported",
	DRDA_CP_OBJNSPRM:    "Object Not Supported",
	DRDA_CP_CMDCHKRM:    "Command Check",
	DRDA_CP_TRGNSPRM:    "Target Not Supported",
	DRDA_CP_AGENT:       "Agent",
	DRDA_CP_MGRLVLLS:    "Manager-Level List",
	DRDA_CP_SUPERVISOR:  "Supervisor",
	DRDA_CP_SECMGR:      "Security Manager",
	DRDA_CP_EXCSATRD:    "Server Attributes Reply Data",
	DRDA_CP_CMNAPPC:     "LU 6.2 Conversational Communications Manager",
	DRDA_CP_DICTIONARY:  "Dictionary",
	DRDA_CP_MGRLVLN:     "Manager-Level Number Attribute",
	DRDA_CP_CMNTCPIP:    "TCP/IP CommunicationManager",
	DRDA_CP_FDODTA:      "FD:OCA Data",
	DRDA_CP_CMNSYNCPT:   "SNA LU 6.2 Sync Point Conversational Communications Manager",
	DRDA_CP_ACCSECRD:    "Access Security Reply Data",
	DRDA_CP_SYNCPTMGR:   "Sync Point Manager",
	DRDA_CP_RSYNCMGR:    "ResynchronizationManager",
	DRDA_CP_CCSIDMGR:    "CCSID Manager",
	DRDA_CP_MONITOR:     "Monitor Events",
	DRDA_CP_MONITORRD:   "Monitor Reply Data",
	DRDA_CP_XAMGR:       "XAManager",
	DRDA_CP_ACCRDB:      "Access RDB",
	DRDA_CP_BGNBND:      "Begin Binding a Package to an RDB",
	DRDA_CP_BNDSQLSTT:   "Bind SQL Statement to an RDB Package",
	DRDA_CP_CLSQRY:      "Close Query",
	DRDA_CP_CNTQRY:      "Continue Query",
	DRDA_CP_DRPPKG:      "Drop RDB Package",
	DRDA_CP_DSCSQLSTT:   "Describe SQL Statement",
	DRDA_CP_ENDBND:      "End Binding a Package to an RDB",
	DRDA_CP_EXCSQLIMM:   "Execute Immediate SQL Statement",
	DRDA_CP_EXCSQLSTT:   "Execute SQL Statement",
	DRDA_CP_OPNQRY:      "Open Query",
	DRDA_CP_PRPSQLSTT:   "Prepare SQL Statement",
	DRDA_CP_RDBCMM:      "RDB Commit Unit of Work",
	DRDA_CP_RDBRLLBCK:   "RDB Rollback Unit of Work",
	DRDA_CP_REBIND:      "Rebind an Existing RDB Package",
	DRDA_CP_DSCRDBTBL:   "Describe RDB Table",
	DRDA_CP_EXCSQLSET:   "Set SQL Environment",
	DRDA_CP_DSCERRCD:    "Description Error Code",
	DRDA_CP_QRYPRCTYP:   "Query Protocol Type",
	DRDA_CP_RDBINTTKN:   "RDB Interrupt Token",
	DRDA_CP_PRDDTA:      "Product-Specific Data",
	DRDA_CP_RDBCMTOK:    "RDB Commit Allowed",
	DRDA_CP_RDBCOLID:    "RDB Collection Identifier",
	DRDA_CP_PKGID:       "RDB Package Identifier",
	DRDA_CP_PKGCNSTKN:   "RDB Package Consistency Token",
	DRDA_CP_RTNSETSTT:   "Return SET Statement",
	DRDA_CP_RDBACCCL:    "RDB Access Manager Class",
	DRDA_CP_RDBNAM:      "Relational Database Name",
	DRDA_CP_OUTEXP:      "Output Expected",
	DRDA_CP_PKGNAMCT:    "RDB Package Name and Consistency Token",
	DRDA_CP_PKGNAMCSN:   "RDB Package Name, Consistency Token, and Section Number",
	DRDA_CP_QRYBLKSZ:    "Query Block Size",
	DRDA_CP_UOWDSP:      "Unit of Work Disposition",
	DRDA_CP_RTNSQLDA:    "Maximum Result Set Count",
	DRDA_CP_RDBALWUPD:   "RDB Allow Updates",
	DRDA_CP_SQLCSRHLD:   "Hold Cursor Position",
	DRDA_CP_STTSTRDEL:   "Statement String Delimiter",
	DRDA_CP_STTDECDEL:   "Statement Decimal Delimiter",
	DRDA_CP_PKGDFTCST:   "Package Default Character Subtype",
	DRDA_CP_QRYBLKCTL:   "Query Block Protocol Control",
	DRDA_CP_CRRTKN:      "Correlation Token",
	DRDA_CP_PRCNAM:      "Procedure Name",
	DRDA_CP_PKGSNLST:    "RDB Result Set Reply Message",
	DRDA_CP_NBRROW:      "Number of Fetch or Insert Rows",
	DRDA_CP_TRGDFTRT:    "Target Default Value Return",
	DRDA_CP_QRYRELSCR:   "Query Relative Scrolling Action",
	DRDA_CP_QRYROWNBR:   "Query Row Number",
	DRDA_CP_QRYRFRTBL:   "Query Refresh Answer Set Table",
	DRDA_CP_MAXRSLCNT:   "Maximum Result Set Count",
	DRDA_CP_MAXBLKEXT:   "Maximum Number of Extra Blocks",
	DRDA_CP_RSLSETFLG:   "Result Set Flags",
	DRDA_CP_TYPSQLDA:    "Type of SQL Descriptor Area",
	DRDA_CP_OUTOVROPT:   "Output Override Option",
	DRDA_CP_RTNEXTDTA:   "Return of EXTDTA Option",
	DRDA_CP_QRYATTSCR:   "Query Attribute for Scrollability",
	DRDA_CP_QRYATTUPD:   "Query Attribute for Updatability",
	DRDA_CP_QRYSCRORN:   "Query Scroll Orientation",
	DRDA_CP_QRYROWSNS:   "Query Row Sensitivity",
	DRDA_CP_QRYBLKRST:   "Query Block Reset",
	DRDA_CP_QRYRTNDTA:   "Query Returns Datat",
	DRDA_CP_QRYROWSET:   "Query Rowset Size",
	DRDA_CP_QRYATTSNS:   "Query Attribute for Sensitivity",
	DRDA_CP_QRYINSID:    "Query Instance Identifier",
	DRDA_CP_QRYCLSIMP:   "Query Close Implicit",
	DRDA_CP_QRYCLSRLS:   "Query Close Lock Release",
	DRDA_CP_QRYOPTVAL:   "QRYOPTVAL",
	DRDA_CP_DIAGLVL:     "SQL Error Diagnostic Level",
	DRDA_CP_ACCRDBRM:    "Access to RDB Completed",
	DRDA_CP_QRYNOPRM:    "Query Not Open",
	DRDA_CP_RDBNACRM:    "RDB Not Accessed",
	DRDA_CP_OPNQRYRM:    "Open Query Complete",
	DRDA_CP_PKGBNARM:    "RDB Package Binding Not Active",
	DRDA_CP_RDBACCRM:    "RDB Currently Accessed",
	DRDA_CP_BGNBNDRM:    "Begin Bind Error",
	DRDA_CP_PKGBPARM:    "RDB Package Binding Process Active",
	DRDA_CP_DSCINVRM:    "Invalid Description",
	DRDA_CP_ENDQRYRM:    "End of Query",
	DRDA_CP_ENDUOWRM:    "End Unit of Work Condition",
	DRDA_CP_ABNUOWRM:    "Abnormal End Unit ofWork Condition",
	DRDA_CP_DTAMCHRM:    "Data Descriptor Mismatch",
	DRDA_CP_QRYPOPRM:    "Query Previously Opened",
	DRDA_CP_RDBNFNRM:    "RDB Not Found",
	DRDA_CP_OPNQFLRM:    "Open Query Failure",
	DRDA_CP_SQLERRRM:    "SQL Error Condition",
	DRDA_CP_RDBUPDRM:    "RDB Update Reply Message",
	DRDA_CP_RSLSETRM:    "RDB Result Set Reply Message",
	DRDA_CP_RDBAFLRM:    "RDB Access Failed Reply Message",
	DRDA_CP_CMDVLTRM:    "Command Violation",
	DRDA_CP_CMMRQSRM:    "Commitment Request",
	DRDA_CP_RDBATHRM:    "Not Authorized to RDB",
	DRDA_CP_SQLAM:       "SQL Application Manager",
	DRDA_CP_SQLCARD:     "SQL Communications Area Reply Data",
	DRDA_CP_SQLCINRD:    "SQL Result Set Column Information Reply Data",
	DRDA_CP_SQLRSLRD:    "SQL Result Set Reply Data",
	DRDA_CP_RDB:         "Relational Database",
	DRDA_CP_FRCFIXROW:   "Force Fixed Row Query Protocol",
	DRDA_CP_SQLDARD:     "SQLDA Reply Data",
	DRDA_CP_SQLDTA:      "SQL Program Variable Data",
	DRDA_CP_SQLDTARD:    "SQL Data Reply Data",
	DRDA_CP_SQLSTT:      "SQL Statement",
	DRDA_CP_OUTOVR:      "Output Override Descriptor",
	DRDA_CP_LMTBLKPRC:   "Limited Block Protocol",
	DRDA_CP_FIXROWPRC:   "Fixed Row Query Protocol",
	DRDA_CP_SQLSTTVRB:   "SQL Statement Variable Descriptions",
	DRDA_CP_QRYDSC:      "Query Answer Set Description",
	DRDA_CP_QRYDTA:      "Query Answer Set Data",
	DRDA_CP_SQLATTR:     "SQL Statement Attributes",
	// Piggy-backed session data (product-specific)
	DRDA_CP_PBSD: "Piggy-backed session data (product-specific)",

	// Isolation level as a byte (product-specific)
	DRDA_CP_PBSD_ISO: "Isolation level as a byte (product-specific)",

	// Current schema as UTF8 String (product-specific)
	DRDA_CP_PBSD_SCHEMA: "Current schema as UTF8 String (product-specific)",
}

var drda_abbrev = map[uint16]string{
	DRDA_CP_DATA:        "DATA",
	DRDA_CP_CODPNT:      "CODPNT",
	DRDA_CP_FDODSC:      "FDODSC",
	DRDA_CP_TYPDEFNAM:   "TYPDEFNAM",
	DRDA_CP_TYPDEFOVR:   "TYPDEFOVR",
	DRDA_CP_CODPNTDR:    "CODPNTDR",
	DRDA_CP_EXCSAT:      "EXCSAT",
	DRDA_CP_SYNCCTL:     "SYNCCTL",
	DRDA_CP_SYNCRSY:     "SYNCRSY",
	DRDA_CP_ACCSEC:      "ACCSEC",
	DRDA_CP_SECCHK:      "SECCHK",
	DRDA_CP_SYNCLOG:     "SYNCLOG",
	DRDA_CP_RSCTYP:      "RSCTYP",
	DRDA_CP_RSNCOD:      "RSNCOD",
	DRDA_CP_RSCNAM:      "RSCNAM",
	DRDA_CP_PRDID:       "PRDID",
	DRDA_CP_PRCCNVCD:    "PRCCNVCD",
	DRDA_CP_VRSNAM:      "VRSNAM",
	DRDA_CP_SRVCLSNM:    "SRVCLSNM",
	DRDA_CP_SVRCOD:      "SVRCOD",
	DRDA_CP_SYNERRCD:    "SYNERRCD",
	DRDA_CP_SRVDGN:      "SRVDGN",
	DRDA_CP_SRVRLSLV:    "SRVRLSLV",
	DRDA_CP_SPVNAM:      "SPVNAM",
	DRDA_CP_EXTNAM:      "EXTNAM",
	DRDA_CP_SRVNAM:      "SRVNAM",
	DRDA_CP_SECMGRNM:    "SECMGRNM",
	DRDA_CP_DEPERRCD:    "DEPERRCD",
	DRDA_CP_CCSIDSBC:    "CCSIDSBC",
	DRDA_CP_CCSIDDBC:    "CCSIDDBC",
	DRDA_CP_CCSIDMBC:    "CCSIDMBC",
	DRDA_CP_USRID:       "USRID",
	DRDA_CP_PASSWORD:    "PASSWORD",
	DRDA_CP_SECMEC:      "SECMEC",
	DRDA_CP_SECCHKCD:    "SECCHKCD",
	DRDA_CP_SVCERRNO:    "SVCERRNO",
	DRDA_CP_SECTKN:      "SECTKN",
	DRDA_CP_NEWPASSWORD: "NEWPASSWORD",
	DRDA_CP_MGRLVLRM:    "MGRLVLRM",
	DRDA_CP_MGRDEPRM:    "MGRDEPRM",
	DRDA_CP_SECCHKRM:    "SECCHKRM",
	DRDA_CP_CMDATHRM:    "CMDATHRM",
	DRDA_CP_AGNPRMRM:    "AGNPRMRM",
	DRDA_CP_RSCLMTRM:    "RSCLMTRM",
	DRDA_CP_PRCCNVRM:    "PRCCNVRM",
	DRDA_CP_CMDCMPRM:    "CMDCMPRM",
	DRDA_CP_SYNTAXRM:    "SYNTAXRM",
	DRDA_CP_CMDNSPRM:    "CMDNSPRM",
	DRDA_CP_PRMNSPRM:    "PRMNSPRM",
	DRDA_CP_VALNSPRM:    "VALNSPRM",
	DRDA_CP_OBJNSPRM:    "OBJNSPRM",
	DRDA_CP_CMDCHKRM:    "CMDCHKRM",
	DRDA_CP_TRGNSPRM:    "TRGNSPRM",
	DRDA_CP_AGENT:       "AGENT",
	DRDA_CP_MGRLVLLS:    "MGRLVLLS",
	DRDA_CP_SUPERVISOR:  "SUPERVISOR",
	DRDA_CP_SECMGR:      "SECMGR",
	DRDA_CP_EXCSATRD:    "EXCSATRD",
	DRDA_CP_CMNAPPC:     "CMNAPPC",
	DRDA_CP_DICTIONARY:  "DICTIONARY",
	DRDA_CP_MGRLVLN:     "MGRLVLN",
	DRDA_CP_CMNTCPIP:    "CMNTCPIP",
	DRDA_CP_FDODTA:      "FDODTA",
	DRDA_CP_CMNSYNCPT:   "CMNSYNCPT",
	DRDA_CP_ACCSECRD:    "ACCSECRD",
	DRDA_CP_SYNCPTMGR:   "SYNCPTMGR",
	DRDA_CP_RSYNCMGR:    "RSYNCMGR",
	DRDA_CP_CCSIDMGR:    "CCSIDMGR",
	DRDA_CP_MONITOR:     "MONITOR",
	DRDA_CP_MONITORRD:   "MONITORRD",
	DRDA_CP_XAMGR:       "XAMGR",
	DRDA_CP_ACCRDB:      "ACCRDB",
	DRDA_CP_BGNBND:      "BGNBND",
	DRDA_CP_BNDSQLSTT:   "BNDSQLSTT",
	DRDA_CP_CLSQRY:      "CLSQRY",
	DRDA_CP_CNTQRY:      "CNTQRY",
	DRDA_CP_DRPPKG:      "DRPPKG",
	DRDA_CP_DSCSQLSTT:   "DSCSQLSTT",
	DRDA_CP_ENDBND:      "ENDBND",
	DRDA_CP_EXCSQLIMM:   "EXCSQLIMM",
	DRDA_CP_EXCSQLSTT:   "EXCSQLSTT",
	DRDA_CP_OPNQRY:      "OPNQRY",
	DRDA_CP_PRPSQLSTT:   "PRPSQLSTT",
	DRDA_CP_RDBCMM:      "RDBCMM",
	DRDA_CP_RDBRLLBCK:   "RDBRLLBCK",
	DRDA_CP_REBIND:      "REBIND",
	DRDA_CP_DSCRDBTBL:   "DSCRDBTBL",
	DRDA_CP_EXCSQLSET:   "EXCSQLSET",
	DRDA_CP_DSCERRCD:    "DSCERRCD",
	DRDA_CP_QRYPRCTYP:   "QRYPRCTYP",
	DRDA_CP_RDBINTTKN:   "RDBINTTKN",
	DRDA_CP_PRDDTA:      "PRDDTA",
	DRDA_CP_RDBCMTOK:    "RDBCMTOK",
	DRDA_CP_RDBCOLID:    "RDBCOLID",
	DRDA_CP_PKGID:       "PKGID",
	DRDA_CP_PKGCNSTKN:   "PKGCNSTKN",
	DRDA_CP_RTNSETSTT:   "RTNSETSTT",
	DRDA_CP_RDBACCCL:    "RDBACCCL",
	DRDA_CP_RDBNAM:      "RDBNAM",
	DRDA_CP_OUTEXP:      "OUTEXP",
	DRDA_CP_PKGNAMCT:    "PKGNAMCT",
	DRDA_CP_PKGNAMCSN:   "PKGNAMCSN",
	DRDA_CP_QRYBLKSZ:    "QRYBLKSZ",
	DRDA_CP_UOWDSP:      "UOWDSP",
	DRDA_CP_RTNSQLDA:    "RTNSQLDA",
	DRDA_CP_RDBALWUPD:   "RDBALWUPD",
	DRDA_CP_SQLCSRHLD:   "SQLCSRHLD",
	DRDA_CP_STTSTRDEL:   "STTSTRDEL",
	DRDA_CP_STTDECDEL:   "STTDECDEL",
	DRDA_CP_PKGDFTCST:   "PKGDFTCST",
	DRDA_CP_QRYBLKCTL:   "QRYBLKCTL",
	DRDA_CP_CRRTKN:      "CRRTKN",
	DRDA_CP_PRCNAM:      "PRCNAM",
	DRDA_CP_PKGSNLST:    "PKGSNLST",
	DRDA_CP_NBRROW:      "NBRROW",
	DRDA_CP_TRGDFTRT:    "TRGDFTRT",
	DRDA_CP_QRYRELSCR:   "QRYRELSCR",
	DRDA_CP_QRYROWNBR:   "QRYROWNBR",
	DRDA_CP_QRYRFRTBL:   "QRYRFRTBL",
	DRDA_CP_MAXRSLCNT:   "MAXRSLCNT",
	DRDA_CP_MAXBLKEXT:   "MAXBLKEXT",
	DRDA_CP_RSLSETFLG:   "RSLSETFLG",
	DRDA_CP_TYPSQLDA:    "TYPSQLDA",
	DRDA_CP_OUTOVROPT:   "OUTOVROPT",
	DRDA_CP_RTNEXTDTA:   "RTNEXTDTA",
	DRDA_CP_QRYATTSCR:   "QRYATTSCR",
	DRDA_CP_QRYATTUPD:   "QRYATTUPD",
	DRDA_CP_QRYSCRORN:   "QRYSCRORN",
	DRDA_CP_QRYROWSNS:   "QRYROWSNS",
	DRDA_CP_QRYBLKRST:   "QRYBLKRST",
	DRDA_CP_QRYRTNDTA:   "QRYRTNDTA",
	DRDA_CP_QRYROWSET:   "QRYROWSET",
	DRDA_CP_QRYATTSNS:   "QRYATTSNS",
	DRDA_CP_QRYINSID:    "QRYINSID",
	DRDA_CP_QRYCLSIMP:   "QRYCLSIMP",
	DRDA_CP_QRYCLSRLS:   "QRYCLSRLS",
	DRDA_CP_QRYOPTVAL:   "QRYOPTVAL",
	DRDA_CP_DIAGLVL:     "DIAGLVL",
	DRDA_CP_ACCRDBRM:    "ACCRDBRM",
	DRDA_CP_QRYNOPRM:    "QRYNOPRM",
	DRDA_CP_RDBNACRM:    "RDBNACRM",
	DRDA_CP_OPNQRYRM:    "OPNQRYRM",
	DRDA_CP_PKGBNARM:    "PKGBNARM",
	DRDA_CP_RDBACCRM:    "RDBACCRM",
	DRDA_CP_BGNBNDRM:    "BGNBNDRM",
	DRDA_CP_PKGBPARM:    "PKGBPARM",
	DRDA_CP_DSCINVRM:    "DSCINVRM",
	DRDA_CP_ENDQRYRM:    "ENDQRYRM",
	DRDA_CP_ENDUOWRM:    "ENDUOWRM",
	DRDA_CP_ABNUOWRM:    "ABNUOWRM",
	DRDA_CP_DTAMCHRM:    "DTAMCHRM",
	DRDA_CP_QRYPOPRM:    "QRYPOPRM",
	DRDA_CP_RDBNFNRM:    "RDBNFNRM",
	DRDA_CP_OPNQFLRM:    "OPNQFLRM",
	DRDA_CP_SQLERRRM:    "SQLERRRM",
	DRDA_CP_RDBUPDRM:    "RDBUPDRM",
	DRDA_CP_RSLSETRM:    "RSLSETRM",
	DRDA_CP_RDBAFLRM:    "RDBAFLRM",
	DRDA_CP_CMDVLTRM:    "CMDVLTRM",
	DRDA_CP_CMMRQSRM:    "CMMRQSRM",
	DRDA_CP_RDBATHRM:    "RDBATHRM",
	DRDA_CP_SQLAM:       "SQLAM",
	DRDA_CP_SQLCARD:     "SQLCARD",
	DRDA_CP_SQLCINRD:    "SQLCINRD",
	DRDA_CP_SQLRSLRD:    "SQLRSLRD",
	DRDA_CP_RDB:         "RDB",
	DRDA_CP_FRCFIXROW:   "FRCFIXROW",
	DRDA_CP_SQLDARD:     "SQLDARD",
	DRDA_CP_SQLDTA:      "SQLDTA",
	DRDA_CP_SQLDTARD:    "SQLDTARD",
	DRDA_CP_SQLSTT:      "SQLSTT",
	DRDA_CP_OUTOVR:      "OUTOVR",
	DRDA_CP_LMTBLKPRC:   "LMTBLKPRC",
	DRDA_CP_FIXROWPRC:   "FIXROWPRC",
	DRDA_CP_SQLSTTVRB:   "SQLSTTVRB",
	DRDA_CP_QRYDSC:      "QRYDSC",
	DRDA_CP_QRYDTA:      "QRYDTA",
	DRDA_CP_SQLATTR:     "SQLATTR",
	DRDA_CP_PBSD:        "PBSD",

	// Isolation level as a byte (product-specific)
	DRDA_CP_PBSD_ISO: "PBSDISO",

	// Current schema as UTF8 String (product-specific)
	DRDA_CP_PBSD_SCHEMA: "PBSD_SCHEMA",
}

var dss_abbrev = map[uint16]string{
	DRDA_DSSFMT_RQSDSS:   "RQSDSS",
	DRDA_DSSFMT_RPYDSS:   "RPYDSS",
	DRDA_DSSFMT_OBJDSS:   "OBJDSS",
	DRDA_DSSFMT_CMNDSS:   "CMNDSS",
	DRDA_DSSFMT_NORPYDSS: "NORPYDSS",
	0:                    "NULL",
}
